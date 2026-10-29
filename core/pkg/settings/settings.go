// Package settings implements the scoped settings resolution model from
// docs/29 (Settings Configurability Matrix): values live at global, device,
// and per-book scope; effective value = most specific non-nil scope, with a
// deterministic merge order global < device < book.
package settings

import "sort"

// Scope identifies a settings layer.
type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeDevice
	ScopeBook
)

// Store is one scope's key/value map. Values are untyped JSON-ish
// primitives; typed decoding happens at the call site.
type Store map[string]interface{}

// Resolver merges the three scopes.
type Resolver struct {
	Global Store
	Device Store
	Book   Store
}

// Get returns the effective value and the scope it came from.
func (r *Resolver) Get(key string) (interface{}, Scope, bool) {
	if r.Book != nil {
		if v, ok := r.Book[key]; ok {
			return v, ScopeBook, true
		}
	}
	if r.Device != nil {
		if v, ok := r.Device[key]; ok {
			return v, ScopeDevice, true
		}
	}
	if r.Global != nil {
		if v, ok := r.Global[key]; ok {
			return v, ScopeGlobal, true
		}
	}
	return nil, ScopeGlobal, false
}

// Set writes a value at the given scope (nil value clears the override,
// falling back to the next-wider scope).
func (r *Resolver) Set(scope Scope, key string, value interface{}) {
	var s *Store
	switch scope {
	case ScopeGlobal:
		if r.Global == nil {
			r.Global = Store{}
		}
		s = &r.Global
	case ScopeDevice:
		if r.Device == nil {
			r.Device = Store{}
		}
		s = &r.Device
	case ScopeBook:
		if r.Book == nil {
			r.Book = Store{}
		}
		s = &r.Book
	default:
		panic("settings: invalid scope")
	}
	if value == nil {
		delete(*s, key)
	} else {
		(*s)[key] = value
	}
}

// Keys returns all effective keys, sorted for deterministic serialisation.
func (r *Resolver) Keys() []string {
	seen := map[string]bool{}
	for _, s := range []Store{r.Global, r.Device, r.Book} {
		for k := range s {
			seen[k] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
