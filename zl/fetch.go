package zl

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Checked is a copy of a catalogue file as I downloaded and checked it, keyed by file name.
type Checked struct {
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

//go:embed verified.json
var checkedJSON []byte

// CheckedFiles lists the catalogue files as of the last check (size and SHA-256 of what the maker served).
var CheckedFiles = func() map[string]Checked {
	m := map[string]Checked{}
	if err := json.Unmarshal(checkedJSON, &m); err != nil {
		panic(err)
	}
	return m
}()

// Fetch downloads a firmware file from the maker's server, which is the source of truth: whatever it
// serves is used, as long as it's a complete .fot file. The returned note only says how it compares with
// the copy I checked; a difference means the maker updated the file and is not an error.
func Fetch(f Firmware) (data []byte, note string, err error) {
	name := f.FileName()
	if data, err = get(f.URL); err != nil {
		return nil, "", err
	}
	if p, _ := Inspect(data); p != "" {
		return nil, "", fmt.Errorf("%s: %s", name, p)
	}
	switch c, known := CheckedFiles[name]; {
	case !known:
		note = "not checked before"
	case sum(data) == c.SHA256:
		note = "identical to the checked copy"
	default:
		note = "differs from the checked copy (the maker updated it)"
	}
	return data, note, nil
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
