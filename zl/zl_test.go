package zl

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func fakeFirmware(n int) []byte {
	b := make([]byte, n)
	copy(b, "POT\x00")
	b[4], b[5], b[6] = byte(n-8), byte((n-8)>>8), byte((n-8)>>16)
	for i := 8; i < n; i++ {
		b[i] = byte(i*7 + 3)
	}
	return b
}

func TestFramesLayout(t *testing.T) {
	file := fakeFirmware(97280) // same size as HBTR002-XTE-2026(10.30).fot → 190 frames
	fr := Frames(file, 0)
	if len(fr) != 190 {
		t.Fatalf("frames = %d, want 190", len(fr))
	}
	// Values seen in the Swift dry-run of the real file.
	if got := fr[0].Data[:14]; !bytes.Equal(got, []byte{0xA0, 0x03, 0, 0, 0, 0, 0, 0x02, 0, 0, 'P', 'O', 'T', 0}) {
		t.Errorf("frame 0 header % X", got)
	}
	if fr[188].Data[1] != 0x7B || fr[188].Data[3] != 0x78 || fr[188].Data[4] != 0x01 || fr[188].Tail[1] != 0x7C {
		t.Errorf("frame 188 counters % X / % X", fr[188].Data[:5], fr[188].Tail[:2])
	}
	if fr[189].Data[1] != 0x7D || fr[189].Tail[1] != 0x7E {
		t.Errorf("frame 189 counters")
	}
	var re []byte
	for _, f := range fr {
		if len(f.Data) != DataPacketSize || len(f.Tail) != 15 || f.Tail[0] != 0x20 {
			t.Fatalf("frame %d sizes %d/%d", f.Index, len(f.Data), len(f.Tail))
		}
		re = append(re, f.Data[10:]...)
		re = append(re, f.Tail[2:]...)
	}
	if !bytes.Equal(re[:len(file)], file) {
		t.Fatal("frames do not reassemble to the file")
	}
}

func TestFramesPaddingAndResume(t *testing.T) {
	file := fakeFirmware(1000)
	fr := Frames(file, 512)
	if len(fr) != 1 || fr[0].FileOffset != 512 || fr[0].Data[1] != 3 || fr[0].Data[3] != 0 {
		t.Fatalf("resume frame wrong: %+v", fr[0].Data[:5])
	}
	// 488 real bytes: all in the data packet, rest padded with FF, tail all FF.
	if fr[0].Data[10+487] != file[999] || fr[0].Data[10+488] != 0xFF || fr[0].Tail[2] != 0xFF {
		t.Fatal("padding wrong")
	}
}

// fakeDevice simulates the page turner's OTA state machine.
type fakeDevice struct {
	t       *testing.T
	file    []byte
	stale   bool // still in transfer mode from an earlier session
	badKey  bool
	resume  int
	state   int
	got     []byte
	pending []byte // data packet waiting for its tail
	replies chan []byte
	nextZz  int
}

func newFake(t *testing.T, file []byte) *fakeDevice {
	return &fakeDevice{t: t, file: file, replies: make(chan []byte, 8), nextZz: 3}
}

func (d *fakeDevice) Write(_ context.Context, p []byte) error {
	reply := func(b ...byte) { d.replies <- b }
	if d.stale {
		reply(0x90, p[1]&0x0F, 0x40)
		return nil
	}
	switch {
	case d.state == 0 && bytes.Equal(p, Cmd1):
		d.state = 1
	case d.state == 1 && bytes.Equal(p, Cmd2):
		d.state = 2
		reply(0x92, 0x00, 0x01, 0x02, 0x70, 0x10, 0x03, 0x02, 0, 0, 0x04, 0x02, 0, 0)
	case d.state == 2 && bytes.Equal(p, Cmd3):
		d.state = 3
		reply(0x91, 0x01, 0x01, 0x70, 0x10)
	case d.state == 3 && bytes.Equal(p, Cmd4):
		d.state = 4
		o := d.resume
		reply(0x91, 0x02, 0x02, byte(o), byte(o>>8), byte(o>>16), byte(o>>24), 0x00, 0x02, 0x00, 0x00, 0xFD, 0x01, 0x01)
	case d.state == 4 && len(p) == DataPacketSize && p[0] == 0xA0:
		if int(p[1]) != d.nextZz&0xFF {
			reply(0x90, p[1], 0x40)
			return nil
		}
		d.pending = p
	case d.state == 4 && len(p) == 15 && p[0] == 0x20 && d.pending != nil:
		if d.badKey {
			reply(0x90, p[1], 0x02)
			return nil
		}
		d.got = append(d.got, d.pending[10:]...)
		d.got = append(d.got, p[2:]...)
		d.pending = nil
		d.nextZz += 2
		if d.resume+len(d.got) >= len(d.file) {
			reply(0x90, p[1], 0xFF)
		} else {
			reply(0x90, p[1], 0x00)
		}
	default:
		d.t.Fatalf("unexpected write in state %d: % X", d.state, p[:min(len(p), 12)])
	}
	return nil
}

