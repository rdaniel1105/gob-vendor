package env

// StringSet implements an string set
type StringSet map[string]struct{}

// NewStringSet creates an empty string set
func NewStringSet() StringSet {
	return StringSet{}
}

// NewStringSetWithArray creates an string set based in the provided array
func NewStringSetWithArray(arr []string) StringSet {
	ss := NewStringSet()

	for _, e := range arr {
		ss.Add(e)
	}

	return ss
}

// Add append an string to the string set
func (ss StringSet) Add(e string) StringSet {
	ss[e] = struct{}{}

	return ss
}

// Remove string to the string set
func (ss StringSet) Remove(e string) StringSet {
	delete(ss, e)

	return ss
}

// Has returns true when the string is in the string set
func (ss StringSet) Has(e string) bool {
	_, ok := ss[e]
	return ok
}
