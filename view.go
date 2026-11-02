package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"pageturner-updater/zl"
)

// ---- layout ----

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Surface).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(680).AlignSelf(ui.Center).Padding(28, 24, 32).Gap(16).Children(func() {
			a.header(c)
			a.stepper(c)
			if a.bleErr != "" {
				notice(c, "bt-err", t.Danger, icAlert, a.bleErr, func() {
					if ui.Button(c.Key("bt-settings"), "Open Bluetooth Settings").Clicked() {
						_ = mygo.Shell.OpenExternal(bluetoothSettings)
					}
				})
			}
			a.devicesCard(c)
			if a.selected() != nil {
				a.checkCard(c)
				a.firmwareCard(c)
				a.updateCard(c)
			}
			a.logSection(c)
		})
	})
	if f, ok := a.chosenFirmware(); ok {
		d := a.selected()
		switch ui.AlertDialog(c, &a.confirming,
			"Install "+f.DisplayName()+"?",
			a.confirmText(f, d.Name),
			"Cancel", "Install") {
		case 1:
			a.updating, a.done, a.failed, a.progress = true, false, "", 0
			go a.runUpdate(d.Advertisement, f)
		}
	}
}

func (a *app) confirmText(f zl.Firmware, device string) string {
	msg := fmt.Sprintf("This replaces the firmware of %s. Keep it charged and next to the Mac, and don't press its buttons until it's done (about 20 seconds).", device)
	if zl.OtherEdition(f, device) {
		msg = fmt.Sprintf("Warning: %s is a different edition than the firmware on %s (%s), so its features will change", f.DisplayName(), device, zl.Variant(device))
		if n := variantNote(f); n != "" {
			msg += " — " + n
		}
		msg += ".\n\n" + fmt.Sprintf("Keep it charged and next to the Mac, and don't press its buttons until it's done.")
	}
	return msg
}

func (a *app) header(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Gap(14).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(48, 48).Radius(12).Background(t.Accent).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Icon(c, icBook).FontSize(26).TextColor(t.AccentText)
		})
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.End).Children(func() {
				ui.Text(c, "Page Turner Updater").FontSize(22).Bold()
				ui.Text(c, a.appVer).FontSize(13).TextColor(t.TextMuted)
			})
			ui.Text(c, "Firmware updates for 智联纵维 page turners — HBTR001/002/003, Smart 1, Zero 1").
				FontSize(13).TextColor(t.TextMuted).Wrap()
		})
	})
}

var stepNames = []string{"Device", "Check", "Firmware", "Install"}

func (a *app) stepper(c *ui.Context) {
	t := c.Theme()
	cur := a.step()
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4, 2).Children(func() {
		for i, name := range stepNames {
			n := i + 1
			if i > 0 {
				col := t.Border
				if n <= cur {
					col = t.Accent
				}
				ui.Box(c.Key(fmt.Sprint("line", n))).Height(2).Grow(1).Radius(1).Background(col)
			}
			ui.Row(c.Key(fmt.Sprint("step", n))).Gap(6).AlignItems(ui.Center).Children(func() {
				stepBadge(c, n, cur)
				col := t.TextMuted
				if n == cur {
					col = t.Text
				}
				ui.Text(c, name).FontSize(13).TextColor(col).SingleLine()
			})
		}
	})
}

// stepBadge is the numbered circle: done (check), current (accent), upcoming (outline).
func stepBadge(c *ui.Context, n, cur int) {
	t := c.Theme()
	b := ui.Box(c).Size(22, 22).Radius(11).AlignItems(ui.Center).Justify(ui.Center)
	switch {
	case n < cur:
		b.Background(t.Accent).Children(func() { ui.Icon(c, icCheck).FontSize(13).TextColor(t.AccentText) })
	case n == cur:
		b.Background(t.Accent).Children(func() { ui.Textf(c, "%d", n).FontSize(12).Bold().TextColor(t.AccentText) })
	default:
		b.Border(1.5, t.Border).Children(func() { ui.Textf(c, "%d", n).FontSize(12).TextColor(t.TextMuted) })
	}
}

