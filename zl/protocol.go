// Package zl implements the firmware-update protocol of 智联纵维 (diykey) page turners
// (HBTR00x, Smart 1, Zero 1). See docs/PROTOCOL.md. Pure logic, no Bluetooth.
package zl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	ManifestURL = "https://diykey-1426533931.cos.ap-guangzhou.myqcloud.com/firmware_manifest.json"

	ServiceUUID = "FF12"
	WriteUUID   = "FF15" // write without response
	NotifyUUID  = "FF14" // notify

	DataPacketSize = 509 // 10-byte header + 499 payload
	DataPayload    = 499
	TailPayload    = 13                        // tail packet = 2-byte header + 13 payload
	FrameSize      = DataPayload + TailPayload // 512 file bytes per frame
	MaxFirmware    = 256 * 1024
)

var (
	Cmd1 = []byte{0xCC, 0xAA, 0x55, 0xEE, 0x12, 0x19, 0xE4}
	Cmd2 = []byte{0x92, 0x00, 0x01, 0x00, 0x03, 0x00, 0x04, 0x00, 0x06, 0x00}
	Cmd3 = []byte{0x91, 0x01, 0x01}
	Cmd4 = []byte{0x91, 0x02, 0x02, 0xFF, 0xFF, 0x06, 0x8E, 0x3D, 0x19}
)

// StatusText explains the third byte of a 0x90 frame acknowledgement.
var StatusText = map[byte]string{
	0x00: "OK", 0x01: "firmware is identical to the current one", 0x02: "key mismatch (wrong firmware for this device)",
	0x0B: "CRC error", 0x40: "sequence error", 0x41: "length error",
	0xFD: "paused", 0xFE: "resume", 0xFF: "upgrade complete",
}

// Frame is one 512-byte slice of the file, sent as a 509-byte data packet and a 15-byte tail packet.
type Frame struct {
	Index      int
	FileOffset int
	Data       []byte
	Tail       []byte
}

// Frames builds every frame exactly like the mini-program (keyboard page sendNextFilePacket /
// sendEndPacketForCurrentFile). Counters start at zz=3, endZz=4, seq=0 regardless of startOffset.
func Frames(file []byte, startOffset int) []Frame {
	var out []Frame
	zz, endZz, seq := 3, 4, 0
	for off := startOffset; off < len(file); off += FrameSize {
		data := make([]byte, DataPacketSize)
		for i := range data {
			data[i] = 0xFF
		}
		data[0], data[1], data[2] = 0xA0, byte(zz), 0
		data[3], data[4] = byte(seq), byte(seq>>8)
		data[5], data[6], data[7], data[8], data[9] = 0, 0, 2, 0, 0
		copy(data[10:], file[off:min(len(file), off+DataPayload)])

		tail := make([]byte, 2+TailPayload)
		for i := range tail {
			tail[i] = 0xFF
		}
		tail[0], tail[1] = 0x20, byte(endZz)
		if t := off + DataPayload; t < len(file) {
			copy(tail[2:], file[t:min(len(file), t+TailPayload)])
		}
		out = append(out, Frame{Index: len(out), FileOffset: off, Data: data, Tail: tail})
		zz += 2
		seq += 2
		endZz += 2
	}
	return out
}

// ParseStart parses the reply to Cmd4: the offset to resume from and the "upgrade allowed" flag.
func ParseStart(r []byte) (offset int, allow byte, ok bool) {
	if len(r) == 0 || r[0] != 0x91 {
		return 0, 0, false
	}
	if len(r) >= 7 {
		offset = int(r[3]) | int(r[4])<<8 | int(r[5])<<16 | int(r[6])<<24
	}
	return offset, r[len(r)-1], true
}

