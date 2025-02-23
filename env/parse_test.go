package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseJSON(t *testing.T) {
	t.Parallel()

	c := require.New(t)

	withTestEnv(`{"hello": "world"}`, func(varName string) {
		v := map[string]string{}
		c.NoError(ParseJSON(varName, &v))

		c.Equal("world", v["hello"])
	})
}

func BenchmarkParseJSON(b *testing.B) {
	withTestEnv(`{"hello": "world"}`, func(varName string) {
		for i := 0; i < b.N; i++ {
			v := map[string]string{}
			_ = ParseJSON(varName, &v)
		}
	})
}
