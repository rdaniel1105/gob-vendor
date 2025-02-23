package client

import (
	"errors"
	"fmt"
	"mime"
	"net/http"

	"golang.org/x/net/html/charset"
)

var (
	// ErrNilResponse means than a invalid nil response was found
	ErrNilResponse = errors.New("invalid nil response")
	// ErrInvalidContentType means than the response content has invalid format
	ErrInvalidContentType = errors.New("invalid content type")
	defaultCharset        = "utf-8"
)

func deduceCharset(res *http.Response) (string, error) {
	if res == nil {
		return "", ErrNilResponse
	}

	contentType := res.Header.Get("Content-Type")
	if contentType == "" {
		return defaultCharset, nil
	}

	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", ErrInvalidContentType
	}

	chSet, ok := params["charset"]
	if !ok {
		return defaultCharset, nil
	}

	encoding, name := charset.Lookup(chSet)
	if encoding == nil {
		return "", fmt.Errorf("unknown encoding: %s", name)
	}

	return name, nil
}
