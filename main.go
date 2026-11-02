// Page Turner Updater — updates the firmware of 智联纵维 (diykey) Kindle page-turner remotes from a Mac.
// Pure Go: MyGo native UI + CoreBluetooth through purego (package ble), protocol in package zl.
//
//	go tool mygo dev      # run
//	go tool mygo build    # package
//
// main.go: state and actions · view.go: the interface · versions.go: learned firmware versions
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"pageturner-updater/ble"
	"pageturner-updater/zl"
)

const bluetoothSettings = "x-apple.systempreferences:com.apple.BluetoothSettings"

type device struct {
	ble.Advertisement
	seen time.Time
}

type checkResult struct {
	writeOK, notifyOK bool
	maxWrite          int
	err               error
}

type app struct {
	win      *mygo.Window
	appVer   string // "v0.1.0", or "dev"
	central  *ble.Central
	bleErr   string
	versions *versionDB

	// 1. devices
	scanning   bool // the presence scan is running
	devices    map[string]*device
	showAll    bool
	selectedID string

	// 2. check
	checking bool
	check    *checkResult

	// installed version (reading it puts the device in update mode for up to a minute)
	readingVer bool
	version    *zl.Version
	verErr     string
	busyUntil  time.Time

	// 3. firmware
	catalogue  []zl.Firmware
	catErr     string
	loadingCat bool
	fwName     string

	// 4. update
	confirming bool
	updating   bool
	progress   float64
	status     string
	done       bool
	failed     string

	logMu   sync.Mutex
	logs    []string
	logOpen bool
	logView ui.ScrollState
}

func newApp(db *versionDB) *app {
	return &app{devices: map[string]*device{}, versions: db, appVer: appVersion()}
}

func main() {
	a := newApp(openVersionDB(defaultVersionDBPath()))
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Page Turner Updater",
			Width:   720,
			Height:  820,
			Content: ui.View(a.view),
		})
		go a.start()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func (a *app) start() {
	c, err := ble.New(mygo.RunOnMain)
	if err != nil {
		a.win.Update(func() { a.bleErr = err.Error() })
		return
	}
	a.central = c
	go a.loadCatalogue()
	go a.pollConnected()
	a.watch()
}

func (a *app) logf(f string, args ...any) {
	line := time.Now().Format("15:04:05.000 ") + fmt.Sprintf(f, args...)
	a.logMu.Lock()
	a.logs = append(a.logs, line)
	if len(a.logs) > 400 {
		a.logs = a.logs[len(a.logs)-400:]
	}
	a.logMu.Unlock()
	if a.win != nil {
		a.win.Invalidate()
	}
}

// ---- derived state ----

func (a *app) selected() *device { return a.devices[a.selectedID] }

func (a *app) checkOK() bool {
	return a.check != nil && a.check.err == nil && a.check.writeOK && a.check.notifyOK && a.check.maxWrite >= zl.DataPacketSize
}

func (a *app) firmwareChoices() []zl.Firmware {
	d := a.selected()
	if d == nil || a.catalogue == nil {
		return nil
	}
	return zl.Compatible(a.catalogue, d.Name)
}

func (a *app) pickDefaultFirmware() {
	a.fwName = ""
	if ch := a.firmwareChoices(); len(ch) == 1 {
		a.fwName = ch[0].FileName()
	}
}

func (a *app) chosenFirmware() (zl.Firmware, bool) {
	d := a.selected()
	if d == nil {
		return zl.Firmware{}, false
	}
	for _, f := range zl.Candidates(a.catalogue, d.Name) {
		if f.FileName() == a.fwName {
			return f, true
		}
	}
	return zl.Firmware{}, false
}

// onDevice reports whether file f is the build currently on the device (by learned versions).
func (a *app) onDevice(f zl.Firmware) bool {
	if a.version == nil {
		return false
	}
	d := a.selected()
	if d == nil || zl.FitFor(f, d.Name) != zl.FitSame {
		return false // version numbers repeat across editions
	}
	v, ok := a.versions.get(f.FileName())
	return ok && v.Code == a.version.Code
}

// busyLeft is how long the device may still be in update mode after a version read.
func (a *app) busyLeft() time.Duration {
	if d := time.Until(a.busyUntil); d > 0 {
		return d
	}
	return 0
}

