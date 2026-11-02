//go:build darwin

// Package ble is a small CoreBluetooth central for macOS in pure Go (purego + Objective-C runtime, no cgo).
//
// CoreBluetooth runs on the main dispatch queue: every call into it is made on the main thread through
// the runOnMain function given to New (mygo.RunOnMain in the app), and delegate callbacks arrive there too.
// The exported methods block and must be called from goroutines, never from the main thread.
package ble

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// ---- Objective-C plumbing ----

type id = objc.ID

var (
	loadOnce                                  sync.Once
	loadErr                                   error
	selCache                                  sync.Map
	keyLocalName, keyMfgData, keyServiceUUIDs id
	keyAllowDuplicates                        id
)

func sel(name string) objc.SEL {
	if s, ok := selCache.Load(name); ok {
		return s.(objc.SEL)
	}
	s := objc.RegisterName(name)
	selCache.Store(name, s)
	return s
}

func cls(name string) id { return id(objc.GetClass(name)) }

func send(o id, s string, args ...any) id { return o.Send(sel(s), args...) }

func load() error {
	loadOnce.Do(func() {
		if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_GLOBAL|purego.RTLD_NOW); err != nil {
			loadErr = err
			return
		}
		cb, err := purego.Dlopen("/System/Library/Frameworks/CoreBluetooth.framework/CoreBluetooth", purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if err != nil {
			loadErr = err
			return
		}
		constant := func(name string) id {
			p, err := purego.Dlsym(cb, name)
			if err != nil || p == 0 {
				return 0
			}
			// p is the address of an NSString* global inside CoreBluetooth (not Go memory).
			return **(**id)(unsafe.Pointer(&p))
		}
		keyLocalName = constant("CBAdvertisementDataLocalNameKey")
		keyMfgData = constant("CBAdvertisementDataManufacturerDataKey")
		keyServiceUUIDs = constant("CBAdvertisementDataServiceUUIDsKey")
		keyAllowDuplicates = constant("CBCentralManagerScanOptionAllowDuplicatesKey")
		loadErr = registerDelegate()
	})
	return loadErr
}

func goString(s id) string {
	if s == 0 {
		return ""
	}
	p := objc.Send[*byte](s, sel("UTF8String"))
	if p == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(p, n))
}

func nsString(s string) id { return send(cls("NSString"), "stringWithUTF8String:", s) }

func goBytes(d id) []byte {
	if d == 0 {
		return nil
	}
	n := objc.Send[uint](d, sel("length"))
	if n == 0 {
		return []byte{}
	}
	p := objc.Send[unsafe.Pointer](d, sel("bytes"))
	return append([]byte(nil), unsafe.Slice((*byte)(p), n)...)
}

func nsData(b []byte) id {
	if len(b) == 0 {
		return send(cls("NSData"), "data")
	}
	return send(cls("NSData"), "dataWithBytes:length:", unsafe.Pointer(&b[0]), uint(len(b)))
}

func cbuuid(s string) id { return send(cls("CBUUID"), "UUIDWithString:", nsString(s)) }

func uuidString(u id) string { return strings.ToUpper(goString(send(u, "UUIDString"))) }

func nsArray(items ...id) id {
	a := send(cls("NSMutableArray"), "array")
	for _, it := range items {
		send(a, "addObject:", it)
	}
	return a
}

func arrayItems(a id) []id {
	if a == 0 {
		return nil
	}
	n := objc.Send[uint](a, sel("count"))
	out := make([]id, 0, n)
	for i := uint(0); i < n; i++ {
		out = append(out, send(a, "objectAtIndex:", i))
	}
	return out
}

func nsErr(e id) error {
	if e == 0 {
		return nil
	}
	return errors.New(goString(send(e, "localizedDescription")))
}

func peripheralID(p id) string {
	return strings.ToUpper(goString(send(send(p, "identifier"), "UUIDString")))
}

// ---- delegate ----

var (
	centralsMu sync.Mutex
	centrals   = map[id]*Central{} // delegate object → central
)

func centralFor(self id) *Central {
	centralsMu.Lock()
	defer centralsMu.Unlock()
	return centrals[self]
}

