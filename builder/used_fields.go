package builder

import "strings"

// usedFields stores keys and their top-level parts,
// turning top-level lookups from O(N) into O(1).
type usedFields struct {
	full map[string]struct{}
	top  map[string]struct{}
}

func newUsedFields() *usedFields {
	return &usedFields{
		full: make(map[string]struct{}),
		top:  make(map[string]struct{}),
	}
}

// add inserts a new key and updates the top-level index.
func (u *usedFields) add(key string) {
	u.full[key] = struct{}{}
	top := key
	if i := strings.IndexByte(key, '.'); i != -1 {
		top = key[:i]
	}
	u.top[top] = struct{}{}
}

// hasTopLevelKey checks if a top-level key exists.
func (u *usedFields) hasTopLevelKey(top string) bool {
	_, ok := u.top[top]
	return ok
}

// hasFullKey checks if an exact key exists.
func (u *usedFields) hasFullKey(key string) bool {
	_, ok := u.full[key]
	return ok
}

// len returns the number of full keys stored in the set.
func (u *usedFields) len() int {
	return len(u.full)
}