// step is the first unfinished step (1-4; 5 = all done), for the step indicator.
func (a *app) step() int {
	switch {
	case a.selected() == nil:
		return 1
	case !a.checkOK():
		return 2
	case a.done:
		return 5
	case a.fwName == "":
		return 3
	}
	return 4
}

// ---- actions (run on goroutines unless noted) ----

func (a *app) loadCatalogue() {
	a.win.Update(func() { a.loadingCat, a.catErr = true, "" })
	fw, err := zl.FetchManifest()
	a.win.Update(func() {
		a.loadingCat = false
		if err != nil {
			a.catErr = err.Error()
			return
		}
		a.catalogue = fw
		a.pickDefaultFirmware()
	})
	if err == nil {
		a.logf("firmware list: %d files", len(fw))
	}
}

var watchedServices = []string{zl.ServiceUUID, "1812", "180F", "180A"}

// watch scans for the app's lifetime and records when each device was last heard, so the list can tell
// "advertising now" from "gone" (asleep, switched off or connected to another host — indistinguishable over BLE).
func (a *app) watch() {
	a.win.Update(func() { a.scanning, a.bleErr = true, "" })
	a.logf("listening for page turners…")
	var lastDraw time.Time
	err := a.central.Scan(context.Background(), watchedServices, func(ad ble.Advertisement) {
		// main thread: same thread as the view
		if ad.Name == "" {
			return
		}
		d, ok := a.devices[ad.ID]
		if !ok {
			d = &device{}
			a.devices[ad.ID] = d
		}
		if ad.Connected && ok {
			ad.RSSI = d.RSSI
		}
		d.Advertisement, d.seen = ad, time.Now()
		if !ok || time.Since(lastDraw) > 500*time.Millisecond { // advertisements arrive many times a second
			lastDraw = time.Now()
			a.win.Invalidate()
		}
	})
	a.win.Update(func() {
		a.scanning = false
		if err != nil {
			a.bleErr = err.Error()
		}
	})
}

// pollConnected refreshes which devices are connected to this Mac (they don't advertise while connected).
func (a *app) pollConnected() {
	for range time.Tick(2 * time.Second) {
		list := a.central.Connected(watchedServices)
		a.win.Update(func() {
			now := map[string]ble.Advertisement{}
			for _, ad := range list {
				now[ad.ID] = ad
			}
			for id, d := range a.devices {
				if ad, ok := now[id]; ok {
					d.Connected, d.seen = true, time.Now()
					if d.Name == "" {
						d.Name = ad.Name
					}
				} else if d.Connected {
					d.Connected = false
				}
			}
			for id, ad := range now {
				if _, ok := a.devices[id]; !ok && ad.Name != "" {
					a.devices[id] = &device{Advertisement: ad, seen: time.Now()}
				}
			}
		})
	}
}

// presence describes whether d can be reached right now.
type presence int

const (
	presReady presence = iota // advertising
	presMac                   // connected to this Mac
	presBusy                  // this app is talking to it
	presGone                  // not heard recently
)

const goneAfter = 8 * time.Second // some devices advertise only every few seconds

func (a *app) presenceOf(d *device) (presence, time.Duration) {
	if d.ID == a.selectedID && (a.checking || a.updating || a.readingVer) {
		return presBusy, 0
	}
	if d.Connected {
		return presMac, 0
	}
	age := time.Since(d.seen)
	if d.seen.IsZero() || age > goneAfter {
		return presGone, age
	}
	return presReady, age
}

// selectDevice runs on the main thread (from the view).
func (a *app) selectDevice(d *device) {
	if a.updating {
		return
	}
	a.selectedID, a.check, a.done, a.failed, a.status, a.progress = d.ID, nil, false, "", "", 0
	a.version, a.verErr = nil, ""
	a.pickDefaultFirmware()
	adv := d.Advertisement
	a.checking = true
	go func() {
		r := a.runCheck(adv)
		a.win.Update(func() {
			if a.selectedID == adv.ID {
				a.check, a.checking = r, false
			}
		})
	}()
}

