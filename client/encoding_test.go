package client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeduceCharset_NilResponse(t *testing.T) {
	c := require.New(t)

	var res *http.Response

	name, err := deduceCharset(res)
	c.Empty(name)
	c.Equal(ErrNilResponse, err)
}

func TestDeduceCharset_UTF8(t *testing.T) {
	c := require.New(t)

	header := http.Header{
		"Content-Type": []string{
			"text/html; charset=utf-8;",
		},
	}

	response := &http.Response{
		Header: header,
	}

	name, err := deduceCharset(response)
	c.Nil(err)
	c.Equal("utf-8", name)

	// returned encoding name should be the same if the header has upper case
	response.Header.Set("Content-Type", "text/html; charset=UTF-8")
	name, err = deduceCharset(response)
	c.Nil(err)
	c.Equal("utf-8", name)
}

func TestDeduceCharset_UnknownEncoding(t *testing.T) {
	c := require.New(t)

	header := http.Header{
		"Content-Type": []string{
			"text/html; charset=unknown;",
		},
	}

	response := &http.Response{
		Header: header,
	}

	name, err := deduceCharset(response)
	c.NotNil(err)
	c.Equal("", name)
}

func TestDeduceCharset_ISO8859_2(t *testing.T) {
	c := require.New(t)

	header := http.Header{
		"Content-Type": []string{
			"text/html; charset=iso-8859-2",
		},
	}

	response := &http.Response{
		Header: header,
	}

	name, err := deduceCharset(response)
	c.Nil(err)
	c.Equal("iso-8859-2", name)

	// returned encoding name should be the same if the header has upper case
	response.Header.Set("Content-Type", "text/html; charset=ISO-8859-2")
	name, err = deduceCharset(response)
	c.Nil(err)
	c.Equal("iso-8859-2", name)
}

func BenchmarkDeduceCharset_ISO8859_2(b *testing.B) {
	c := require.New(b)

	header := http.Header{
		"Content-Type": []string{
			"text/html; charset=iso-8859-2",
		},
	}

	response := &http.Response{
		Header: header,
	}

	for i := 0; i < b.N; i++ {
		name, err := deduceCharset(response)
		c.Nil(err)
		c.Equal("iso-8859-2", name)
	}
}
