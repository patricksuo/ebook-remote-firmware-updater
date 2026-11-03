# Firmware list

Every file in the maker's firmware list, as I downloaded and checked it on **2026-11-03**.
Source: `https://diykey-1426533931.cos.ap-guangzhou.myqcloud.com/firmware_manifest.json`
(the same list the official WeChat mini-program uses). The app downloads from there; it doesn't ship these files.

- **Date**: the `date` field in the maker's list (when the file was put online). `(10.30)`, `(11.01)` in a name are the maker's build labels.
- **Size** and **SHA-256**: of the file actually served. The `size` field in the maker's list is wrong for many files, so I don't use it.
- **Batch**: the batch key in the file header (offset `0x1E`): `D9EE331E6B04` = 2025, `7E2B167668E0` = 2026. A remote refuses a file with the other batch's key ("key mismatch") and nothing is installed.
- The list's `version` field is `v1.0.0` for every file and its `description` is just a number (`固件1` … `固件43`), so neither is shown.
- The app has these hashes built in (`zl/verified.json`). If the maker replaces a file, the app still installs it and notes the difference in the Technical log.

To check a file yourself: `shasum -a 256 <file>`.


## HBTR001 · Mini

| File | Date | Size (bytes) | Batch | Notes |
|---|---|---:|---|---|
| `HBTR001-2025-长按小按键5秒重置蓝牙.fot` | 2026-08-03 | 93,696 | 2025 | Android reader edition (Boox, iReader, Hanvon, Bigme, Hisense…, Android phones). Not Kindle, Xteink or iOS. “长按小按键5秒重置蓝牙” = hold the small button 5 s to reset Bluetooth |
| `HBTR001-2026-长按小按键5秒重置蓝牙.fot` | 2026-08-03 | 93,696 | 2026 | Android reader edition (Boox, iReader, Hanvon, Bigme, Hisense…, Android phones). Not Kindle, Xteink or iOS. “长按小按键5秒重置蓝牙” = hold the small button 5 s to reset Bluetooth |
| `HBTR001-E-2025(10.30).fot` | 2026-10-30 | 95,232 | 2025 | Kindle edition |
| `HBTR001-E-2026(10.30).fot` | 2026-10-30 | 95,232 | 2026 | Kindle edition |
| `HBTR001-K-2025-长按小按键5秒重置蓝牙.fot` | 2026-08-03 | 94,208 | 2025 | Kobo edition. “长按小按键5秒重置蓝牙” = hold the small button 5 s to reset Bluetooth |
| `HBTR001-K-2025-长按小按键锁屏.fot` | 2026-08-06 | 93,696 | 2025 | Kobo edition. “长按小按键锁屏” = hold the small button to lock the screen |
| `HBTR001-K-2026-长按小按键5秒重置蓝牙.fot` | 2026-08-03 | 94,208 | 2026 | Kobo edition. “长按小按键5秒重置蓝牙” = hold the small button 5 s to reset Bluetooth |
| `HBTR001-K-2026-长按小按键锁屏.fot` | 2026-08-06 | 93,696 | 2026 | Kobo edition. “长按小按键锁屏” = hold the small button to lock the screen |
| `HBTR001-XT-2025-长按小按键5秒重置蓝牙.fot` | 2026-08-03 | 97,280 | 2025 | Xteink (阅星瞳) edition. Not for the Xteink-only `XT01` remote. “长按小按键5秒重置蓝牙” = hold the small button 5 s to reset Bluetooth |
| `HBTR001-XT-2026-长按小按键5秒重置蓝牙.fot` | 2026-08-03 | 97,280 | 2026 | Xteink (阅星瞳) edition. Not for the Xteink-only `XT01` remote. “长按小按键5秒重置蓝牙” = hold the small button 5 s to reset Bluetooth |

<details><summary>SHA-256</summary>

```
2caa767b467d2c9085de76a1b854025adbd1f63a7d1214b75a629e3a8f7e86ca  HBTR001-2025-长按小按键5秒重置蓝牙.fot
c0ee50c5cb35a1b201d3dce5cc07426f36604c72add5c187c9962545de47244c  HBTR001-2026-长按小按键5秒重置蓝牙.fot
83235b6b6d1a9117dfedf966ea31a7039f5c69d23fe9b41d80f97209ce5642ae  HBTR001-E-2025(10.30).fot
8e2aac45d05b86fc6ad126b3ba762cb1c6aebcd4b869641cf5829e568a827f37  HBTR001-E-2026(10.30).fot
e49b6426cc5990993bc6f86f7d1fb50f364de6a0c192c70678e65f54cc6e939c  HBTR001-K-2025-长按小按键5秒重置蓝牙.fot
3acafdbc0a07c3d08d57d0dd9f3b1981b29b3d577dd6a1ce89115b204cbf4f3d  HBTR001-K-2025-长按小按键锁屏.fot
ea1e53500191502d9ab174c532ea146c62de23e928a4f8f58b1dadbe7c4e4a4c  HBTR001-K-2026-长按小按键5秒重置蓝牙.fot
c089d0ed65779d1ef140fd1fc1070dd48a55a82b140d0c3f1c2c9606f161b996  HBTR001-K-2026-长按小按键锁屏.fot
c5f710dda5f3b95cc7aa514253da65d4e9b0ff81fdded131b2651d11b9dd38e0  HBTR001-XT-2025-长按小按键5秒重置蓝牙.fot
836cc2f27c92dba023b230686ad5d1e021fc67245068ff0ebbbe75ddeac086de  HBTR001-XT-2026-长按小按键5秒重置蓝牙.fot
```