// retryCheck looks for the device again by name (its advertisement may be stale), then checks it.
func (a *app) retryCheck(name string) {
	a.checking, a.check = true, nil
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		ad, err := a.findAgain(ctx, name)
		if err != nil {
			a.win.Update(func() {
				a.checking = false
				a.check = &checkResult{err: fmt.Errorf("%s isn't advertising. Press a button on it to wake it up, and make sure no phone or reader is connected to it", name)}
			})
			return
		}
		a.win.Update(func() {
			d, ok := a.devices[ad.ID]
			if !ok {
				d = &device{}
				a.devices[ad.ID] = d
			}
			d.Advertisement, d.seen = ad, time.Now()
			a.checking = false
			a.selectDevice(d)
		})
	}()
}

// runCheck connects read-only and verifies the OTA service, characteristics and MTU.
func (a *app) runCheck(adv ble.Advertisement) *checkResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	a.logf("checking %s (read-only)", adv.Name)
	conn, err := a.central.Connect(ctx, adv)
	if err != nil {
		a.logf("connect failed: %v", err)
		if strings.Contains(err.Error(), "timed out") {
			err = fmt.Errorf("it didn't answer. It may have gone to sleep, or a phone or reader nearby connected to it first. Press a button on it, turn off Bluetooth on your other devices, then Retry")
		}
		return &checkResult{err: err}
	}
	defer conn.Close()
	props, err := conn.Discover(ctx, zl.ServiceUUID, zl.WriteUUID, zl.NotifyUUID)
	if err != nil {
		a.logf("discover failed: %v", err)
		return &checkResult{err: fmt.Errorf("%w — this device doesn't offer the 智联 update service", err)}
	}
	r := &checkResult{
		writeOK:  props[zl.WriteUUID]&ble.PropWriteWithoutResponse != 0,
		notifyOK: props[zl.NotifyUUID]&ble.PropNotify != 0,
		maxWrite: conn.MaxWrite(true),
	}
	a.logf("FF15 %#x, FF14 %#x, max write %d", props[zl.WriteUUID], props[zl.NotifyUUID], r.maxWrite)
	return r
}

// otaLink adapts a BLE connection to zl.Link.
type otaLink struct {
	conn    *ble.Conn
	replies <-chan []byte
}

func (l otaLink) Write(ctx context.Context, p []byte) error {
	return l.conn.WriteNoResponse(ctx, zl.WriteUUID, p)
}

func (l otaLink) Recv(ctx context.Context, timeout time.Duration) ([]byte, error) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case r := <-l.replies:
		return r, nil
	case <-l.conn.Done():
		return nil, errors.New("the page turner disconnected")
	case <-t.C:
		return nil, zl.ErrTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// openOTA connects and subscribes to the update channel.
func (a *app) openOTA(ctx context.Context, adv ble.Advertisement) (*ble.Conn, otaLink, error) {
	conn, err := a.central.Connect(ctx, adv)
	if err != nil {
		return nil, otaLink{}, err
	}
	if _, err := conn.Discover(ctx, zl.ServiceUUID, zl.WriteUUID, zl.NotifyUUID); err != nil {
		conn.Close()
		return nil, otaLink{}, err
	}
	replies, err := conn.Subscribe(ctx, zl.NotifyUUID)
	if err != nil {
		conn.Close()
		return nil, otaLink{}, err
	}
	return conn, otaLink{conn, replies}, nil
}

// queryVersion reads INFO_DEV_VER (Cmd1+Cmd2 only); the device then stays in update mode for up to a minute.
func (a *app) queryVersion(ctx context.Context, adv ble.Advertisement) (zl.Version, error) {
	conn, link, err := a.openOTA(ctx, adv)
	if err != nil {
		return zl.Version{}, err
	}
	defer conn.Close()
	return zl.ReadVersion(ctx, link)
}

// setVersion records a version read (main thread).
func (a *app) setVersion(v *zl.Version, err error) {
	a.readingVer = false
	if v != nil {
		a.version, a.verErr = v, ""
		a.busyUntil = time.Now().Add(zl.BusyAfterQuery)
	}
	if err != nil {
		a.verErr = err.Error()
		if errors.Is(err, zl.ErrStillInOTA) {
			a.busyUntil = time.Now().Add(zl.BusyAfterQuery)
		}
	}
}