func registerDelegate() error {
	var protos []*objc.Protocol
	for _, n := range []string{"CBCentralManagerDelegate", "CBPeripheralDelegate"} {
		if p := objc.GetProtocol(n); p != nil {
			protos = append(protos, p)
		}
	}
	m := func(s string, fn any) objc.MethodDef { return objc.MethodDef{Cmd: sel(s), Fn: fn} }
	_, err := objc.RegisterClass("KPTBLEDelegate", objc.GetClass("NSObject"), protos, nil, []objc.MethodDef{
		m("centralManagerDidUpdateState:", func(self id, _ objc.SEL, mgr id) {
			if c := centralFor(self); c != nil {
				c.setState(State(objc.Send[int](mgr, sel("state"))))
			}
		}),
		m("centralManager:didDiscoverPeripheral:advertisementData:RSSI:", func(self id, _ objc.SEL, _ id, p id, adv id, rssi id) {
			if c := centralFor(self); c != nil {
				c.discovered(p, adv, int(objc.Send[int32](rssi, sel("intValue"))))
			}
		}),
		m("centralManager:didConnectPeripheral:", func(self id, _ objc.SEL, _ id, p id) {
			if c := centralFor(self); c != nil {
				c.peer(p).result(&c.peer(p).connect, nil)
			}
		}),
		m("centralManager:didFailToConnectPeripheral:error:", func(self id, _ objc.SEL, _ id, p id, e id) {
			if c := centralFor(self); c != nil {
				err := nsErr(e)
				if err == nil {
					err = errors.New("connection failed")
				}
				c.peer(p).result(&c.peer(p).connect, err)
			}
		}),
		m("centralManager:didDisconnectPeripheral:error:", func(self id, _ objc.SEL, _ id, p id, e id) {
			if c := centralFor(self); c != nil {
				c.peer(p).disconnected(nsErr(e))
			}
		}),
		m("peripheral:didDiscoverServices:", func(self id, _ objc.SEL, p id, e id) {
			if c := centralFor(self); c != nil {
				c.peer(p).result(&c.peer(p).services, nsErr(e))
			}
		}),
		m("peripheral:didDiscoverCharacteristicsForService:error:", func(self id, _ objc.SEL, p id, _ id, e id) {
			if c := centralFor(self); c != nil {
				c.peer(p).result(&c.peer(p).chars, nsErr(e))
			}
		}),
		m("peripheral:didUpdateNotificationStateForCharacteristic:error:", func(self id, _ objc.SEL, p id, _ id, e id) {
			if c := centralFor(self); c != nil {
				c.peer(p).result(&c.peer(p).notify, nsErr(e))
			}
		}),
		m("peripheral:didUpdateValueForCharacteristic:error:", func(self id, _ objc.SEL, p id, ch id, e id) {
			if c := centralFor(self); c != nil {
				c.peer(p).value(uuidString(send(ch, "UUID")), goBytes(send(ch, "value")), nsErr(e))
			}
		}),
		m("peripheralIsReadyToSendWriteWithoutResponse:", func(self id, _ objc.SEL, p id) {
			if c := centralFor(self); c != nil {
				select {
				case c.peer(p).ready <- struct{}{}:
				default:
				}
			}
		}),
	})
	return err
}

// ---- public API ----

// State mirrors CBManagerState.
type State int

const (
	StateUnknown State = iota
	StateResetting
	StateUnsupported
	StateUnauthorized
	StatePoweredOff
	StatePoweredOn
)

func (s State) Error() error {
	switch s {
	case StatePoweredOn:
		return nil
	case StateUnauthorized:
		return errors.New("Bluetooth access is not allowed — enable this app in System Settings › Privacy & Security › Bluetooth")
	case StatePoweredOff:
		return errors.New("Bluetooth is turned off")
	case StateUnsupported:
		return errors.New("this Mac does not support Bluetooth Low Energy")
	}
	return errors.New("Bluetooth is not ready yet")
}

// Advertisement is one scan result.
type Advertisement struct {
	ID         string // CoreBluetooth identifier (changes with the device's address)
	Name       string
	RSSI       int
	CompanyID  uint16 // from manufacturer data, if any
	HasCompany bool
	Services   []string
	Connected  bool // already connected to this Mac (bonded), not advertising
	p          id
}

