package main

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"pageturner-updater/ble"
	"pageturner-updater/zl"
)

func testApp(t *testing.T) *app {
	a := newApp(openVersionDB(filepath.Join(t.TempDir(), "versions.json")))
	a.appVer = "v" + mygoVersion(t) // as in the built app, so snapshots match the README screenshot
	b, err := os.ReadFile("testdata/firmware_manifest.json")
	if err != nil {
		t.Skip(err)
	}
	var m struct{ Firmwares []zl.Firmware }
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	a.catalogue = m.Firmwares
	a.devices["A"] = &device{Advertisement: ble.Advertisement{ID: "A", Name: "HBTR002-XTE3", RSSI: -42, CompanyID: 0x0171, HasCompany: true}, seen: time.Now()}
	a.devices["B"] = &device{Advertisement: ble.Advertisement{ID: "B", Name: "COROS PACE Pro", RSSI: -37}, seen: time.Now()}
	return a
}

// snapshot writes the tester's frame to $SNAPSHOT_DIR/<name>.png when set.
func snapshot(t *testing.T, tt *ui.Tester, name string) {
	dir := os.Getenv("SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tt.SetScale(2) // Retina-sharp for the README
	tt.Frame()
	img := tt.Image()
	tt.SetScale(1)
	tt.Frame()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func checked(a *app) {
	a.selectedID = "A"
	a.check = &checkResult{writeOK: true, notifyOK: true, maxWrite: 509}
}

func TestViewFlow(t *testing.T) {
	a := testApp(t)
	a.scanning = true
	tt := ui.NewTester(a.view, 720, 820)
	snapshot(t, tt, "1-scanning")
	if !tt.HasText("HBTR002-XTE3") || tt.HasText("COROS PACE Pro") {
		t.Fatalf("device list should show only supported devices: %q", tt.Texts())
	}
	if !tt.HasText("Kindle mode") || !tt.HasText("Ready · strong signal") {
		t.Errorf("mode/signal missing: %q", tt.Texts())
	}

	// Selected and checked (set directly: clicking would start a real BLE check).
	a.scanning = false
	checked(a)
	a.pickDefaultFirmware()
	tt.SetSize(720, 1500)
	tt.Frame()
	if a.fwName != "" {
		t.Errorf("2025/2026: nothing should be preselected, got %q", a.fwName)
	}
	if !tt.HasText("Choose a firmware file above.") {
		t.Errorf("install should wait for a firmware choice: %q", tt.Texts())
	}
	for _, s := range []string{"HBTR002-XTE-2025(10.30)", "HBTR002-XTE-2026(10.30)", "HBTR002-KD", "HBTR002-X-2026(11.01)", "Published 2026-10-26 · V1.0.9"} {
		if !tt.HasText(s) {
			t.Errorf("missing %q", s)
		}
	}
	if err := tt.Click("HBTR002-XTE-2026(10.30)"); err != nil {
		t.Fatal(err)
	}
	if a.fwName != "HBTR002-XTE-2026(10.30).fot" {
		t.Fatalf("fwName = %q", a.fwName)
	}
	snapshot(t, tt, "2-ready")

	if err := tt.Click("Install HBTR002-XTE-2026(10.30)"); err != nil {
		t.Fatal(err)
	}
	if !a.confirming || !tt.HasText("Install HBTR002-XTE-2026(10.30)?") {
		t.Fatalf("confirmation not shown: %q", tt.Texts())
	}
	snapshot(t, tt, "3-confirm")
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if a.confirming || a.updating {
		t.Fatal("cancel should not start the update")
	}

	a.updating, a.progress, a.status = true, 0.42, "Installing — keep the page turner close and don't press its buttons…"
	tt.Frame()
	if !tt.HasText("42%") {
		t.Errorf("progress text missing")
	}
	snapshot(t, tt, "4-installing")

	a.updating, a.failed, a.status = false, zl.ErrStillInOTA.Error(), ""
	tt.Frame()
	snapshot(t, tt, "5-failed")

	a.failed, a.done, a.progress = "", true, 1
	a.status = "Installed HBTR002-XTE-2026(10.30) in 15 s."
	a.version = &zl.Version{Code: 0x1090}
	tt.Frame()
	for _, s := range []string{"Update complete", "The page turner now reports V1.0.9.", "Open Bluetooth Settings"} {
		if !tt.HasText(s) {
			t.Errorf("success panel missing %q", s)
		}
	}
	if tt.HasText("Install HBTR002-XTE-2026(10.30)") {
		t.Error("no install button after success")
	}
	snapshot(t, tt, "6-done")
	if err := tt.Click("Done"); err != nil || a.done {
		t.Fatalf("Done: %v", err)
	}
}

func TestViewCheckFailedBlocksInstall(t *testing.T) {
	a := testApp(t)
	checked(a)
	a.check.maxWrite = 182
	a.fwName = "HBTR002-XTE-2026(10.30).fot"
	tt := ui.NewTester(a.view, 720, 1100)
	if !tt.HasText("The device check didn't pass.") {
		t.Fatalf("small MTU should block the install: %q", tt.Texts())
	}
	_ = tt.Click("Install HBTR002-XTE-2026(10.30)")
	if a.confirming {
		t.Fatal("disabled button opened the confirmation")
	}
}

func TestViewVersion(t *testing.T) {
	a := testApp(t)
	checked(a)
	tt := ui.NewTester(a.view, 720, 1600)
	if !tt.HasText("Not read yet") || !tt.HasText("Read") {
		t.Fatalf("version box missing: %q", tt.Texts())
	}
	a.version = &zl.Version{Code: 0x1090}
	a.busyUntil = time.Now().Add(30 * time.Second)
	a.fwName = "HBTR002-XTE-2026(10.30).fot"
	tt.Frame()
	for _, s := range []string{"V1.0.9", "HBTR002-XTE(10.30)", "On device"} {
		if !tt.HasText(s) {
			t.Errorf("missing %q in %q", s, tt.Texts())
		}
	}
	snapshot(t, tt, "7-version-busy")
	_ = tt.Click("Install HBTR002-XTE-2026(10.30)")
	if a.confirming {
		t.Fatal("install allowed while the device is still in update mode")
	}
	a.busyUntil = time.Time{}
	tt.Frame()
	if err := tt.Click("Install HBTR002-XTE-2026(10.30)"); err != nil || !a.confirming {
		t.Fatalf("install not allowed after the busy window: %v", err)
	}

	// Unknown version: no build claimed.
	a.confirming = false
	a.version = &zl.Version{Code: 0x1070}
	tt.Frame()
	if !tt.HasText("V1.0.7") || !tt.HasText("build not known yet") || tt.HasText("On device") {
		t.Errorf("unknown version shown wrong: %q", tt.Texts())
	}
}

func TestVersionDB(t *testing.T) {
	p := filepath.Join(t.TempDir(), "v.json")
	db := openVersionDB(p)
	if v, ok := db.get("HBTR002-XTE-2025(10.30).fot"); !ok || v.Code != 0x1090 {
		t.Fatal("seed missing for the 2025 twin")
	}
	if err := db.set("HBTR002-X-2026(11.01).fot", 0x10A0); err != nil {
		t.Fatal(err)
	}
	db2 := openVersionDB(p) // reload from disk
	if v, ok := db2.get("HBTR002-X-2025(11.01).fot"); !ok || v.Code != 0x10A0 {
		t.Fatalf("learned version not persisted/shared with the twin: %v %v", v, ok)
	}
}

func TestVariantNote(t *testing.T) {
	for name, want := range map[string]string{
		"HBTR001-E-2025(10.30).fot":       variantNotes["HBTR001-E"],
		"HBTR001-E-2026(10.30).fot":       variantNotes["HBTR001-E"],
		"HBTR002-XTE-2026(10.30).fot":     "3 modes: Kindle (2024+ Kindles, software 5.20.1+), Android (volume keys), Xteink readers",
		"HBTR002-X-2026(11.01).fot":       "Phone & tablet edition: iOS, Android, HarmonyOS; e-ink only Hanvon, Boox, Bigme, Hisense. No Kindle mode",
		"HBTR001-2025-长按小按键5秒重置蓝牙.fot":    variantNotes["HBTR001"], // plain series name, not shadowed by -E/-K/-XT
		"HBTR001-XT-2025-长按小按键5秒重置蓝牙.fot": "Xteink (阅星瞳) readers (separate hardware: only XT firmware)",
		"HBTR001-K-2026-长按小按键锁屏.fot":      "Kobo readers (Kobo/Kindle hardware: can switch to HBTR001-E)",
	} {
		f := zl.Firmware{URL: "https://x/" + name}
		if got := variantNote(f); got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
}

func TestViewOtherEdition(t *testing.T) {
	a := testApp(t)
	checked(a)
	tt := ui.NewTester(a.view, 720, 1600)
	if !tt.HasText("Different edition") || !tt.HasText("Phone & tablet edition: iOS, Android, HarmonyOS; e-ink only Hanvon, Boox, Bigme, Hisense. No Kindle mode") {
		t.Fatalf("X edition not flagged: %q", tt.Texts())
	}
	a.fwName = "HBTR002-X-2026(11.01).fot"
	tt.Frame()
	if err := tt.Click("Install HBTR002-X-2026(11.01)"); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range tt.Texts() {
		if strings.HasPrefix(s, "Warning: HBTR002-X-2026(11.01) is a different edition") {
			found = true
		}
	}
	if !found {
		t.Errorf("confirmation should warn: %q", tt.Texts())
	}
	snapshot(t, tt, "8-other-edition")
}

func TestViewPresence(t *testing.T) {
	a := testApp(t)
	a.devices["C"] = &device{Advertisement: ble.Advertisement{ID: "C", Name: "HBTR002-XT", RSSI: -50, CompanyID: 0x05AC, HasCompany: true}, seen: time.Now().Add(-20 * time.Second)}
	a.devices["D"] = &device{Advertisement: ble.Advertisement{ID: "D", Name: "HBTR001-K", Connected: true}, seen: time.Now()}
	tt := ui.NewTester(a.view, 720, 1000)
	for _, s := range []string{
		"Ready · strong signal",
		"Not visible for 20 s — asleep, switched off, or connected to a phone or reader",
		"Connected to this Mac",
		"Android mode",
	} {
		if !tt.HasText(s) {
			t.Errorf("missing %q in %q", s, tt.Texts())
		}
	}
	snapshot(t, tt, "9-presence")
	// While the app is checking the selected device it shows as busy.
	a.selectedID, a.checking = "A", true
	tt.Frame()
	if !tt.HasText("Connected — working…") {
		t.Errorf("busy state missing: %q", tt.Texts())
	}
}

func TestViewIncompatibleHardware(t *testing.T) {
	a := testApp(t)
	a.devices["K"] = &device{Advertisement: ble.Advertisement{ID: "K", Name: "HBTR001-K"}, seen: time.Now()}
	a.selectedID = "K"
	a.check = &checkResult{writeOK: true, notifyOK: true, maxWrite: 509}
	tt := ui.NewTester(a.view, 720, 2000)
	if tt.HasText("HBTR001-XT-2026-长按小按键5秒重置蓝牙") || !tt.HasText("HBTR001-E-2026(10.30)") || !tt.HasText("Different edition") {
		t.Fatalf("Xteink-hardware files must be hidden, Kindle files listed: %q", tt.Texts())
	}
	a.fwName = "HBTR001-E-2026(10.30).fot" // Kobo → Kindle: allowed with a warning
	tt.Frame()
	if err := tt.Click("Install HBTR001-E-2026(10.30)"); err != nil || !a.confirming {
		t.Fatalf("Kobo→Kindle should be allowed: %v", err)
	}
	snapshot(t, tt, "10-hbtr001")
}

func TestVersionNumbersRepeatAcrossEditions(t *testing.T) {
	a := testApp(t)
	checked(a) // HBTR002-XTE3
	a.version = &zl.Version{Code: 0x1070}
	tt := ui.NewTester(a.view, 720, 1600)
	// V1.0.7 is also HBTR002-X(11.01), but this device runs the XTE edition: don't claim X is installed.
	if !tt.HasText("build not known yet") || tt.HasText("On device") {
		t.Fatalf("XTE device with V1.0.7 must not be labelled as X: %q", tt.Texts())
	}
	// The same version on a device that runs X is the X build.
	a.devices["X"] = &device{Advertisement: ble.Advertisement{ID: "X", Name: "HBTR002-X"}, seen: time.Now()}
	a.selectedID = "X"
	tt.Frame()
	if !tt.HasText("HBTR002-X(11.01)") || !tt.HasText("On device") {
		t.Fatalf("X device with V1.0.7 should show the X build: %q", tt.Texts())
	}
}

// mygoVersion is the app version from mygo.json.
func mygoVersion(t *testing.T) string {
	b, err := os.ReadFile("mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct{ Version string }
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m.Version
}