</details>

## HBTR002 · Ergo

| File | Date | Size (bytes) | Batch | Notes |
|---|---|---:|---|---|
| `HBTR002-KD.fot` | 2026-09-03 | 93,696 | 2026 | iFlytek (讯飞) edition. No year in the name; the batch key is 2026 |
| `HBTR002-X-2025(11.01).fot` | 2026-11-01 | 101,376 | 2025 | Phone & tablet edition: ↕ short-video swipes, ⇄ reading swipes, volume |
| `HBTR002-X-2026(11.01).fot` | 2026-11-01 | 101,376 | 2026 | Phone & tablet edition: ↕ short-video swipes, ⇄ reading swipes, volume |
| `HBTR002-XTE-2025(10.30).fot` | 2026-10-26 | 97,280 | 2025 | 3 modes: ↕ Kindle, ↔ Xteink, volume = Android readers |
| `HBTR002-XTE-2026(10.30).fot` | 2026-10-26 | 97,280 | 2026 | 3 modes: ↕ Kindle, ↔ Xteink, volume = Android readers |

<details><summary>SHA-256</summary>

```
6550343ee7732ec655a52d8518a74df0393064b02a3b1e4b684a3bb6099bf6db  HBTR002-KD.fot
fcbf18281f386f733d94b70d44d9a890ca2c1be5adef138164d9c2c94aecc33c  HBTR002-X-2025(11.01).fot
8a2aac8ea8d83aaac0e6b1bd3af69ad073f0d9375ec2f1a8f9d0c8768764c17a  HBTR002-X-2026(11.01).fot
bc4d715c098ae728a721605df0e6c246d529f1ddbf755605aacb5b9a27dc3d8a  HBTR002-XTE-2025(10.30).fot
bdf94acacbd543f5a72c79e86ccc7b2ab7c73a7a6decdaede0ae105c1a5f6d12  HBTR002-XTE-2026(10.30).fot
```

</details>

## HBTR003 · Keycap

| File | Date | Size (bytes) | Batch | Notes |
|---|---|---:|---|---|
| `HBTR003-KD-2026.fot` | 2026-08-03 | 97,280 | 2026 | iFlytek (讯飞) edition. 2026 batch only |
| `HBTR003-XTE-2025(10.30).fot` | 2026-10-30 | 96,768 | 2025 | 3 modes: ↕ Kindle, ↔ Xteink, volume = Android readers |
| `HBTR003-XTE-2026(10.30).fot` | 2026-10-30 | 96,768 | 2026 | 3 modes: ↕ Kindle, ↔ Xteink, volume = Android readers |

<details><summary>SHA-256</summary>

```
b8a8bdf61a716ac39ad044e9728ad6d97210e41042f7166e3feac5bc4b24431a  HBTR003-KD-2026.fot
9de15bdd77cea8d6900c7d54b2166ca64467de000d6806751ef89edca395fd53  HBTR003-XTE-2025(10.30).fot
bc48a76c0580d0ce48b43c9ca62d00c96e201761b4e561c92327e92cbc69de31  HBTR003-XTE-2026(10.30).fot
```

</details>

## Smart 1 (the maker's 3-key keycap remote; this app doesn't support it yet)

| File | Date | Size (bytes) | Batch | Notes |
|---|---|---:|---|---|
| `Smart 1-EMM.fot` | 2026-09-19 | 100,352 | other (`4F4FDD6309BB`) |  |
| `Smart 1-EMM新版本.fot` | 2026-10-12 | 103,936 | other (`4F4FDD6309BB`) |  |
| `Smart 1-EMP-Kindle-20261030.fot` | 2026-10-31 | 101,888 | other (`4F4FDD6309BB`) |  |
| `Smart 1-EMP-kindle新版本-261030.fot` | 2026-10-30 | 104,448 | other (`4F4FDD6309BB`) |  |
| `Smart 1-EMP.fot` | 2026-08-13 | 100,352 | other (`4F4FDD6309BB`) |  |
| `Smart 1-EMP新版本.fot` | 2026-10-14 | 104,448 | other (`4F4FDD6309BB`) |  |

<details><summary>SHA-256</summary>

```
b64091d67ff6b2c1ed531aaf170d7cc531614a48129179cc71706ddce88f218b  Smart 1-EMM.fot
a71bef680fb1a65ce9c6ff113d542294f9aa40df4263d5d2beb95204c95d76ae  Smart 1-EMM新版本.fot
53746de3279e75b779c74b1e93a77c241ed9160736654dde191678f143ab6932  Smart 1-EMP-Kindle-20261030.fot
03489bb041958dadae8dd5bb11d8d370d165697714b9ab0e9cb6e6a09447c597  Smart 1-EMP-kindle新版本-261030.fot
ed6f8ed6b0c50cfd8f101850b3411ccf045f13ee79741f27141915c7b8b6e1ce  Smart 1-EMP.fot
5d3bdd68773ad8b25ea59898f53abb1381bb47f9df9b93629a647b19e36e55d3  Smart 1-EMP新版本.fot
```

</details>

## Not in the list

- **HBTR003-X**: no firmware. Updating an HBTR003-X to HBTR003-XTE can't be undone.
- **HBTR002-E, HBTR003-E**: the maker's upcoming Kindle editions; no files yet.