func (d *fakeDevice) Recv(ctx context.Context, timeout time.Duration) ([]byte, error) {
	select {
	case r := <-d.replies:
		return r, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestFlashFull(t *testing.T) {
	file := fakeFirmware(97280)
	d := newFake(t, file)
	var last int
	err := Flash(context.Background(), d, file, Options{Progress: func(done, total int) { last = done }})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(d.got[:len(file)], file) || last != len(file) {
		t.Fatalf("device received %d bytes, progress %d", len(d.got), last)
	}
}

func TestFlashResume(t *testing.T) {
	file := fakeFirmware(97280)
	d := newFake(t, file)
	d.resume = 51200
	if err := Flash(context.Background(), d, file, Options{}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(d.got[:len(file)-51200], file[51200:]) {
		t.Fatal("resume sent wrong bytes")
	}
}

func TestFlashHandshakeOnly(t *testing.T) {
	file := fakeFirmware(4096)
	d := newFake(t, file)
	if err := Flash(context.Background(), d, file, Options{HandshakeOnly: true}); err != nil {
		t.Fatal(err)
	}
	if len(d.got) != 0 || d.state != 4 {
		t.Fatal("handshake-only sent data")
	}
}

func TestFlashStale(t *testing.T) {
	file := fakeFirmware(4096)
	d := newFake(t, file)
	d.stale = true
	if err := Flash(context.Background(), d, file, Options{}); !errors.Is(err, ErrStillInOTA) {
		t.Fatalf("err = %v, want ErrStillInOTA", err)
	}
}

func TestFlashKeyMismatch(t *testing.T) {
	file := fakeFirmware(4096)
	d := newFake(t, file)
	d.badKey = true
	var de DeviceError
	if err := Flash(context.Background(), d, file, Options{}); !errors.As(err, &de) || de.Status != 0x02 {
		t.Fatalf("err = %v, want key mismatch", err)
	}
}

func TestFlashRejectsNonFot(t *testing.T) {
	if err := Flash(context.Background(), newFake(t, nil), []byte("hello world, not firmware"), Options{}); err == nil {
		t.Fatal("accepted a non-.fot file")
	}
}

func TestCatalogueRules(t *testing.T) {
	b, err := os.ReadFile("../testdata/firmware_manifest.json")
	if err != nil {
		t.Skip(err)
	}
	var m struct{ Firmwares []Firmware }
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if got := Series("HBTR002-XTE3"); got != "HBTR002" {
		t.Errorf("Series = %q", got)
	}
	if got := Variant("HBTR002-XTE3"); got != "XTE" {
		t.Errorf("Variant = %q", got)
	}
	if n := len(Candidates(m.Firmwares, "HBTR002-XTE3")); n != 5 {
		t.Errorf("candidates = %d, want 5 (KD, X-2025, X-2026, XTE-2025, XTE-2026)", n)
	}
	// Same hardware (XTE and X) → all series files stay listed, like the mini-program.
	if n := len(Compatible(m.Firmwares, "HBTR002-XTE3")); n != 5 {
		t.Errorf("HBTR002-XTE3 compatible = %d, want 5", n)
	}
	// Kobo hardware: the Xteink-only XT files are hidden, Kindle (E) files are listed.
	for _, f := range Compatible(m.Firmwares, "HBTR001-K") {
		if FileVariant(f) == "XT" {
			t.Errorf("HBTR001-K lists %s", f.FileName())
		}
	}
	if !Supported("HBTR002-XTE3") || !Supported("Smart 1-EMP") || Supported("COROS PACE Pro") {
		t.Error("Supported()")
	}
}

func TestReadVersion(t *testing.T) {
	d := newFake(t, fakeFirmware(4096))
	v, err := ReadVersion(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	// fake replies like the real device before our flash: 92 00 01 02 70 10 …
	if v.Code != 0x1070 || v.String() != "V1.0.7" || len(v.Params) != 3 {
		t.Fatalf("got %+v %q", v, v.String())
	}
	stale := newFake(t, fakeFirmware(4096))
	stale.stale = true
	if _, err := ReadVersion(context.Background(), stale); !errors.Is(err, ErrStillInOTA) {
		t.Fatalf("stale: %v", err)
	}
}

func TestBuildKey(t *testing.T) {
	for in, want := range map[string]string{
		"HBTR002-XTE-2026(10.30).fot":     "HBTR002-XTE(10.30)",
		"HBTR002-XTE-2025(10.30).fot":     "HBTR002-XTE(10.30)",
		"HBTR003-KD-2026.fot":             "HBTR003-KD",
		"HBTR001-K-2025-长按小按键锁屏.fot":      "HBTR001-K-长按小按键锁屏",
		"Smart 1-EMP-Kindle-20261030.fot": "Smart 1-EMP-Kindle-20261030",
	} {
		if got := BuildKey(in); got != want {
			t.Errorf("BuildKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMode(t *testing.T) {
	cases := []struct {
		name string
		id   uint16
		has  bool
		want string
	}{
		{"HBTR002-XTE3", 0x0171, true, "Kindle"},
		{"HBTR002-XTE", 0x05AC, true, "Xteink"},
		{"HBTR002-XT", 0x05AC, true, "Android"},
		{"HBTR001-K", 0x0171, true, "Kindle"},
		{"HBTR001-K", 0x05AC, true, "Android / Xteink"},
		{"HBTR001-K", 0, false, ""},
	}
	for _, c := range cases {
		if got := Mode(c.name, c.id, c.has); got != c.want {
			t.Errorf("Mode(%q, %#x) = %q, want %q", c.name, c.id, got, c.want)
		}
	}
}

func TestEditions(t *testing.T) {
	f := func(n string) Firmware { return Firmware{URL: "https://x/" + n} }
	for name, want := range map[string]string{
		"HBTR002-X-2026(11.01).fot":       "X",
		"HBTR002-XTE-2025(10.30).fot":     "XTE",
		"HBTR002-KD.fot":                  "KD",
		"HBTR003-KD-2026.fot":             "KD",
		"HBTR001-2025-长按小按键5秒重置蓝牙.fot":    "",
		"HBTR001-K-2026-长按小按键锁屏.fot":      "K",
		"Smart 1-EMP-Kindle-20261030.fot": "EMP",
	} {
		if got := FileVariant(f(name)); got != want {
			t.Errorf("FileVariant(%s) = %q, want %q", name, got, want)
		}
	}
	if !OtherEdition(f("HBTR002-X-2026(11.01).fot"), "HBTR002-XTE3") || OtherEdition(f("HBTR002-XTE-2026(10.30).fot"), "HBTR002-XT") {
		t.Error("OtherEdition")
	}
	if Variant("HBTR001-XT") != "XT" {
		t.Error("alias must be per series")
	}
}

func TestFitHBTR001Hardware(t *testing.T) {
	f := func(n string) Firmware { return Firmware{URL: "https://x/" + n} }
	cases := []struct {
		file, device string
		want         Fit
	}{
		{"HBTR001-K-2026-长按小按键锁屏.fot", "HBTR001-K", FitSame},
		{"HBTR001-E-2026(10.30).fot", "HBTR001-K", FitOtherEdition}, // Kobo ↔ Kindle: same hardware
		{"HBTR001-K-2026-长按小按键锁屏.fot", "HBTR001-E", FitOtherEdition},
		{"HBTR001-XT-2026-长按小按键5秒重置蓝牙.fot", "HBTR001-K", FitIncompatible}, // Xteink hardware differs
		{"HBTR001-E-2026(10.30).fot", "HBTR001-XT", FitIncompatible},
		{"HBTR001-2026-长按小按键5秒重置蓝牙.fot", "HBTR001-K", FitOtherEdition}, // unknown: warn only
		{"HBTR002-X-2026(11.01).fot", "HBTR002-XTE3", FitOtherEdition},
	}
	for _, c := range cases {
		if got := FitFor(f(c.file), c.device); got != c.want {
			t.Errorf("FitFor(%s, %s) = %v, want %v", c.file, c.device, got, c.want)
		}
	}
}

func TestFetchUsesMakersFileAndChecksIt(t *testing.T) {
	good := make([]byte, 1024)
	copy(good, "POT\x00")
	binary.LittleEndian.PutUint32(good[4:], uint32(len(good)-8))
	serve := good
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(serve) }))
	defer srv.Close()
	f := Firmware{URL: srv.URL + "/HBTR001-E-2025(10.30).fot"}

	// A complete file that differs from the checked copy (the maker updated it) is still used.
	data, note, err := Fetch(f)
	if err != nil || len(data) != len(good) || !strings.Contains(note, "maker updated") {
		t.Fatalf("updated file: %d bytes, %q, %v", len(data), note, err)
	}
	// A cut-off download is refused.
	serve = good[:600]
	if _, _, err := Fetch(f); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("truncated file accepted: %v", err)
	}
	// The embedded list of checked files is well-formed.
	for name, c := range CheckedFiles {
		if c.Size < 8 || len(c.SHA256) != 64 {
			t.Errorf("%s: bad entry %+v", name, c)
		}
	}
}