type Central struct {
	runOnMain func(func())
	mgr, del  id

	mu      sync.Mutex
	state   State
	stateCh chan struct{} // closed and replaced on every state change
	peers   map[string]*peer

	listeners map[int]func(Advertisement) // active Scan callers; the radio scans while there is one
	nextL     int
}

// New creates the central manager. runOnMain must run fn on the main thread (e.g. mygo.RunOnMain).
func New(runOnMain func(func())) (*Central, error) {
	if err := load(); err != nil {
		return nil, fmt.Errorf("CoreBluetooth: %w", err)
	}
	c := &Central{runOnMain: runOnMain, stateCh: make(chan struct{}), peers: map[string]*peer{}, listeners: map[int]func(Advertisement){}}
	c.main(func() {
		c.del = send(cls("KPTBLEDelegate"), "new")
		centralsMu.Lock()
		centrals[c.del] = c
		centralsMu.Unlock()
		c.mgr = send(send(cls("CBCentralManager"), "alloc"), "initWithDelegate:queue:options:", c.del, id(0), id(0))
	})
	return c, nil
}

// main runs fn on the main thread and waits for it.
func (c *Central) main(fn func()) {
	done := make(chan struct{})
	c.runOnMain(func() {
		defer close(done)
		fn()
	})
	<-done
}

func (c *Central) setState(s State) {
	c.mu.Lock()
	c.state = s
	close(c.stateCh)
	c.stateCh = make(chan struct{})
	c.mu.Unlock()
}

// WaitReady waits until Bluetooth is powered on, or returns why it isn't.
func (c *Central) WaitReady(ctx context.Context) error {
	for {
		c.mu.Lock()
		s, ch := c.state, c.stateCh
		c.mu.Unlock()
		if s == StatePoweredOn {
			return nil
		}
		if s == StateUnauthorized || s == StatePoweredOff || s == StateUnsupported {
			return s.Error()
		}
		select {
		case <-ch:
		case <-ctx.Done():
			return s.Error()
		}
	}
}

// Scan reports every advertisement packet (duplicates included, so callers can tell a device is still
// advertising) to onAdv until ctx is done. Several Scan calls may run at once; they share one radio scan.
// Devices already connected to this Mac with one of the given services are reported first (Connected=true).
// onAdv runs on the main thread and must return quickly.
func (c *Central) Scan(ctx context.Context, services []string, onAdv func(Advertisement)) error {
	if err := c.WaitReady(ctx); err != nil {
		return err
	}
	var key int
	c.main(func() {
		for _, a := range c.connected(services) {
			onAdv(a)
		}
		c.mu.Lock()
		key = c.nextL
		c.nextL++
		c.listeners[key] = onAdv
		first := len(c.listeners) == 1
		c.mu.Unlock()
		if first {
			opts := id(0)
			if keyAllowDuplicates != 0 {
				opts = send(cls("NSDictionary"), "dictionaryWithObject:forKey:", send(cls("NSNumber"), "numberWithBool:", true), keyAllowDuplicates)
			}
			send(c.mgr, "scanForPeripheralsWithServices:options:", id(0), opts)
		}
	})
	<-ctx.Done()
	c.main(func() {
		c.mu.Lock()
		delete(c.listeners, key)
		last := len(c.listeners) == 0
		c.mu.Unlock()
		if last {
			send(c.mgr, "stopScan")
		}
	})
	return nil
}

// connected lists devices with one of services that are connected to this Mac (they don't advertise). Main thread.
func (c *Central) connected(services []string) []Advertisement {
	if len(services) == 0 {
		return nil
	}
	var uu []id
	for _, s := range services {
		uu = append(uu, cbuuid(s))
	}
	var out []Advertisement
	for _, p := range arrayItems(send(c.mgr, "retrieveConnectedPeripheralsWithServices:", nsArray(uu...))) {
		pr := c.peer(p)
		out = append(out, Advertisement{ID: pr.id, Name: goString(send(p, "name")), Connected: true, p: pr.obj})
	}
	return out
}

// Connected lists devices with one of services that are currently connected to this Mac.
func (c *Central) Connected(services []string) []Advertisement {
	var out []Advertisement
	c.main(func() { out = c.connected(services) })
	return out
}

