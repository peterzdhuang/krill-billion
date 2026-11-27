package packbuild

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadRealDefs(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Join(filepath.Dir(file), "..", "..", "packs", "defs")
	defs, err := LoadDefs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 84 {
		t.Fatalf("defs = %d, want 84", len(defs))
	}
	seen := map[string]struct{}{}
	var countries, pokemon int
	for _, d := range defs {
		if _, ok := seen[d.ID]; ok {
			t.Fatalf("duplicate prompt id %s", d.ID)
		}
		seen[d.ID] = struct{}{}
		switch d.Check {
		case "restcountries":
			countries++
		case "pokeapi":
			pokemon++
		case "":
		default:
			t.Fatalf("prompt %s has unexpected check %q", d.ID, d.Check)
		}
	}
	if countries != 5 || pokemon != 5 {
		t.Fatalf("checks: restcountries=%d pokeapi=%d, want 5 and 5", countries, pokemon)
	}
}
