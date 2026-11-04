# Updating with the maker's official WeChat mini-program (iPhone / Android)

No Mac? You can do the same update with the maker's own updater: the **智联纵维 (Zhilian Zongwei)** mini-program
inside the WeChat app. It installs the same firmware files from the same server as the Mac app; only the route differs.

What you need: a phone with WeChat, your remote (charged and switched on), about 10 minutes the first time.
The screenshots below are from an iPhone in English, updating an HBTR002 (Ergo) to `HBTR002-XTE-2026`. Your remote's
firmware list will show its own files (`HBTR001-…` for a Mini, `HBTR003-…` for a Keycap).

**First check which firmware you need** in the README tables:
[Will it work with my Kindle?](../README.md#will-it-work-with-my-kindle). `XT01` remotes can't be updated at all.

## 1. Get a WeChat account

Skip this if you already use WeChat.

| | | |
|:---:|:---:|:---:|
| <img src="images/wechat/01-signup.jpg" width="220" alt="WeChat welcome screen: Log In, Sign Up"> | <img src="images/wechat/02-method.jpg" width="220" alt="Select Sign Up Method: Mobile, Apple, Facebook"> | <img src="images/wechat/03-mobile.jpg" width="220" alt="Sign Up with Mobile: name, region, mobile number, password"> |
| Install WeChat from the App Store or Google Play, tap **Sign Up** | Pick any method. Even with **Sign up with Apple**, WeChat still asks for a phone number next | Set **Region** to your country first (it starts on *China's Mainland*), then your mobile number and a password. Tick the agreement, **Accept and Continue** |

WeChat then texts you a code. It may also run a security check; for some new accounts that means an existing
WeChat user has to scan a QR code to vouch for you. If you get stuck there and don't know anyone on WeChat, use the
[Mac app](../README.md) instead.

## 2. Open the 智联纵维 mini-program

| | |
|:---:|:---:|
| <img src="images/wechat/04-search.jpg" width="220" alt="WeChat search for zhilianzongwei, top hit 智联纵维 Mini Program"> | <img src="images/wechat/05-home.jpg" width="220" alt="Mini-program home page, translated to English"> |
| In WeChat, tap the **search** icon and type `zhilianzongwei` (or `智联纵维`). Tap the top hit marked **Mini Program**, with the ZLINK logo | The home page is in Chinese; in my recording WeChat translated it to English by itself. It lists the same steps as below |

## 3. Connect the remote

Before you start:
- Turn on **Bluetooth and Location** on the phone, and allow WeChat to use both when it asks.
- **Forget the remote** wherever it's paired (Kindle, phone, Mac) and switch it OFF and ON: it only shows up while
  it isn't connected to anything. Its LED blinks when it's ready.

| | |
|:---:|:---:|
| <img src="images/wechat/06-devices.jpg" width="220" alt="Searching for devices: HBTR002-X in the list"> | <img src="images/wechat/07-pairing.jpg" width="220" alt="iPhone Bluetooth Pairing Request with Cancel and Pair"> |
| On the **home page** tab, tap **Search for Bluetooth**, then tap your remote (`HBTR001…`, `HBTR002…` or `HBTR003…`) | You'll see *Device Name* and *Unpair* when it's connected. **iPhone: if a "Bluetooth Pairing Request" pops up, tap Cancel.** If you pair it in iOS, the update fails |

## 4. Install the firmware

| | | |
|:---:|:---:|:---:|
| <img src="images/wechat/08-firmware.jpg" width="220" alt="Flash firmware tab: list of HBTR002 firmware files"> | <img src="images/wechat/09-confirm.jpg" width="220" alt="Confirm upgrade dialog: HBTR002-XTE-2026, 95.0 KB"> | <img src="images/wechat/10-progress.jpg" width="220" alt="Currently upgrading progress bar"> |
| Open the **Flash firmware** tab at the bottom. It lists the files for your remote (the translation says *Optional fasteners*; it means *available firmware*). Tap the one you need | Check the name, then **Confirm upgrade** | Keep the phone next to the remote and don't press its buttons. It took about 35 seconds here |

<img src="images/wechat/11-done.jpg" width="220" alt="Upgrade successful dialog">

**Upgrade successful**: the remote restarts by itself. Tap OK (the translation says *determine*).

### 2025 or 2026?

Most files come in a `-2025` and a `-2026` version for two production batches of the chip; you can't tell them apart
from the outside. **Try 2025 first.** If the update stops right away with a *KEY mismatch* error
(文件包返回错误，KEY不匹配), nothing was installed: install the 2026 file instead. If both are refused, your remote is
a different edition than you think; check the README tables.

## 5. Pair it with your Kindle

1. In the mini-program, tap **Unpair**, or close WeChat, so the remote is free again.
2. **HBTR002 / HBTR003 (XTE firmware):** set the slide switch to the **up/down (↕)** position. That's Kindle mode.
3. On the Kindle: *Settings › Bluetooth*, then pair the remote.

The Kindle needs software **5.20.1.0.2 or later** and a supported model; see
[Which Kindles work](../README.md#will-it-work-with-my-kindle).

## If something goes wrong

- **The remote isn't in the list.** It's still connected to something: turn off Bluetooth on your Kindle or other
  phone for a moment, switch the remote OFF and ON, and search again. If the search area is blank, close and reopen
  the mini-program.
- **The update stopped halfway.** Switch the remote OFF and ON, connect again and install the same file again.
- **HBTR002: "Bluetooth reset".** The Flash firmware page also shows the maker's reset tip: press and hold the
  center button while moving the side switch (but not to OFF).
- **The Kindle doesn't find the remote.** Check the Kindle's model and software, the switch position (↕), and that the
  remote isn't still connected to the phone.

Questions, or a remote that isn't in the README tables yet? [Open an issue](https://github.com/patricksuo/ebook-remote-firmware-updater/issues).