func (c *Central) discovered(p, adv id, rssi int) {
	c.mu.Lock()
	ls := make([]func(Advertisement), 0, len(c.listeners))
	for _, l := range c.listeners {
		ls = append(ls, l)
	}
	c.mu.Unlock()
	if len(ls) == 0 {
		return
	}
	pr := c.peer(p)
	a := Advertisement{ID: pr.id, RSSI: rssi, p: pr.obj}
	if keyLocalName != 0 {
		a.Name = goString(send(adv, "objectForKey:", keyLocalName))
	}
	if a.Name == "" {
		a.Name = goString(send(p, "name"))
	}
	if keyMfgData != 0 {
		if d := goBytes(send(adv, "objectForKey:", keyMfgData)); len(d) >= 2 {
			a.CompanyID, a.HasCompany = uint16(d[0])|uint16(d[1])<<8, true
		}
	}
	if keyServiceUUIDs != 0 {
		for _, u := range arrayItems(send(adv, "objectForKey:", keyServiceUUIDs)) {
			a.Services = append(a.Services, uuidString(u))
		}
	}
	for _, l := range ls {
		l(a)
	}
}

// ---- per-peripheral state ----

type opResult struct{ err error }

type peer struct {
	id       string
	obj      id // retained CBPeripheral
	connect  chan opResult
	services chan opResult
	chars    chan opResult
	notify   chan opResult
	ready    chan struct{}

	mu     sync.Mutex
	values chan []byte // notifications of the subscribed characteristic
	watch  string
	gone   chan struct{} // closed on disconnect
	err    error
}

// peer returns the state for p, creating (and retaining p) on first sight. Main thread.
func (c *Central) peer(p id) *peer {
	key := peripheralID(p)
	c.mu.Lock()
	defer c.mu.Unlock()
	if pr, ok := c.peers[key]; ok {
		return pr
	}
	send(p, "retain")
	send(p, "setDelegate:", c.del)
	pr := &peer{id: key, obj: p, ready: make(chan struct{}, 1), gone: make(chan struct{})}
	pr.reset()
	c.peers[key] = pr
	return pr
}

func (p *peer) reset() {
	p.connect = make(chan opResult, 1)
	p.services = make(chan opResult, 1)
	p.chars = make(chan opResult, 1)
	p.notify = make(chan opResult, 1)
}

func (p *peer) result(ch *chan opResult, err error) {
	select {
	case *ch <- opResult{err}:
	default:
	}
}

func (p *peer) disconnected(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err == nil {
		err = errors.New("disconnected")
	}
	p.err = err
	select {
	case <-p.gone:
	default:
		close(p.gone)
	}
	p.result(&p.connect, err)
}

func (p *peer) value(uuid string, b []byte, err error) {
	p.mu.Lock()
	ch, w := p.values, p.watch
	p.mu.Unlock()
	if ch == nil || err != nil || !strings.HasSuffix(uuid, w) {
		return
	}
	select {
	case ch <- b:
	default: // the protocol never has more than a few replies in flight
	}
}

