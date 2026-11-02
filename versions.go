package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"pageturner-updater/zl"
)

// versionDB remembers which firmware version (INFO_DEV_VER) each catalogue build reports once flashed.
// The catalogue has no real version numbers, and the version string sits in the encrypted part of the file,
// so the only way to learn it is to read the device after an update. Keyed by zl.BuildKey (year ignored).
type versionDB struct {
	mu   sync.Mutex
	path string
	m    map[string]uint16
}

// Seen on real hardware (2026-11-02).
var seedVersions = map[string]uint16{
	"HBTR002-XTE(10.30)": 0x1090, // V1.0.9
	"HBTR002-X(11.01)":   0x1070, // V1.0.7 — the same number as the XTE device's factory firmware: version
	// numbers are per edition, not unique across editions
}

func openVersionDB(path string) *versionDB {
	db := &versionDB{path: path, m: map[string]uint16{}}
	for k, v := range seedVersions {
		db.m[k] = v
	}
	if path == "" {
		return db
	}
	if b, err := os.ReadFile(path); err == nil {
		var saved map[string]uint16
		if json.Unmarshal(b, &saved) == nil {
			for k, v := range saved {
				db.m[k] = v
			}
		}
	}
	return db
}

func defaultVersionDBPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Page Turner Updater", "versions.json")
}

func (db *versionDB) get(fileName string) (zl.Version, bool) {
	db.mu.Lock()
	defer db.mu.Unlock()
	c, ok := db.m[zl.BuildKey(fileName)]
	return zl.Version{Code: c}, ok
}

func (db *versionDB) set(fileName string, code uint16) error {
	db.mu.Lock()
	db.m[zl.BuildKey(fileName)] = code
	b, _ := json.MarshalIndent(db.m, "", "  ")
	path := db.path
	db.mu.Unlock()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// buildsWithVersion lists the builds among files known to report code. Callers should pass only files of the
// device's current edition, since editions reuse version numbers (X 11.01 and an older XTE are both V1.0.7).
func (db *versionDB) buildsWithVersion(files []zl.Firmware, code uint16) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range files {
		if v, ok := db.get(f.FileName()); ok && v.Code == code {
			k := zl.BuildKey(f.FileName())
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}
