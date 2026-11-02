package main

import (
	"os"
	"path/filepath"
	"regexp"
)

var plistVersion = regexp.MustCompile(`<key>CFBundleShortVersionString</key>\s*<string>([^<]+)</string>`)

// appVersion is the version of the .app bundle (from mygo.json via Info.plist), or "dev" when run with mygo dev.
func appVersion() string {
	exe, err := os.Executable()
	if err != nil {
		return "dev"
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "..", "Info.plist"))
	if err != nil {
		return "dev"
	}
	if m := plistVersion.FindSubmatch(b); m != nil {
		return "v" + string(m[1])
	}
	return "dev"
}
