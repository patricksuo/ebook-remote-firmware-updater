// blecheck is a READ-ONLY check of the pure-Go CoreBluetooth binding: scan, connect, discover the 智联
// OTA service, print the MTU. Nothing is written and no notifications are enabled.
//
//	go build -o /tmp/blecheck ./cmd/blecheck && /tmp/blecheck HBTR
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"pageturner-updater/ble"
	"pageturner-updater/internal/mainloop"
	"pageturner-updater/zl"
)

func main() {
	want := "HBTR"
	if len(os.Args) > 1 {
		want = os.Args[1]
	}
	mainloop.Run(func() {
		if err := run(strings.ToLower(want)); err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}
		os.Exit(0)
	})
}

func run(want string) error {
	c, err := ble.New(mainloop.RunOnMain)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.WaitReady(ctx); err != nil {
		return err
	}
	fmt.Println("Bluetooth on; scanning 10 s for", want)
	found := make(chan ble.Advertisement, 1)
	scanCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	go func() {
		_ = c.Scan(scanCtx, []string{zl.ServiceUUID, "1812", "180F"}, func(a ble.Advertisement) {
			if a.Name == "" {
				return
			}
			fmt.Printf("  %-28s rssi=%4d company=%#04x connected=%v services=%v\n", a.Name, a.RSSI, a.CompanyID, a.Connected, a.Services)
			if strings.Contains(strings.ToLower(a.Name), want) {
				select {
				case found <- a:
				default:
				}
			}
		})
	}()
	var a ble.Advertisement
	select {
	case a = <-found:
		stop()
	case <-scanCtx.Done():
		return fmt.Errorf("no device matching %q", want)
	}
	time.Sleep(300 * time.Millisecond) // let Scan stop
	fmt.Println("connecting to", a.Name)
	conn, err := c.Connect(ctx, a)
	if err != nil {
		return err
	}
	defer conn.Close()
	props, err := conn.Discover(ctx, zl.ServiceUUID, zl.WriteUUID, zl.NotifyUUID)
	if err != nil {
		return err
	}
	fmt.Printf("FF15 props %#x (writeNoResp=%v)\n", props[zl.WriteUUID], props[zl.WriteUUID]&ble.PropWriteWithoutResponse != 0)
	fmt.Printf("FF14 props %#x (notify=%v)\n", props[zl.NotifyUUID], props[zl.NotifyUUID]&ble.PropNotify != 0)
	fmt.Println("max write without response:", conn.MaxWrite(true))
	fmt.Println("read-only: nothing written")
	return nil
}
