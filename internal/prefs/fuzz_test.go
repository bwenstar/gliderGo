package prefs

// Hostile bytes, on purpose (docs/IMPROVEMENTS.md 4.30).
//
// prefs.json is the player's own file, but it is edited by hand and written by other builds, and
// since 2.79 Save builds its bytes out of whatever Load read rather than only out of the struct.
// Plain `go test` runs FuzzLoadSave over its seeds; `make fuzz` runs the engine, and an input that
// fails is written to testdata/fuzz/FuzzLoadSave/ to be committed.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// FuzzLoadSave is a prefs.json from anywhere, loaded and saved.
//
// What must hold for any file:
//   - Load hands back settings and Save takes them, which is Load's whole contract;
//   - what Save wrote loads with no notes and gives back the same settings, so Validate is
//     idempotent, as its comment says;
//   - every key at the top of the file that no field has is in what Save wrote, as it was read;
//   - saved again, it is the same bytes, so a file does not drift a save at a time.
func FuzzLoadSave(f *testing.F) {
	def, err := json.MarshalIndent(Default(), "", "  ")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(def)
	f.Add([]byte(`{"version": 2, "house": "Teddy World", "Volume": 2,
		"fixes": {"mirror_flame": true, "a_fix_from_2031": true},
		"player1": "not an object any more", "an_option_from_2031": {"nested": [1, 2, 3]},
		"another": null}`))
	f.Add([]byte(`{"volume": 2, "house": "Teddy World"}`))
	f.Add([]byte(`{"volume": 3, "volume": "loud", "VOLUME": 4, "fixes": null, "Fixes": {"x": 1}}`))
	f.Add([]byte(`{"high_name": "éééééééééééééééééééé", "scale": -1, "pause_key": "q"}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[1, 2]`))
	f.Add([]byte(`{"house": "a"} trailing`))

	ours, err := json.Marshal(Default())
	if err != nil {
		f.Fatal(err)
	}
	known, _, _ := members(ours)

	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), Name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		p, _ := LoadFile(path)
		if p == nil {
			t.Fatal("LoadFile returned no settings")
		}
		if err := p.Save(); err != nil {
			t.Fatalf("Save refuses what LoadFile read: %v", err)
		}
		saved, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		q, err := LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(q.Notes) != 0 {
			t.Fatalf("what Save wrote loads with notes: %v\n%s", q.Notes, saved)
		}
		if !reflect.DeepEqual(normalize(p), normalize(q)) {
			t.Fatalf("the save changed the settings:\n saved %+v\n read  %+v", normalize(p), normalize(q))
		}

		var in, out map[string]json.RawMessage
		if err := json.Unmarshal(saved, &out); err != nil {
			t.Fatalf("Save wrote something that is not a JSON object: %v\n%s", err, saved)
		}
		if p.read != nil && json.Unmarshal(data, &in) == nil {
			for k, v := range in {
				if slicesContainFold(known, k) {
					continue
				}
				if !bytes.Equal(compact(t, v), compact(t, out[k])) {
					t.Fatalf("key %q was %s and is %s after a save", k, v, out[k])
				}
			}
		}

		if err := q.Save(); err != nil {
			t.Fatal(err)
		}
		again, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(again, saved) {
			t.Fatalf("a second save changed the file:\n first %s\n second %s", saved, again)
		}
	})
}

func slicesContainFold(keys []string, k string) bool {
	for _, o := range keys {
		if strings.EqualFold(o, k) {
			return true
		}
	}
	return false
}

func compact(t *testing.T, v json.RawMessage) []byte {
	if v == nil {
		return nil
	}
	var b bytes.Buffer
	if err := json.Compact(&b, v); err != nil {
		t.Fatalf("%s is not JSON: %v", v, err)
	}
	return b.Bytes()
}
