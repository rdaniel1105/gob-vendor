//go:build !nomock

package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	// ErrMockFailed is an error
	ErrMockFailed = errors.New("fail on http client mock")
	// ErrMockNotFound is an error
	ErrMockNotFound = errors.New("mocked http response not found")

	// ErrMockHTTPResponseFailed is an error
	ErrMockHTTPResponseFailed = errors.New("mocked client http response failed")
)

// Mock a http client mock
type Mock struct {
}

// InitMock initializes http client mock
// and returns mocked client
// Deprecated: use httpmock instead
func InitMock() *Mock {
	mockClient := &Mock{}

	ActivateMock()

	return mockClient
}

// Close restores the previous client
func (mock *Mock) Close() {
	DeactivateMock()
}

// AddResponse adds an http response to mock
func (mock *Mock) AddResponse(method string, url string, res *http.Response) {
	data, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	AddMockedResponse(method, url, res.StatusCode, string(data))
}

// AddResponseFromBody adds an http response from body
func (mock *Mock) AddResponseFromBody(method string, url string, statusCode int, body []byte) *http.Response {
	res := NewMockResponse(statusCode, body)
	mock.AddResponse(method, url, res)

	return res
}

// Do mock request
func (mock *Mock) Do(request *http.Request) (*http.Response, error) {
	response, err := Default.execAndLog(context.Background(), request, nil)
	if err != nil && strings.Contains(err.Error(), "no responder") {
		return response, ErrMockNotFound
	}

	if response.StatusCode >= http.StatusBadRequest {
		return response, ErrMockHTTPResponseFailed
	}

	return response, err
}

// AddResponseFromFile adds an http response from file
func (mock *Mock) AddResponseFromFile(method string, url string, statusCode int, filepath string) (*http.Response, error) {
	file, err := os.Open(filepath) // #nosec
	if err != nil {
		return nil, ErrMockFailed
	}

	body, err := io.ReadAll(file)
	if err != nil {
		return nil, ErrMockFailed
	}

	return mock.AddResponseFromBody(method, url, statusCode, body), nil
}

// NewMockResponse response mock constructor
func NewMockResponse(statusCode int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBuffer(body)),
	}
}