func wait(ctx context.Context, ch chan opResult, gone chan struct{}, timeout time.Duration, what string) error {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case r := <-ch:
		return r.err
	case <-gone:
		return errors.New("device disconnected during " + what)
	case <-t.C:
		return fmt.Errorf("timed out: %s", what)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ---- connection ----

type Conn struct {
	c     *Central
	p     *peer
	Name  string
	chars map[string]id
	svc   id
}

// Connect connects to a scanned device (or reuses the system's connection if it is already connected).
func (c *Central) Connect(ctx context.Context, a Advertisement) (*Conn, error) {
	var pr *peer
	c.main(func() {
		pr = c.peer(a.p)
		pr.mu.Lock()
		pr.reset()
		pr.gone = make(chan struct{})
		pr.err = nil
		pr.mu.Unlock()
		send(c.mgr, "connectPeripheral:options:", pr.obj, id(0))
	})
	if err := wait(ctx, pr.connect, make(chan struct{}), 20*time.Second, "connecting"); err != nil {
		c.main(func() { send(c.mgr, "cancelPeripheralConnection:", pr.obj) })
		return nil, err
	}
	return &Conn{c: c, p: pr, Name: a.Name, chars: map[string]id{}}, nil
}

// Done is closed when the device disconnects.
func (k *Conn) Done() <-chan struct{} { return k.p.gone }

// Discover finds service and its characteristics. It returns which of chars are present with their
// properties (CBCharacteristicProperties bits).
func (k *Conn) Discover(ctx context.Context, service string, chars ...string) (map[string]uint, error) {
	var started bool
	k.c.main(func() {
		send(k.p.obj, "discoverServices:", nsArray(cbuuid(service)))
		started = true
	})
	if !started {
		return nil, errors.New("not connected")
	}
	if err := wait(ctx, k.p.services, k.p.gone, 10*time.Second, "discovering services"); err != nil {
		return nil, err
	}
	k.c.main(func() {
		for _, s := range arrayItems(send(k.p.obj, "services")) {
			if strings.HasSuffix(uuidString(send(s, "UUID")), strings.ToUpper(service)) {
				k.svc = s
			}
		}
		if k.svc != 0 {
			var uu []id
			for _, ch := range chars {
				uu = append(uu, cbuuid(ch))
			}
			send(k.p.obj, "discoverCharacteristics:forService:", nsArray(uu...), k.svc)
		}
	})
	if k.svc == 0 {
		return nil, fmt.Errorf("service %s not found", service)
	}
	if err := wait(ctx, k.p.chars, k.p.gone, 10*time.Second, "discovering characteristics"); err != nil {
		return nil, err
	}
	found := map[string]uint{}
	k.c.main(func() {
		for _, ch := range arrayItems(send(k.svc, "characteristics")) {
			u := uuidString(send(ch, "UUID"))
			for _, want := range chars {
				if strings.HasSuffix(u, strings.ToUpper(want)) {
					k.chars[strings.ToUpper(want)] = ch
					found[strings.ToUpper(want)] = objc.Send[uint](ch, sel("properties"))
				}
			}
		}
	})
	return found, nil
}

// Characteristic property bits (CBCharacteristicProperties).
const (
	PropRead                 = 0x02
	PropWriteWithoutResponse = 0x04
	PropWrite                = 0x08
	PropNotify               = 0x10
)

// MaxWrite is maximumWriteValueLengthForType: (withoutResponse=true → type 1).
func (k *Conn) MaxWrite(withoutResponse bool) int {
	t := 0
	if withoutResponse {
		t = 1
	}
	var n uint
	k.c.main(func() { n = objc.Send[uint](k.p.obj, sel("maximumWriteValueLengthForType:"), t) })
	return int(n)
}

// Subscribe enables notifications on char; values then arrive from Notifications.
func (k *Conn) Subscribe(ctx context.Context, char string) (<-chan []byte, error) {
	ch := k.chars[strings.ToUpper(char)]
	if ch == 0 {
		return nil, fmt.Errorf("characteristic %s not discovered", char)
	}
	values := make(chan []byte, 64)
	k.p.mu.Lock()
	k.p.values, k.p.watch = values, strings.ToUpper(char)
	k.p.mu.Unlock()
	k.c.main(func() { send(k.p.obj, "setNotifyValue:forCharacteristic:", true, ch) })
	if err := wait(ctx, k.p.notify, k.p.gone, 10*time.Second, "enabling notifications"); err != nil {
		return nil, err
	}
	return values, nil
}

// WriteNoResponse writes without response, waiting for CoreBluetooth's flow control.
func (k *Conn) WriteNoResponse(ctx context.Context, char string, b []byte) error {
	ch := k.chars[strings.ToUpper(char)]
	if ch == 0 {
		return fmt.Errorf("characteristic %s not discovered", char)
	}
	for {
		var ok bool
		k.c.main(func() {
			ok = objc.Send[bool](k.p.obj, sel("canSendWriteWithoutResponse"))
			if ok {
				send(k.p.obj, "writeValue:forCharacteristic:type:", nsData(b), ch, 1)
			}
		})
		if ok {
			return nil
		}
		select {
		case <-k.p.ready:
		case <-time.After(200 * time.Millisecond): // poll in case the ready callback was missed
		case <-k.p.gone:
			return errors.New("device disconnected")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Close disconnects (macOS may keep its own connection to a bonded HID device).
func (k *Conn) Close() {
	k.p.mu.Lock()
	k.p.values = nil
	k.p.mu.Unlock()
	k.c.main(func() { send(k.c.mgr, "cancelPeripheralConnection:", k.p.obj) })
}