// Inspect returns problems with a .fot image ("" if it looks fine) and informational notes.
func Inspect(file []byte) (problem string, notes []string) {
	if len(file) < 8 || string(file[:4]) != "POT\x00" {
		return "not a known .fot image (missing \"POT\" header)", nil
	}
	n := int(file[4]) | int(file[5])<<8 | int(file[6])<<16 | int(file[7])<<24
	notes = append(notes, fmt.Sprintf("header length field %d, file %d bytes", n, len(file)))
	if n != len(file)-8 {
		return fmt.Sprintf("incomplete or damaged file (header says %d bytes, got %d)", n+8, len(file)), notes
	}
	if len(file) > MaxFirmware {
		return fmt.Sprintf("larger than %d KB", MaxFirmware/1024), notes
	}
	return "", notes
}

// ---- firmware catalogue ----

type Firmware struct {
	Version     string `json:"version"`
	Date        string `json:"date"`
	URL         string `json:"url"`
	Size        int    `json:"size"` // not reliable; use the downloaded length
	Description string `json:"description"`
}

// FileName is the decoded last path element of the URL, e.g. "HBTR002-XTE-2026(10.30).fot".
func (f Firmware) FileName() string {
	i := strings.LastIndex(f.URL, "/")
	name := f.URL[i+1:]
	if d, err := url.PathUnescape(name); err == nil {
		return d
	}
	return name
}

