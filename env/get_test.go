package env

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetStringDefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	defaultValue := "val"
	c.Equal(defaultValue, GetString("GET_STRING", defaultValue))
}

func TestGetStringCustomValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	withTestEnv("custom", func(varName string) {
		c.Equal("custom", GetString(varName, ""))
	})
}

func TestGetBoolDefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	defaultValue := true
	c.Equal(defaultValue, GetBool("GET_BOOL", defaultValue))
}

func TestGetBoolCustomValue(t *testing.T) {
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
			c.Equal(test.expected, GetBool(varName, test.defaultVal), "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestGetInt64DefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	defaultValue := int64(872)
	c.Equal(defaultValue, GetInt64("GET_INT64", defaultValue))
}

func TestGetInt64CustomValue(t *testing.T) {
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
			c.Equal(test.expected, GetInt64(varName, test.defaultVal), "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestGetInt32DefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	defaultValue := int32(872)
	c.Equal(defaultValue, GetInt32("GET_INT64", defaultValue))
}

func TestGetInt32CustomValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	ttable := []struct {
		val        interface{}
		defaultVal int32
		expected   int32
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
			c.Equal(test.expected, GetInt32(varName, test.defaultVal), "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestGetFloat64DefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	defaultValue := 873.3
	c.Equal(defaultValue, GetFloat64("GET_FLOAT64", defaultValue))
}

func TestGetFloat64CustomValue(t *testing.T) {
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
			c.Equal(test.expected, GetFloat64(varName, test.defaultVal), "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestGetStringArrayDefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	defaultValue := []string{"a", "b"}
	c.Equal(defaultValue, GetStringArray("GET_STRING_ARRAY", ",", defaultValue))
}

func TestGetStringArrayCustomValue(t *testing.T) {
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
			c.Equal(test.expected, GetStringArray(varName, test.sep, test.defaultVal), "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestGetStringSetCustomValue(t *testing.T) {
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
			c.Equal(test.expected, GetStringSet(varName, test.sep, test.defaultVal), "Failed to get %#v with %#v", varName, test.val)
		})
	}
}

func TestGetDurationDefaultValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	defaultValue := 1 * time.Hour
	c.Equal(defaultValue, GetDuration("GET_DURATION", defaultValue))
}

func TestGetDurationCustomValue(t *testing.T) {
	t.Parallel()
	c := require.New(t)

	cases := []struct {
		val        interface{}
		defaultVal time.Duration
		expected   time.Duration
	}{
		{"1h", 5 * time.Hour, 1 * time.Hour},
		{"1 hour", 5 * time.Hour, 5 * time.Hour},
	}

	for _, test := range cases {
		withTestEnv(test.val, func(varName string) {
			c.Equal(test.expected, GetDuration(varName, test.defaultVal))
		})
	}
}

func BenchmarkGetStringArray(b *testing.B) {
	defaultValue := []string{"a", "b"}

	withTestEnv("1,2,3,4", func(varName string) {
		for i := 0; i < b.N; i++ {
			GetStringArray(varName, ",", defaultValue)
		}
	})
}
