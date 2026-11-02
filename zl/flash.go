package zl

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Link is an open OTA channel: writes go to FF15 (without response), replies arrive from FF14 notifications.
type Link interface {
	// Write sends one packet; it may block for flow control.
	Write(ctx context.Context, p []byte) error
	// Recv returns the next notification, or an error after timeout / when the device disconnects.
	Recv(ctx context.Context, timeout time.Duration) ([]byte, error)
}

var (
	ErrTimeout = errors.New("timed out waiting for the device")
	// ErrStillInOTA: an earlier session stopped after the handshake and the device is still waiting for data.
	ErrStillInOTA = errors.New("the page turner is still in update mode from an earlier attempt — switch it OFF and back ON, then try again")
	ErrNotAllowed = errors.New("the page turner refused the update")
)

type Options struct {
	HandshakeOnly bool
	Progress      func(doneBytes, totalBytes int) // called after each acknowledged frame
	Log           func(string)
}

// DeviceError is a non-OK status in a frame acknowledgement.
type DeviceError struct{ Status byte }

func (e DeviceError) Error() string {
	t, ok := StatusText[e.Status]
	if !ok {
		t = "unknown error"
	}
	return fmt.Sprintf("device reported 0x%02X: %s", e.Status, t)
}

// Flash runs the whole update (or just the handshake) over link. It mirrors the mini-program.
func Flash(ctx context.Context, link Link, file []byte, o Options) error {
	logf := func(f string, a ...any) {
		if o.Log != nil {
			o.Log(fmt.Sprintf(f, a...))
		}
	}
	if p, _ := Inspect(file); p != "" {
		return errors.New(p)
	}
	send := func(name string, p []byte) error {
		logf("→ %s %s", name, hexs(p, 20))
		return link.Write(ctx, p)
	}
	// wait returns the next reply whose first byte is want; 0x90 during the handshake means a stale session.
	wait := func(want byte, timeout time.Duration, what string) ([]byte, error) {
		deadline := time.Now().Add(timeout)
		for {
			left := time.Until(deadline)
			if left <= 0 {
				return nil, fmt.Errorf("%w (%s)", ErrTimeout, what)
			}
			r, err := link.Recv(ctx, left)
			if err != nil {
				if errors.Is(err, ErrTimeout) {
					return nil, fmt.Errorf("%w (%s)", ErrTimeout, what)
				}
				return nil, err
			}
			logf("← %s", hexs(r, 20))
			if len(r) == 0 {
				continue
			}
			if r[0] == want {
				return r, nil
			}
			if r[0] == 0x90 && want != 0x90 {
				return nil, ErrStillInOTA
			}
			logf("(ignored unexpected reply)")
		}
	}

	if err := send("cmd1", Cmd1); err != nil {
		return err
	}
	if err := send("cmd2", Cmd2); err != nil {
		return err
	}
	if _, err := wait(0x92, 5*time.Second, "reply to cmd2"); err != nil {
		return err
	}
	if err := send("cmd3", Cmd3); err != nil {
		return err
	}
	if _, err := wait(0x91, 5*time.Second, "reply to cmd3"); err != nil {
		return err
	}
	if err := send("cmd4", Cmd4); err != nil {
		return err
	}
	r, err := wait(0x91, 5*time.Second, "reply to cmd4")
	if err != nil {
		return err
	}
	off, allow, _ := ParseStart(r)
	logf("device: start offset %d, allow 0x%02X", off, allow)
	if allow != 1 {
		return ErrNotAllowed
	}
	if off < 0 || off >= len(file) {
		return fmt.Errorf("device asked to resume at %d, outside the %d-byte file", off, len(file))
	}
	if o.HandshakeOnly {
		logf("handshake only: stopping (switch the device OFF/ON before the next attempt)")
		return nil
	}

	frames := Frames(file, off)
	logf("sending %d frames from offset %d", len(frames), off)
	for i, f := range frames {
		if err := link.Write(ctx, f.Data); err != nil {
			return err
		}
		if err := link.Write(ctx, f.Tail); err != nil {
			return err
		}
		ack, err := wait(0x90, 10*time.Second, fmt.Sprintf("acknowledgement of frame %d", i))
		if err != nil {
			return err
		}
		status := byte(0xEE)
		if len(ack) > 2 {
			status = ack[2]
		}
		done := min(len(file), f.FileOffset+FrameSize)
		switch status {
		case 0xFF:
			if o.Progress != nil {
				o.Progress(len(file), len(file))
			}
			logf("upgrade complete")
			return nil
		case 0x00:
			if o.Progress != nil {
				o.Progress(done, len(file))
			}
		default:
			return DeviceError{status}
		}
	}
	// All frames acknowledged with 00: the final 0xFF should follow.
	ack, err := wait(0x90, 15*time.Second, "final confirmation")
	if err != nil {
		return err
	}
	if len(ack) > 2 && ack[2] == 0xFF {
		if o.Progress != nil {
			o.Progress(len(file), len(file))
		}
		return nil
	}
	if len(ack) > 2 {
		return DeviceError{ack[2]}
	}
	return errors.New("unexpected final reply")
}

func hexs(b []byte, limit int) string {
	s := ""
	for i, x := range b {
		if i == limit {
			return s + fmt.Sprintf("… (%d bytes)", len(b))
		}
		s += fmt.Sprintf("%02X ", x)
	}
	return s
}

// BusyAfterQuery is how long the device stays in transfer mode after ReadVersion (it times out by itself;
// observed < 60 s). Starting an update meanwhile fails with ErrStillInOTA.
const BusyAfterQuery = 60 * time.Second

// Version is the reply to Cmd2: param 0x01 is the installed firmware version (e.g. 0x1090).
type Version struct {
	Code   uint16
	Params map[byte][]byte
}

// String renders the code as the firmware's SW_VERSION, e.g. 0x1090 → "V1.0.9". Bluetrum's FOT module
// (bt_fota.c, bsp_fot_init) packs the digits of SW_VERSION "Va.b.c" into the top three nibbles.
func (v Version) String() string {
	return fmt.Sprintf("V%d.%d.%d", v.Code>>12&0xF, v.Code>>8&0xF, v.Code>>4&0xF)
}

// ReadVersion sends only Cmd1+Cmd2 and parses the 0x92 reply. Side effect: the device enters transfer
// mode for up to BusyAfterQuery.
func ReadVersion(ctx context.Context, link Link) (Version, error) {
	if err := link.Write(ctx, Cmd1); err != nil {
		return Version{}, err
	}
	if err := link.Write(ctx, Cmd2); err != nil {
		return Version{}, err
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		left := time.Until(deadline)
		if left <= 0 {
			return Version{}, fmt.Errorf("%w (version reply)", ErrTimeout)
		}
		r, err := link.Recv(ctx, left)
		if err != nil {
			return Version{}, err
		}
		if len(r) == 0 {
			continue
		}
		if r[0] == 0x90 {
			return Version{}, ErrStillInOTA
		}
		if r[0] != 0x92 {
			continue
		}
		v := Version{Params: map[byte][]byte{}}
		for i := 2; i+1 < len(r); {
			id, n := r[i], int(r[i+1])
			if i+2+n > len(r) {
				break
			}
			v.Params[id] = append([]byte(nil), r[i+2:i+2+n]...)
			i += 2 + n
		}
		p := v.Params[0x01]
		if len(p) < 2 {
			return v, errors.New("version missing in the device's reply")
		}
		v.Code = uint16(p[0]) | uint16(p[1])<<8
		return v, nil
	}
}
