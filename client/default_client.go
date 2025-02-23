package client

import (
	"context"
	"crypto/tls" // #nosec
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rdaniel1105/gob-vendor/env"
)

var (
	// Default is the default HTTP client
	Default *Client

	// DefaultTransport for the client
	DefaultTransport *http.Transport

	// DefaultRoundTripper for the client
	DefaultRoundTripper http.RoundTripper

	httpClientTimeout = 15 * time.Second

	httpClientRetries = int(env.GetInt64("HTTP_CLIENT_RETRIES", 3))

	// DefaultProxyManager is the default manager for proxies
	DefaultProxyManager = NewProxyManager()

	httpSkipVerify = env.GetBool("HTTP_CLIENT_SKIP_VERIFY", false)

	forceHTTPSRedirect = env.GetBool("FORCE_HTTPS_REDIRECT", false)

	forceAttemptHTTP2 = env.GetBool("HTTP_CLIENT_FORCE_ATTEMPT_HTTP2", false)
)

func init() {
	configTimeout, _ := strconv.Atoi(env.GetString("HTTP_CLIENT_TIMEOUT", "15"))
	if configTimeout > 0 {
		httpClientTimeout = time.Duration(configTimeout) * time.Second
	}

	dialer := &net.Dialer{
		Timeout:   httpClientTimeout,
		KeepAlive: 60 * time.Second,
	}
	DefaultTransport = &http.Transport{
		Proxy:               DefaultProxyManager.ProxyForRequest,
		DialContext:         dialer.DialContext,
		MaxIdleConns:        128,
		MaxIdleConnsPerHost: 128,
		IdleConnTimeout:     60 * time.Second,
		ForceAttemptHTTP2:   forceAttemptHTTP2,
	}

	if httpSkipVerify {
		DefaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: httpSkipVerify} // #nosec
	}

	DefaultRoundTripper = DefaultTransport

	var err error

	Default, err = New()
	if err != nil {
		panic(err)
	}
}

// TimeoutDuration return the duration limit of a http request
func TimeoutDuration() time.Duration {
	return httpClientTimeout
}

// ActivateMock activates the mock
func ActivateMock() {
	Default.ActivateMock()
}

// DeactivateMock deactivates the mock
func DeactivateMock() {
	Default.DeactivateMock()
}

// AddMockedResponseFromFile adds a mocked response given a file path relative to the test file
func AddMockedResponseFromFile(method string, url string, statusCode int, filePath string) {
	Default.AddMockedResponseFromFile(method, url, statusCode, filePath)
}

// AddMultipleMockedResponses add a mocked response given one to one from each file
func AddMultipleMockedResponses(method string, url string, statusCode int, filesPath []string) {
	Default.AddMultipleMockedResponses(method, url, statusCode, filesPath)
}

// AddMockedResponse adds a mocked response given its content
func AddMockedResponse(method string, url string, statusCode int, content string) {
	Default.AddMockedResponse(method, url, statusCode, content)
}

// Get does an http GET request
func Get(ctx context.Context, url string, headers http.Header, params url.Values) (*http.Response, error) {
	return Default.Get(ctx, url, headers, params)
}

// Post does an http POST request
func Post(ctx context.Context, url string, headers http.Header, params url.Values) (*http.Response, error) {
	return Default.Post(ctx, url, headers, params)
}

// SendRequest sends a http request
func SendRequest(ctx context.Context, method string, url string, headers http.Header, body string) (*http.Response, error) {
	return Default.SendRequest(ctx, method, url, headers, body)
}
