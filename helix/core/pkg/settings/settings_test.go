package settings

import "testing"

func TestScopePrecedence(t *testing.T) {
	r := &Resolver{
		Global: Store{"font_size_sp": 17.0, "theme": "helix_day_brand"},
		Device: Store{"font_size_sp": 19.0},
		Book:   Store{"theme": "helix_night_brand"},
	}
	if v, sc, _ := r.Get("font_size_sp"); v != 19.0 || sc != ScopeDevice {
		t.Fatalf("device must override global: %v %v", v, sc)
	}
	if v, sc, _ := r.Get("theme"); v != "helix_night_brand" || sc != ScopeBook {
		t.Fatalf("book must override all: %v %v", v, sc)
	}
	if _, _, ok := r.Get("missing"); ok {
		t.Fatal("missing key must report !ok")
	}
}

func TestClearOverrideFallsBack(t *testing.T) {
	r := &Resolver{Global: Store{"x": 1.0}, Device: Store{"x": 2.0}}
	r.Set(ScopeDevice, "x", nil)
	if v, sc, _ := r.Get("x"); v != 1.0 || sc != ScopeGlobal {
		t.Fatalf("clearing device override must fall back to global: %v %v", v, sc)
	}
}

func TestSetCreatesStores(t *testing.T) {
	r := &Resolver{}
	r.Set(ScopeBook, "margin_dp", 24.0)
	if v, sc, ok := r.Get("margin_dp"); !ok || v != 24.0 || sc != ScopeBook {
		t.Fatal("set on nil store must create it")
	}
}

func TestInvalidScopeHasDeliberateFailure(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != "settings: invalid scope" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()
	(&Resolver{}).Set(Scope(99), "key", "value")
}

func TestKeysSortedUnion(t *testing.T) {
	r := &Resolver{Global: Store{"b": 1, "a": 1}, Book: Store{"c": 1}}
	keys := r.Keys()
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("bad keys: %v", keys)
	}
}
