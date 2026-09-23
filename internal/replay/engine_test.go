package replay

import (
	"os"
	"reflect"
	"runtime"
	"testing"
)

func TestEngineNeedsATree(t *testing.T) {
	if _, err := Engine(nil); err == nil {
		t.Fatal("Engine(nil) returned a fingerprint; with no tree there is nothing to fingerprint")
	}
}

// The fingerprint is what two machines compare, and two machines have different numbers of
// cores. The runs go side by side, so a run that leaned on something another run left behind --
// a cache, a table, a stream -- would give one number on a laptop and another on a desktop, and
// refuse races between two copies of one build. One core, which runs them one after another,
// and every core agree.
func TestEngineIsTheSameOnOneCoreAsOnMany(t *testing.T) {
	const root = "../../assets/extracted" // the tree assets.Tree() embeds, which this cannot import
	if _, err := os.Stat(root); err != nil {
		t.Skipf("no extracted assets at %s: run `make assets`", root)
	}
	tree := os.DirFS(root)
	many, err := Engine(tree)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	one, err := Engine(tree)
	if err != nil {
		t.Fatal(err)
	}
	if one != many {
		t.Fatalf("the fingerprint is %016X on one core and %016X on %d", one, many, runtime.NumCPU())
	}
}

// Every field of a Sample is either something a race can feel, and in the fingerprint, or not,
// and out of it -- and a field added to Sample has to be put on one side or the other here
// before this passes. A field that is in has to move the hash, and one that is out must not.
func TestEverySampleFieldIsInOrOutOfTheFingerprint(t *testing.T) {
	in := map[string]bool{
		"Frame": true, "Even": true, "Room": true, "Mode": true, "Dest": true, "Score": true,
		"Mortals": true, "Stars": true, "Rand": true,
	}
	out := map[string]bool{
		"Work2Main": true, "Back2Work": true, "Renders": true, "Pendulums": true,
		"ClockFrame": true, "Guarded": true, "Dropped": true, "Sounds": true,
		// A race is one-player on each side, and so are Engine's runs.
		"Two": true, "Mode2": true, "Dest2": true, "Escaped": true, "ArectEscaped": true,
		"First": true, "OneLeft": true, "Dead": true,
	}

	base := string(Sample{}.simulation(nil))
	typ := reflect.TypeOf(Sample{})
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if in[f.Name] == out[f.Name] {
			t.Errorf("Sample.%s is new: decide whether a race can feel it, and put it in "+
				"simulation or leave it out, and list it here either way", f.Name)
			continue
		}
		// A struct field is changed one part at a time, so that Dest's four sides are each
		// checked and not just the first.
		parts := []int{-1}
		if f.Type.Kind() == reflect.Struct {
			parts = parts[:0]
			for j := 0; j < f.Type.NumField(); j++ {
				parts = append(parts, j)
			}
		}
		for _, j := range parts {
			var s Sample
			v := reflect.ValueOf(&s).Elem().Field(i)
			name := f.Name
			if j >= 0 {
				v, name = v.Field(j), f.Name+"."+f.Type.Field(j).Name
			}
			switch v.Kind() {
			case reflect.Bool:
				v.SetBool(true)
			case reflect.String:
				v.SetString("x")
			default:
				v.SetInt(1)
			}
			moved := string(s.simulation(nil)) != base
			switch {
			case in[f.Name] && !moved:
				t.Errorf("Sample.%s is meant to be in the fingerprint and does not move it", name)
			case out[f.Name] && moved:
				t.Errorf("Sample.%s is meant to be out of the fingerprint and moves it", name)
			}
		}
	}
}