func (a *app) card(c *ui.Context, n int, title, subtitle string, fn func()) {
	t := c.Theme()
	ui.Column(c.Key(fmt.Sprint("card", n))).Padding(18, 20).Gap(14).Radius(14).
		Background(t.Background).Border(1, t.Border).
		Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.04)).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			stepBadge(c, n, a.step())
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Text(c, title).FontSize(15).FontWeight(600)
				if subtitle != "" {
					ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted).Wrap()
				}
			})
		})
		fn()
	})
}

// option is a selectable row (device or firmware). Returns true when clicked.
func option(c *ui.Context, key string, selected, enabled bool, fn func()) bool {
	t := c.Theme()
	bg, border := t.Background, t.Border
	if selected {
		bg, border = t.Accent.Alpha(0.08), t.Accent
	}
	e := ui.Row(c.Key(key)).Padding(11, 14).Gap(12).Radius(10).AlignItems(ui.Center).Background(bg).Border(1, border)
	if enabled {
		e = e.Cursor(ui.CursorPointer)
		if e.Hovered() && !selected {
			e = e.Background(t.SurfaceHover)
		}
	} else {
		e = e.Opacity(0.55)
	}
	e.Children(func() {
		fn()
		if selected {
			ui.Icon(c, icCheckDisc).FontSize(18).TextColor(t.Accent)
		}
	})
	return enabled && e.Clicked()
}

func pill(c *ui.Context, text string, col ui.Color) {
	ui.Box(c).Padding(2, 8).Radius(999).Background(col.Alpha(0.14)).Children(func() {
		ui.Text(c, text).FontSize(11).FontWeight(600).TextColor(col).SingleLine()
	})
}

func notice(c *ui.Context, key string, col ui.Color, icon *ui.SVG, text string, extra func()) {
	ui.Row(c.Key(key)).Padding(12, 14).Gap(10).Radius(10).AlignItems(ui.Center).
		Background(col.Alpha(0.10)).Border(1, col.Alpha(0.35)).Children(func() {
		ui.Icon(c, icon).FontSize(18).TextColor(col)
		ui.Text(c, text).Wrap().Grow(1)
		if extra != nil {
			extra()
		}
	})
}

func muted(c *ui.Context, text string) ui.Element {
	return ui.Text(c, text).FontSize(12).TextColor(c.Theme().TextMuted).Wrap()
}

// ---- 1. device ----

func signalWords(rssi int) string {
	switch {
	case rssi == 0:
		return ""
	case rssi >= -60:
		return "strong signal"
	case rssi >= -75:
		return "good signal"
	}
	return "weak signal — move closer"
}

func (a *app) devicesCard(c *ui.Context) {
	t := c.Theme()
	a.card(c, 1, "Choose your page turner", "Press a button on it to wake it up. It must not be connected to a phone or reader: turn their Bluetooth off for a moment. Any mode works for updating.", func() {
		list := a.sortedDevices()
		if len(list) == 0 {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(14).Radius(10).Background(t.Surface).Children(func() {
				if a.scanning {
					ui.Spinner(c)
					ui.Text(c, "Looking for page turners…").TextColor(t.TextMuted)
				} else {
					ui.Icon(c, icBluetooth).FontSize(18).TextColor(t.TextMuted)
					ui.Text(c, "No page turner found. Wake it up and scan again.").TextColor(t.TextMuted)
				}
			})
		}
		ui.Column(c).Gap(8).Children(func() {
			for _, d := range list {
				d := d
				supported := zl.Supported(d.Name)
				if option(c, "dev-"+d.ID, d.ID == a.selectedID, supported && !a.updating, func() {
					ui.Box(c).Size(34, 34).Radius(17).Background(t.Accent.Alpha(0.12)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
						ui.Icon(c, icBluetooth).FontSize(17).TextColor(t.Accent)
					})
					ui.Column(c).Gap(2).Grow(1).Children(func() {
						ui.Text(c, d.Name).FontWeight(600)
						a.presenceLine(c, d, supported)
					})
					if m := zl.Mode(d.Name, d.CompanyID, d.HasCompany); m != "" {
						pill(c, m+" mode", t.Accent)
					}
				}) {
					a.selectDevice(d)
				}
			}
		})
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			if a.scanning {
				ui.Spinner(c)
				muted(c, "Listening for page turners")
			}
			ui.Box(c).Grow(1)
			ui.Checkbox(c.Key("all"), &a.showAll, "Show all Bluetooth devices").FontSize(12)
		})
	})
}