// DisplayName is FileName without ".fot".
func (f Firmware) DisplayName() string {
	n := f.FileName()
	if strings.HasSuffix(strings.ToLower(n), ".fot") {
		return n[:len(n)-4]
	}
	return n
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

func FetchManifest() ([]Firmware, error) {
	b, err := get(ManifestURL)
	if err != nil {
		return nil, err
	}
	var m struct {
		Firmwares []Firmware `json:"firmwares"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("firmware list: %w", err)
	}
	return m.Firmwares, nil
}

func get(u string) ([]byte, error) {
	resp, err := httpClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download %s: HTTP %d", u, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 16<<20))
}

var lastChunk = regexp.MustCompile(`-[^-]*$`)

// Series applies the mini-program rule: drop the last "-xxx" chunk of the BLE name.
func Series(deviceName string) string {
	return strings.TrimSpace(lastChunk.ReplaceAllString(deviceName, ""))
}

// Candidates are the catalogue files whose name starts with the device's series (case-insensitive), sorted.
func Candidates(all []Firmware, deviceName string) []Firmware {
	s := strings.ToUpper(Series(deviceName))
	var out []Firmware
	for _, f := range all {
		if strings.HasPrefix(strings.ToUpper(f.FileName()), s) {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FileName() < out[j].FileName() })
	return out
}

var trailingDigits = regexp.MustCompile(`\d+$`)

var yearChunk = regexp.MustCompile(`-20\d\d([-(]|$)`)

// BuildKey identifies the program in a catalogue file, ignoring the hardware-batch year: the 2025 and 2026
// files are the same build with a different chip KEY. "HBTR002-XTE-2026(10.30).fot" → "HBTR002-XTE(10.30)".
func BuildKey(fileName string) string {
	n := strings.TrimSuffix(fileName, ".fot")
	return yearChunk.ReplaceAllString(n, "$1")
}

// variantAliases maps a BLE-name variant to the firmware variant, per series. The HBTR002-XTE firmware
// advertises as "-XTE3" (Kindle), "-XTE" (Xteink) and "-XT" (Android) depending on the mode switch.
var variantAliases = map[string]map[string]string{
	"HBTR002": {"XT": "XTE"},
	"HBTR003": {"XT": "XTE"}, // same 3-mode XTE firmware; mode names assumed to follow HBTR002
}

// Variant guesses the firmware variant from the BLE name's last chunk: "HBTR002-XTE3" → "XTE",
// "HBTR002-XT" → "XTE" (alias). Trailing digits are dropped (the Kindle-mode name appends "3").
func Variant(deviceName string) string {
	i := strings.LastIndex(deviceName, "-")
	if i < 0 {
		return ""
	}
	v := trailingDigits.ReplaceAllString(strings.TrimSpace(deviceName[i+1:]), "")
	if a, ok := variantAliases[strings.ToUpper(Series(deviceName))][strings.ToUpper(v)]; ok {
		return a
	}
	return v
}

var fileVariant = regexp.MustCompile(`^[^-]+-([A-Za-z]+)(?:-|\(|$)`)

// FileVariant is the variant in a catalogue file name: "HBTR002-X-2026(11.01).fot" → "X",
// "HBTR002-KD.fot" → "KD", "HBTR001-2025-长按….fot" → "" (no variant).
func FileVariant(f Firmware) string {
	if m := fileVariant.FindStringSubmatch(f.DisplayName()); m != nil {
		return strings.ToUpper(m[1])
	}
	return ""
}

// hardwareGroups lists, per series, which firmware variants run on the same hardware (can be cross-flashed).
// Variants of a series in different groups need different hardware. Unlisted variants: unknown.
var hardwareGroups = map[string][][]string{
	// HBTR001 exists as two hardware versions: the Xteink one runs only XT; Kobo (K) and Kindle (E) devices
	// can be flashed with each other's firmware (confirmed by the maker: HBTR001-K → HBTR001-E).
	"HBTR001": {{"K", "E"}, {"XT"}},
	// HBTR002: an XTE device was flashed with X (phone/tablet edition) and back; both run.
	"HBTR002": {{"XTE", "X"}},
}

// Fit says how a catalogue file relates to the firmware currently on a device.
type Fit int

const (
	FitSame         Fit = iota // same edition as on the device
	FitOtherEdition            // a different edition for the same (or possibly the same) hardware: features change
	FitIncompatible            // needs different hardware: don't install
)

func groupOf(series, variant string) int {
	for i, g := range hardwareGroups[strings.ToUpper(series)] {
		for _, v := range g {
			if v == variant {
				return i
			}
		}
	}
	return -1
}

// FitFor compares file f with the firmware variant of the device (from its BLE name).
func FitFor(f Firmware, deviceName string) Fit {
	dv := strings.ToUpper(Variant(deviceName))
	fv := FileVariant(f)
	if dv == "" || fv == dv {
		return FitSame
	}
	series := Series(deviceName)
	if gd, gf := groupOf(series, dv), groupOf(series, fv); gd >= 0 && gf >= 0 && gd != gf {
		return FitIncompatible
	}
	return FitOtherEdition
}

// OtherEdition reports whether f is for a different edition than the device's current firmware
// (e.g. HBTR002-X phone/tablet firmware on an HBTR002-XTE e-reader remote).
func OtherEdition(f Firmware, deviceName string) bool { return FitFor(f, deviceName) != FitSame }

// Compatible is Candidates without the files known to need different hardware (FitIncompatible).
func Compatible(all []Firmware, deviceName string) []Firmware {
	var out []Firmware
	for _, f := range Candidates(all, deviceName) {
		if FitFor(f, deviceName) != FitIncompatible {
			out = append(out, f)
		}
	}
	return out
}

// Mode tells which host a page turner is currently set up for. The name decides when it's a known one
// (HBTR002 product page: "-XT" = Android, "-XTE" = Xteink readers; "-XTE3" = Kindle, observed); otherwise
// the advertised company ID (0x0171 Kindle; Apple 0x05AC is used by the non-Kindle modes).
func Mode(name string, companyID uint16, hasCompany bool) string {
	switch u := strings.ToUpper(strings.TrimSpace(name)); {
	case strings.HasSuffix(u, "-XTE3"):
		return "Kindle"
	case strings.HasSuffix(u, "-XTE"):
		return "Xteink"
	case strings.HasSuffix(u, "-XT"):
		return "Android"
	}
	if !hasCompany {
		return ""
	}
	switch companyID {
	case 0x0171:
		return "Kindle"
	case 0x05AC:
		return "Android / Xteink"
	}
	return ""
}

// Supported reports whether a BLE name looks like a 智联纵维 device.
func Supported(name string) bool {
	u := strings.ToUpper(strings.TrimSpace(name))
	for _, p := range []string{"HBTR", "SMART 1", "SMART1", "ZERO 1", "ZERO1"} {
		if strings.HasPrefix(u, p) {
			return true
		}
	}
	return false
}
