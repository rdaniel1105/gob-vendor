package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringSetHas(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	ttable := []struct {
		stringSet StringSet
		element   string
		expected  bool
	}{
		{NewStringSetWithArray([]string{"a", "b", "c"}), "a", true},
		{NewStringSetWithArray([]string{"a", "b", "c"}), "z", false},
	}

	for _, test := range ttable {
		result := test.stringSet.Has(test.element)
		c.Equal(test.expected, result, "StringSet#Has should return %#v for %#v with %#v but returned %#v", test.expected, test.stringSet, test.element, result)
	}
}

func TestStringSetRemove(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	sSet := NewStringSetWithArray([]string{"a", "b", "c"})

	c.True(sSet.Has("a"))
	sSet.Remove("a")
	c.False(sSet.Has("a"))
}

func BenchmarkStringSetHas(b *testing.B) {
	c := require.New(b)

	for i := 0; i < b.N; i++ {
		ss := NewStringSetWithArray([]string{"a", "b", "c"})
		c.False(ss.Has("z"))
	}
}