// presenceLine is the status under a device name: advertising / connected / not visible, refreshed every second.
func (a *app) presenceLine(c *ui.Context, d *device, supported bool) {
	t := c.Theme()
	p, age := a.presenceOf(d)
	dot, text := t.Success, "Ready"
	switch p {
	case presReady:
		if s := signalWords(d.RSSI); s != "" {
			text += " · " + s
		}
	case presMac:
		dot, text = t.Accent, "Connected to this Mac"
	case presBusy:
		dot, text = t.Accent, "Connected — working…"
	case presGone:
		dot = t.TextMuted
		text = "Not visible"
		if !d.seen.IsZero() {
			text = fmt.Sprintf("Not visible for %s", roughAge(age))
		}
		text += " — asleep, switched off, or connected to a phone or reader"
	}
	if !supported {
		text += " · not a supported page turner"
	}
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(7, 7).Radius(4).Background(dot)
		muted(c, text)
	})
	c.After(time.Second)
}

func roughAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d s", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d.Minutes()))
	}
	return "a long time"
}

func (a *app) sortedDevices() []*device {
	var out []*device
	for _, d := range a.devices {
		if a.showAll || zl.Supported(d.Name) {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		si, sj := zl.Supported(out[i].Name), zl.Supported(out[j].Name)
		if si != sj {
			return si
		}
		// By name, not signal strength: RSSI changes constantly and made the list jump. Unnamed devices last.
		ni, nj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if (ni == "") != (nj == "") {
			return ni != ""
		}
		if ni != nj {
			return ni < nj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ---- 2. check ----

func (a *app) checkCard(c *ui.Context) {
	t := c.Theme()
	a.card(c, 2, "Check the page turner", "Read-only: nothing is changed on the device.", func() {
		switch {
		case a.checking:
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "Connecting…").TextColor(t.TextMuted)
			})
			muted(c, "If macOS shows “Connection Request from …”, either button is fine.")
			return
		case a.check == nil:
			return
		case a.check.err != nil:
			notice(c, "check-err", t.Danger, icAlert, "Couldn't check it: "+a.check.err.Error(), func() {
				if ui.Button(c.Key("retry"), "Retry").Clicked() {
					a.retryCheck(a.selected().Name)
				}
			})
			return
		}
		ui.Column(c).Gap(8).Children(func() {
			checkLine(c, a.check.writeOK, "Update channel available")
			checkLine(c, a.check.notifyOK, "Device can answer")
			checkLine(c, a.check.maxWrite >= zl.DataPacketSize, fmt.Sprintf("Connection fast enough (%d-byte packets)", a.check.maxWrite))
		})
		a.versionBox(c)
	})
}

func checkLine(c *ui.Context, ok bool, text string) {
	t := c.Theme()
	icon, col := icCheck, t.Success
	if !ok {
		icon, col = icX, t.Danger
	}
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(20, 20).Radius(10).Background(col.Alpha(0.14)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Icon(c, icon).FontSize(12).TextColor(col)
		})
		ui.Text(c, text)
	})
}

