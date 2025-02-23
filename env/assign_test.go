package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssignStringDefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	val := "default"
	AssignString(&val, "ASSIGN_STRING")
	c.Equal("default", val)
}

func TestAssignStringCustomValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	withTestEnv("custom", func(varName string) {
		val := "default"
		AssignString(&val, varName)
		c.Equal("custom", val)
	})
}

func TestAssignBoolDefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	val := true
	AssignBool(&val, "ASSIGN_BOOL")
	c.Equal(true, val)
}

func TestAssignBoolCustomValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	ttable := []struct {
		val        interface{}
		defaultVal bool
		expected   bool
	}{
		{false, true, false},
		{"no", true, false},
		{0, true, false},
		{"true", false, true},
		{"yes", false, true},
		{1, false, true},
		{"x", true, true},
	}

	for _, test := range ttable {
		withTestEnv(test.val, func(varName string) {
			val := test.defaultVal
			AssignBool(&val, varName)

			c.Equal(test.expected, val, "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestAssignInt64DefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	val := int64(874)
	AssignInt64(&val, "ASSIGN_INT64")
	c.Equal(int64(874), val)
}

func TestAssignInt64CustomValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	ttable := []struct {
		val        interface{}
		defaultVal int64
		expected   int64
	}{
		{false, -1, -1},
		{"34", -1, 34},
		{-34, -1, -34},
		{"00001", -1, 1},
		{"x", -1, -1},
		{0x7F, -1, 127},
	}

	for _, test := range ttable {
		withTestEnv(test.val, func(varName string) {
			val := test.defaultVal
			AssignInt64(&val, varName)

			c.Equal(test.expected, val, "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestAssignFloat64DefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	val := 873.3
	AssignFloat64(&val, "ASSIGN_FLOAT64")
	c.Equal(873.3, val)
}

func TestAssignFloat64CustomValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	ttable := []struct {
		val        interface{}
		defaultVal float64
		expected   float64
	}{
		{false, -1, -1},
		{"34.3", -1, 34.3},
		{-34.934, -1, -34.934},
		{"00001", -1, 1},
		{"x", -1, -1},
		{0x7F, -1, 127},
	}

	for _, test := range ttable {
		withTestEnv(test.val, func(varName string) {
			val := test.defaultVal
			AssignFloat64(&val, varName)

			c.Equal(test.expected, val, "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestAssignStringArrayDefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	val := []string{"a", "b"}
	AssignStringArray(&val, "ASSIGN_STRING_ARRAY", ",")
	c.Equal([]string{"a", "b"}, val)
}

func TestAssignStringArrayCustomValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	ttable := []struct {
		val        interface{}
		sep        string
		defaultVal []string
		expected   []string
	}{
		{false, ",", []string{"default"}, []string{"false"}},
		{"1,2,3,4", ",", []string{"default"}, []string{"1", "2", "3", "4"}},
		{"1;2;3;4", ";", []string{"default"}, []string{"1", "2", "3", "4"}},
		{"1;2;3;4", ",", []string{"default"}, []string{"1;2;3;4"}},
		{"x", ",", []string{"default"}, []string{"x"}},
	}

	for _, test := range ttable {
		withTestEnv(test.val, func(varName string) {
			val := test.defaultVal
			AssignStringArray(&val, varName, test.sep)

			c.Equal(test.expected, val, "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestAssignStringSetCustomValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	ttable := []struct {
		val        interface{}
		sep        string
		defaultVal StringSet
		expected   StringSet
	}{
		{false, ",", NewStringSetWithArray([]string{"default"}), NewStringSetWithArray([]string{"false"})},
		{"1,2,3,4", ",", NewStringSetWithArray([]string{"default"}), NewStringSetWithArray([]string{"1", "2", "3", "4"})},
		{"1;2;3;4", ";", NewStringSetWithArray([]string{"default"}), NewStringSetWithArray([]string{"1", "2", "3", "4"})},
		{"1;2;3;4", ",", NewStringSetWithArray([]string{"default"}), NewStringSetWithArray([]string{"1;2;3;4"})},
		{"x", ",", NewStringSetWithArray([]string{"default"}), NewStringSetWithArray([]string{"x"})},
	}

	for _, test := range ttable {
		withTestEnv(test.val, func(varName string) {
			val := test.defaultVal
			AssignStringSet(&val, varName, test.sep)

			c.Equal(test.expected, val, "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func BenchmarkAssignStringArray(b *testing.B) {
	withTestEnv("1,2,3,4", func(varName string) {
		for i := 0; i < b.N; i++ {
			var val []string
			AssignStringArray(&val, varName, ",")
		}
	})
}
