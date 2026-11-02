# Page Turner Updater for Mac

A free Mac app that updates the firmware of **智联纵维 (Zhilian Zongwei)** Bluetooth page turners
(HBTR001 / HBTR002 / HBTR003) **without the WeChat mini-program**.

The main use: Kindles released in 2024 or later, on Kindle software **5.20.1 or newer**, now accept Bluetooth page turners.
Many of these remotes ship with Kobo, Xteink or phone firmware and need a firmware update before a Kindle will
pair with them. The maker offers that update in a WeChat mini-program.
This app does the same update from a Mac, in English.

> Unofficial, and I'm not affiliated with the manufacturer. It downloads the maker's own firmware files from their public firmware
> list and sends them over the same Bluetooth update channel the mini-program uses. Use at your own risk.

## Disclosure

- I'm not affiliated with the maker, but I'm not neutral either: I run a Shopify store ([hqb.market](https://hqb.market))
  that sells their HBTR page turners, and I buy from them directly. I have no connection with the Amazon stores selling these remotes.
- I mostly test firmware switching on an HBTR002-XTE. I plan to test the HBTR001-K and the HBTR003-X next.
  If you have one of these and want to try before I do, you're welcome to, at your own risk, and please share the result.
- I'd like to add support for Hanlinyue (翰林阅) page turners too. I haven't looked into them yet; their updater
  seems to need a Windows PC, but a Mac version may be possible.

## Which remote do I have?

The maker makes three bodies. It names them only HBTR001, HBTR002 and HBTR003, and Amazon listings call them
different things; in my store they're **Mini**, **Ergo** and **Keycap**. Each body comes in several **editions**:
same shell, different firmware or chip batch. The edition is in the Amazon listing or on the box.

| <img src="docs/images/hbtr001.jpg" width="160" alt="HBTR001"> | <img src="docs/images/hbtr002.jpg" width="160" alt="HBTR002"> | <img src="docs/images/hbtr003.jpg" width="160" alt="HBTR003"> |
|:---:|:---:|:---:|
| **HBTR001 · Mini** | **HBTR002 · Ergo** | **HBTR003 · Keycap** |
| two round buttons | three round buttons, mode switch on the side | two mechanical keys, three LEDs, slide switch |

## Will it work with my Kindle?

**If your remote already shows up in the Kindle's Bluetooth list, you don't need this app.**

### HBTR001 · Mini

| Edition | On Amazon? | Kindle-ready? | Confidence | How I know |
|---|---|---|---|---|
| E (Kindle) | Coming soon (per the maker) | ✅ **Yes** | High | Sold by the maker as the Kindle edition |
| K (Kobo) | Yes | 🔧 **Needs update** → `HBTR001-E (10.30)` | High | Reddit users did it with the official WeChat mini-program ([example](https://www.reddit.com/r/kindle/comments/1xl9tiz/comment/19rzosvw6l1l0/)); the maker confirms K → E. Not yet tested with this app |
| Xteink: Amazon "Bluetooth Page Turner Clicker for XTEINK X4 E-Book Reader", shows up as `XT01` | Yes | ❌ **No** | High | The maker says no: it's Xteink-only hardware. The Amazon listing itself says to pair with "XT01". [A Reddit user](https://www.reddit.com/r/kindle/comments/1xam01u/comment/19s0igvdvu2de/) found no firmware for it in the official mini-program |

### HBTR002 · Ergo

| Edition | On Amazon? | Kindle-ready? | Confidence | How I know |
|---|---|---|---|---|
| E (Kindle) | Coming soon (per the maker) | ✅ **Yes** | Medium | The maker's announcement; not out yet |
| XTE (3 modes) | No (sold in China only, AFAIK) | ✅ **Yes** | High | Tested by me: pairs with a Kindle |
| XT (older Xteink edition) |  | 🔧 **Needs update** → `HBTR002-XTE (10.30)` | Medium | The maker's reply to an HBTR002-XT owner: install 002-XTE in the mini-program |

### HBTR003 · Keycap

| Edition | On Amazon? | Kindle-ready? | Confidence | How I know |
|---|---|---|---|---|
| E (Kindle) | Coming soon (per the maker) | ✅ **Yes** | Medium | The maker's announcement; not out yet |
| XTE (3 modes) | No (sold in China only, AFAIK) | ✅ **Yes** | High | Tested by me: pairs with a Kindle out of the box |
| X (phone & tablet / "TikTok") | Yes | 🔧 **Needs update** → `HBTR003-XTE (10.30)` | Medium | The maker's China store says HBTR003-X can be upgraded to HBTR003-XTE. The Amazon unit shows up as `HBTR003-X` too, but it isn't confirmed to be the same product. There's no HBTR003-X firmware to go back to, so you'd lose the phone swipe modes. **Have one? Please try it and tell me** (see the end of this page) |

Kindle side: only Kindles released in 2024 or later, plus the 2022 Kindle Scribe (1st gen), with Kindle software 5.20.1+
accept these remotes. Jailbroken Kindles on older software need Bluetooth/key-mapping plugins instead.

### 2025 or 2026?

Most firmware, including all the Kindle-capable files (E and XTE), comes in a `-2025` and a `-2026` file. The two contain the same program, built for two production
batches of the chip. You can't tell the batch from the outside, and Amazon stock can be old.
**Try 2025 first; if the update stops right away with "key mismatch", install 2026.** A wrong batch is refused
before anything is written, so it doesn't harm the remote.

### What changes after the update

- **HBTR001-E is Kindle-only.** It stops working with other readers. You can flash `HBTR001-K` back at any time.
- **HBTR002/HBTR003-XTE** has 3 modes on the slide switch: **position marked up/down (↕) = Kindle**, left/right (↔) = Xteink, volume = Android
  readers (volume-key page turning). Switch to the Kindle position before pairing with the Kindle.
- **iPhone/iPad and TikTok-style swiping**: the XTE firmware has no touch-swipe mode. The phone edition (`HBTR002-X`)
  swipes (up/down for short videos, left/right for reading); per the maker's manual, iOS needs
  *Settings › Accessibility › Zoom* turned on for that.

## Using the app

1. Switch the remote on. If it's paired to a phone, Kindle or Mac, **forget it there first**. It only shows up while it isn't connected to anything.
2. Open the app and choose your remote. *Check* is read-only.
3. Choose the firmware from the tables above.
4. Click *Install*. It takes about 20 seconds; keep the remote next to the Mac and don't press its buttons.
   The remote restarts by itself.
5. Pair it with your Kindle: *Settings › Bluetooth* on the Kindle, remote in the Kindle position.

If macOS shows "Connection Request from …", either button is fine. Afterwards you can remove the remote from the Mac's
Bluetooth list.

## Download

Get the latest `page-turner-updater-<version>.dmg` from the [Releases](../../releases) page.
It runs on any Mac with macOS 12 or later, Apple Silicon or Intel.

1. Open the .dmg and drag **Page Turner Updater** into *Applications*.
2. The app isn't signed with an Apple developer certificate, so the first launch is blocked.
   Open it once, then go to *System Settings › Privacy & Security*, scroll down, and click **Open Anyway** next to
   "Page Turner Updater was blocked". (On macOS 14 and earlier you can also right-click the app and choose *Open*.)
3. Allow Bluetooth access when macOS asks.

## Build from source

With Go 1.27+:

```sh
go tool mygo build -platform darwin/universal   # → build/darwin-universal/Page Turner Updater.app + .dmg
```

Development:

```sh
go tool mygo dev     # run with live reload
go test ./...        # protocol tests (simulated device) + headless UI tests
go build -o bin/blecheck ./cmd/blecheck && bin/blecheck HBTR   # read-only Bluetooth check from the terminal
```

No cgo: [MyGo](https://github.com/egoist/mygo) native UI, CoreBluetooth through
[purego](https://github.com/ebitengine/purego)'s Objective-C runtime.

| Package | What |
|---|---|
| `zl` | protocol: firmware list, firmware/hardware matching, frame builder, update state machine |
| `ble` | CoreBluetooth central (scan, connect, notify, flow-controlled writes) |
| `main.go`, `view.go` | the UI: choose device → check → choose firmware → install |

How the update protocol works: [docs/PROTOCOL.md](docs/PROTOCOL.md).

## Versions

The current version is shown next to the app's name, and each release is listed on the [Releases](../../releases) page.
The firmware itself isn't versioned by this app: it always offers the maker's current files.

## License

The app's code is under the [MIT License](LICENSE). The firmware files belong to the maker; they aren't part of this
repository and are downloaded from the maker's server when you install them.

## Help me fill in the table

If you update a remote marked ❔ Unknown or Medium confidence, please open an issue with:
- the Bluetooth name before and after,
- which file you installed (2025 or 2026) and whether it worked,
- where you bought it (an Amazon link helps).
