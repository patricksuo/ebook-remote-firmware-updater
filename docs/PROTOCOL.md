# Update protocol

The page turners use a Bluetrum (中科蓝讯) BLE chip. Updates go through the chip SDK's standard **FOT**
("firmware over the air") module. Everything here was checked against real HBTR002 hardware.

## Firmware list

`GET https://diykey-1426533931.cos.ap-guangzhou.myqcloud.com/firmware_manifest.json`

```json
{"firmwares": [{"version": "v1.0.0", "date": "2026-10-30", "url": "https://…/HBTR002-XTE-2026(10.30).fot",
                "size": 97280, "description": "…"}]}
```

The `version` field is not meaningful. The file name is what identifies a firmware. The official mini-program shows a file
when its name starts with the device's Bluetooth name minus the last `-…` part (`HBTR002-XTE3` → `HBTR002`).
It applies no other filter.

## GATT

| Service | Characteristic | Use |
|---|---|---|
| `FF12` | `FF15` write without response | commands and data |
| `FF12` | `FF14` notify | replies |

macOS negotiates ATT MTU 512, so 509-byte writes fit.

## Sequence

All bytes in hex. → = write to FF15, ← = notification on FF14.

| Step | Send | Expect |
|---|---|---|
| auth + info | → `CC AA 55 EE 12 19 E4`, then → `92 00 01 00 03 00 04 00 06 00` | ← `92 …` (TLV; param `01` = installed version) |
| version | → `91 01 01` | ← `91 01 01 <LE16 version>` |
| update request | → `91 02 02 FF FF 06 8E 3D 19` | ← `91 02 02 <LE32 start offset> … <allow>`; allow `01` = go |
| data | frames (below), one per ack | ← `90 xx yy` |

**Version encoding:** the firmware's `SW_VERSION` string `"Va.b.c"` becomes `a<<12 | b<<8 | c<<4`, so `0x1090` = V1.0.9.

### Frames

Each frame carries 512 bytes of the file. Counters start at `zz = 3`, `endZz = 4`, `seq = 0`, `offset = start offset`.

1. Data packet, 509 bytes: `A0 zz 00 seqLo seqHi 00 00 02 00 00` + `file[offset : offset+499]`, padded with `FF`.
   Then `zz += 2` (wraps at 256) and `seq += 2`.
2. Tail packet, 15 bytes, sent right after: `20 endZz` + `file[offset+499 : offset+512]`, padded with `FF`.
   Then `endZz += 2` and `offset += 512`.
3. Wait for `90 xx yy`:

| `yy` | Meaning |
|---|---|
| `00` | OK, send the next frame |
| `FF` | done; the device reboots |
| `01` | same firmware as installed |
| `02` | **key mismatch**: file built for another chip batch (e.g. a 2025 file on a 2026 unit). Rejected, nothing installed |
| `0B` | CRC error |
| `40` | sequence error, also seen when the device is still in transfer mode from an earlier session |
| `41` | length error |

**Gotcha:** the auth + info step alone already puts the device in transfer mode. It leaves that mode by itself
within about 60 s, or when switched off and on. Run the whole sequence in one connection.

## `.fot` file header

| Offset | Size | Meaning |
|---|---|---|
| 0x00 | 4 | `POT\0` |
| 0x04 | 4 | file length − 8 |
| 0x18 | 2 | build fingerprint (the same in the 2025 and 2026 files of one build) |
| 0x1E | 6 | **batch key**: `D9 EE 33 1E 6B 04` = 2025, `7E 2B 16 76 68 E0` = 2026 |

The 2025 and 2026 files of the same firmware differ only in the batch key and a few header checksums.
The program itself is encrypted and identical in both.

## Modes of the HBTR002-XTE firmware

Each switch position advertises with its own Bluetooth name and address:

| Switch | Mode | Bluetooth name | Acts as |
|---|---|---|---|
| ↕ (up/down) | Kindle | `HBTR002-XTE3` | keyboard PageDown / PageUp, brightness ± (Amazon company ID `0x0171`) |
| volume | Android readers | `HBTR002-XT` | Volume − / +, Back |
| ↔ (left/right) | Xteink readers | `HBTR002-XTE` | (nothing visible to a Mac) |

## Modes of the HBTR002-X firmware (phone & tablet edition)

Bluetooth name `HBTR002-X`, Apple vendor ID (`0x05AC`, PID `0x0220`). To the phone it is a **consumer-control device plus a
single-finger touchscreen**, with no keyboard and no mouse. The swipe modes simulate a finger on the screen
(coordinates 0–1000):

| Switch | LED | Button 1 | Button 2 | Button 3 | For |
|---|---|---|---|---|---|
| ↕ (up/down) | green | swipe **up** = next video (center, Y 560 → 0) | swipe **down** = previous video (Y 345 → 1000) | pause/play; double press = like; hold = home | short videos (TikTok etc.) |
| ⇄ (left/right) | blue | swipe **left** = next page (middle, X 700 → 0) | swipe **right** = previous page (X 300 → 1000) | hold = home | reading apps on iPhone/iPad that turn pages by swiping |
| 🔈 (volume) | green | **Volume −** = next page | **Volume +** = previous page | back; double press = home; hold 3 s = lock | Android readers with volume-key page turning; camera shutter |

Swipe coordinates are what I measured; button meanings and LED colors are from the maker's manual for HBTR002-X.
The volume mode works on iPhone and iPad as is, and the swipe modes work on Android directly. **On iOS the swipe modes
need *Settings › Accessibility › Zoom* turned on** (maker's manual; a three-finger double tap resets the zoom if the
screen gets magnified).

LED on all HBTR002 editions (maker's manual): blinking = waiting to pair, off = connected; green/blue = battery OK,
red = low battery.
