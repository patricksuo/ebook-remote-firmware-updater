package main

import (
	"strings"

	"pageturner-updater/zl"
)

// variantNotes describes what a firmware variant is for, keyed by "<series>-<variant>" as in the file names.
// The catalogue itself has no descriptions; these come from product pages and the maker's public posts.
var variantNotes = map[string]string{
	"HBTR001-K":   "Kobo readers (Kobo/Kindle hardware: can switch to HBTR001-E)",
	"HBTR001-XT":  "Xteink (阅星瞳) readers (separate hardware: only XT firmware)",
	"HBTR001-E":   "Kindle only: 2024+ Kindles on 5.20.1+, and the 2022 Kindle Scribe (Kobo/Kindle hardware: can switch to HBTR001-K)",
	"HBTR001":     "Android readers (Boox, iReader, Hanvon, Bigme, Hisense…), Android phones and tablets. Not Kindle, Xteink or iOS",
	"HBTR002-KD":  "iFlytek (科大讯飞) readers",
	"HBTR003-KD":  "iFlytek (科大讯飞) readers",
	"HBTR003-XTE": "3 modes: Kindle (2024+ Kindles, software 5.20.1+), Android (volume keys), Xteink readers",
	"HBTR002-X":   "Phone & tablet edition: iOS, Android, HarmonyOS; e-ink only Hanvon, Boox, Bigme, Hisense. No Kindle mode",
	"HBTR002-XTE": "3 modes: Kindle (2024+ Kindles, software 5.20.1+), Android (volume keys), Xteink readers",
}

// variantNote returns the note for a catalogue file, matching the longest "<series>-<variant>" prefix that is
// followed by "-", "(" or the end ("HBTR002-X" must not match "HBTR002-XTE-…").
func variantNote(f zl.Firmware) string {
	name := strings.ToUpper(f.DisplayName())
	best, note := 0, ""
	for k, v := range variantNotes {
		k = strings.ToUpper(k)
		if !strings.HasPrefix(name, k) || len(k) <= best {
			continue
		}
		if rest := name[len(k):]; rest == "" || rest[0] == '-' || rest[0] == '(' {
			best, note = len(k), v
		}
	}
	return note
}