// installedName describes the version on the device using the learned versions.
func (a *app) installedName() string {
	if a.version == nil {
		return ""
	}
	d := a.selected()
	var files []zl.Firmware
	if d != nil {
		for _, f := range zl.Candidates(a.catalogue, d.Name) {
			if zl.FitFor(f, d.Name) == zl.FitSame { // version numbers repeat across editions
				files = append(files, f)
			}
		}
	}
	if names := a.versions.buildsWithVersion(files, a.version.Code); len(names) > 0 {
		return strings.Join(names, " or ")
	}
	return ""
}

func (a *app) versionBox(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Padding(12, 14).Gap(6).Radius(10).Background(t.Surface).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icChip).FontSize(18).TextColor(t.TextMuted)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				muted(c, "Firmware on the device")
				switch {
				case a.readingVer:
					ui.Text(c, "Reading…").TextColor(t.TextMuted)
				case a.version != nil:
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, a.version.String()).FontWeight(600)
						if n := a.installedName(); n != "" {
							ui.Text(c, n).TextColor(t.TextMuted)
						} else {
							ui.Text(c, "build not known yet").TextColor(t.TextMuted)
						}
					})
				default:
					ui.Text(c, "Not read yet").TextColor(t.TextMuted)
				}
			})
			if a.readingVer {
				ui.Spinner(c)
			} else {
				label := "Read"
				if a.version != nil {
					label = "Read again"
				}
				if ui.Button(c.Key("readver"), label).
					Disabled(!a.checkOK() || a.updating || a.busyLeft() > 0).Clicked() {
					a.readingVer, a.verErr = true, ""
					go a.readVersion(a.selected().Advertisement)
				}
			}
		})
		if a.verErr != "" {
			ui.Text(c, a.verErr).FontSize(12).TextColor(t.Danger).Wrap()
		}
		if left := a.busyLeft(); left > 0 {
			muted(c, fmt.Sprintf("The page turner stays in update mode for about %d s and won't turn pages meanwhile.", int(left.Seconds())+1))
			c.After(time.Second)
		} else if a.version == nil && !a.readingVer {
			muted(c, "Reading the version pauses the page turner for up to a minute.")
		}
	})
}

// ---- 3. firmware ----

func (a *app) firmwareCard(c *ui.Context) {
	t := c.Theme()
	d := a.selected()
	a.card(c, 3, "Choose the firmware", fmt.Sprintf("Files for %s. 2025 and 2026 are the same firmware for different hardware batches — a wrong one is refused by the device.", zl.Series(d.Name)), func() {
		switch {
		case a.loadingCat:
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "Loading the firmware list…").TextColor(t.TextMuted)
			})
			return
		case a.catErr != "":
			notice(c, "cat-err", t.Danger, icAlert, "Couldn't load the firmware list: "+a.catErr, func() {
				if ui.Button(c.Key("reload"), "Retry").Clicked() {
					go a.loadCatalogue()
				}
			})
			return
		}
		choices := a.firmwareChoices()
		if len(choices) == 0 {
			notice(c, "no-fw", t.Warning, icAlert, "No firmware in the list matches this device.", nil)
		}
		ui.Column(c).Gap(8).Children(func() {
			for _, f := range choices {
				f := f
				name := f.FileName()
				if option(c, "fw-"+name, a.fwName == name, !a.updating, func() {
					ui.Box(c).Size(34, 34).Radius(8).Background(t.Surface).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
						ui.Icon(c, icDownload).FontSize(16).TextColor(t.TextMuted)
					})
					ui.Column(c).Gap(2).Grow(1).Children(func() {
						ui.Text(c, f.DisplayName()).FontWeight(600)
						if n := variantNote(f); n != "" {
							ui.Text(c, n).FontSize(12)
						}
						info := "Published " + f.Date
						if v, ok := a.versions.get(name); ok {
							info += " · " + v.String()
						}
						muted(c, info)
					})
					switch {
					case a.onDevice(f):
						pill(c, "On device", t.Success)
					case zl.FitFor(f, d.Name) == zl.FitOtherEdition:
						pill(c, "Different edition", t.Warning)
					}
				}) {
					a.fwName = name
				}
			}
		})
	})
}