// readVersion is the "Read version" button.
func (a *app) readVersion(adv ble.Advertisement) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	a.logf("reading installed version")
	v, err := a.queryVersion(ctx, adv)
	if err != nil {
		a.logf("version: %v", err)
		a.win.Update(func() { a.setVersion(nil, err) })
		return
	}
	a.logf("installed version %s (0x%04X), params %v", v, v.Code, v.Params)
	a.win.Update(func() { a.setVersion(&v, nil) })
}

// findAgain looks for the device after it restarted. A new firmware may come back under a different name and
// address (HBTR002-XTE3 → HBTR002-X after installing the X edition), so any device of the same series counts.
func (a *app) findAgain(ctx context.Context, name string) (ble.Advertisement, error) {
	series := strings.ToUpper(zl.Series(name))
	match := func(n string) bool {
		return n == name || (series != "" && strings.ToUpper(zl.Series(n)) == series)
	}
	found := make(chan ble.Advertisement, 1)
	sctx, stop := context.WithCancel(ctx)
	defer stop()
	go func() {
		_ = a.central.Scan(sctx, watchedServices, func(ad ble.Advertisement) {
			if match(ad.Name) {
				select {
				case found <- ad:
				default:
				}
			}
		})
	}()
	select {
	case ad := <-found:
		stop()
		time.Sleep(300 * time.Millisecond) // let the scan stop
		return ad, nil
	case <-ctx.Done():
		return ble.Advertisement{}, errors.New("the page turner didn't come back")
	}
}

func (a *app) runUpdate(adv ble.Advertisement, fw zl.Firmware) {
	fail := func(err error) {
		a.logf("FAILED: %v", err)
		a.win.Update(func() { a.updating, a.failed, a.status = false, err.Error(), "" })
	}
	status := func(s string) { a.win.Update(func() { a.status = s }) }

	status("Downloading firmware…")
	file, note, err := zl.Fetch(fw)
	if err != nil {
		fail(err)
		return
	}
	a.logf("%s: %d bytes, %s", fw.FileName(), len(file), note)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	status("Connecting…")
	conn, link, err := a.openOTA(ctx, adv)
	if err != nil {
		fail(err)
		return
	}
	status("Installing — keep the page turner close and don't press its buttons…")
	start := time.Now()
	err = zl.Flash(ctx, link, file, zl.Options{
		Log: func(s string) { a.logf("%s", s) },
		Progress: func(done, total int) {
			a.win.Update(func() { a.progress = float64(done) / float64(total) })
		},
	})
	conn.Close()
	if err != nil {
		fail(err)
		return
	}
	took := time.Since(start)
	a.logf("installed in %.1f s", took.Seconds())

	// Learn this build's version: wait for the restart, find the device again, read INFO_DEV_VER.
	a.win.Update(func() { a.progress, a.status = 1, "Installed. Waiting for the page turner to restart…" })
	time.Sleep(6 * time.Second)
	vctx, vcancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer vcancel()
	var learned *zl.Version
	again, err := a.findAgain(vctx, adv.Name)
	if err != nil {
		a.logf("version after update: %v", err)
	} else if v, err := a.queryVersion(vctx, again); err != nil {
		a.logf("version after update: %v", err)
	} else {
		learned = &v
		if err := a.versions.set(fw.FileName(), v.Code); err != nil {
			a.logf("saving version: %v", err)
		}
		a.logf("%s reports %s (0x%04X), remembered", zl.BuildKey(fw.FileName()), v, v.Code)
	}
	a.win.Update(func() {
		a.updating, a.done = false, true
		a.status = fmt.Sprintf("Installed %s in %.0f s.", fw.DisplayName(), took.Seconds())
		if again.ID != "" && again.ID != a.selectedID {
			// it came back as a different BLE identity (new name/address): follow it
			d, ok := a.devices[again.ID]
			if !ok {
				d = &device{}
				a.devices[again.ID] = d
			}
			d.Advertisement, d.seen = again, time.Now()
			a.selectedID = again.ID
			a.status += fmt.Sprintf(" It now appears as %s.", again.Name)
		}
		if learned != nil {
			a.setVersion(learned, nil)
		}
	})
}