// ---- 4. install ----

func (a *app) updateCard(c *ui.Context) {
	t := c.Theme()
	f, haveFW := a.chosenFirmware()
	a.card(c, 4, "Install", "", func() {
		switch {
		case a.updating:
			ui.Column(c).Gap(8).Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, a.status).Wrap().Grow(1)
					ui.Textf(c, "%.0f%%", a.progress*100).FontWeight(600).TextColor(t.Accent)
				})
				ui.Progress(c, a.progress).Label("Install progress")
			})
			return
		case a.done:
			a.successPanel(c)
			return
		}
		if a.failed != "" {
			msg := strings.ToUpper(a.failed[:1]) + a.failed[1:]
			if strings.Contains(msg, "timed out") || strings.Contains(msg, "disconnected") {
				msg += " Switch the page turner OFF and back ON, then try again — the update resumes where it stopped."
			}
			notice(c, "failed", t.Danger, icAlert, msg, nil)
		}
		ready := a.checkOK() && haveFW && a.busyLeft() == 0 && !a.readingVer
		label := "Install firmware"
		if haveFW {
			label = "Install " + f.DisplayName()
		}
		if ui.PrimaryButton(c.Key("go"), label).FillWidth().Height(40).Disabled(!ready).Clicked() {
			a.confirming = true
		}
		switch {
		case a.checking || a.check == nil:
			muted(c, "Waiting for the device check…")
		case !a.checkOK():
			muted(c, "The device check didn't pass.")
		case a.busyLeft() > 0:
			muted(c, fmt.Sprintf("Wait %d s — the page turner is still in update mode from reading its version.", int(a.busyLeft().Seconds())+1))
		case !haveFW:
			muted(c, "Choose a firmware file above.")
		default:
			muted(c, "Takes about 20 seconds. The page turner restarts by itself.")
		}
	})
}

func (a *app) successPanel(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Padding(18).Gap(10).Radius(12).AlignItems(ui.Center).
		Background(t.Success.Alpha(0.08)).Border(1, t.Success.Alpha(0.3)).Children(func() {
		ui.Icon(c, icCheckDisc).FontSize(36).TextColor(t.Success)
		ui.Text(c, "Update complete").FontSize(17).Bold()
		ui.Text(c, a.status).TextColor(t.TextMuted).TextAlign(ui.Center).Wrap()
		if a.version != nil {
			ui.Textf(c, "The page turner now reports %s.", a.version).TextColor(t.TextMuted)
		}
		ui.Text(c, "Last step: forget it on this Mac (System Settings › Bluetooth › ⓘ › Forget This Device), then pair it with your Kindle again.").
			TextAlign(ui.Center).Wrap()
		ui.Row(c).Gap(10).Children(func() {
			if ui.PrimaryButton(c.Key("bt"), "Open Bluetooth Settings").Clicked() {
				_ = mygo.Shell.OpenExternal(bluetoothSettings)
			}
			if ui.Button(c.Key("again"), "Done").Clicked() {
				a.done, a.progress, a.status = false, 0, ""
			}
		})
	})
}

// ---- log ----

func (a *app) logSection(c *ui.Context) {
	t := c.Theme()
	ui.Collapsible(c.Key("log"), "Technical log", &a.logOpen, func() {
		a.logMu.Lock()
		lines := append([]string(nil), a.logs...)
		a.logMu.Unlock()
		if a.logView.Y >= a.logView.MaxY-4 {
			a.logView.Y = 1e9
		}
		ui.Scroll(c.Key("logscroll")).TrackScroll(&a.logView).Height(200).Padding(10).Radius(8).
			Background(t.Background).Border(1, t.Border).Children(func() {
			for i, l := range lines {
				ui.Text(c.Key(fmt.Sprint("l", i)), l).Font("monospace").FontSize(11).Selectable()
			}
		})
	}).FontSize(12)
}
