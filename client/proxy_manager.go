package client

import (
	"context"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/rdaniel1105/gob-vendor/env"
)

type proxyCtxKeyType int

var (
	proxyCtxKey proxyCtxKeyType = 1
)

// ProxyManager manages the http proxies
type ProxyManager struct {
	proxyURLs        []*url.URL
	mutex            sync.Mutex
	nextPos          int
	useProxyRotation bool
}

const (
	// NoneProxy not use proxy
	NoneProxy = "none"

	// NoneProvider not use proxy provider
	NoneProvider = "none"
)

// NewProxyManager returns a new instance of a proxy manager
func NewProxyManager() *ProxyManager {
	useProxyRotation := env.GetBool("USE_PROXY_ROTATION", false)

	proxies := parseProxiesFromEnv()
	initialPos := 0

	if len(proxies) > 0 {
		initialPos = rand.Intn(len(proxies)) // #nosec G404
	}

	return &ProxyManager{
		proxyURLs:        proxies,
		nextPos:          initialPos,
		useProxyRotation: useProxyRotation,
	}
}

func parseProxiesFromEnv() []*url.URL {
	proxies := os.Getenv("HTTP_PROXY_URL")

	if proxies == "" {
		return []*url.URL{}
	}

	parsedProxies := strings.Split(proxies, ";")
	proxyURLs := make([]*url.URL, 0, len(parsedProxies))

	for _, proxy := range parsedProxies {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			continue
		}

		proxyURLs = append(proxyURLs, proxyURL)
	}

	return proxyURLs
}

// Get obtains a random proxy URL
func (m *ProxyManager) Get(ctx context.Context) *url.URL {
	if len(m.proxyURLs) == 0 {
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	proxy := m.proxyURLs[m.nextPos]
	m.nextPos = (m.nextPos + 1) % len(m.proxyURLs)

	return proxy
}

// ProxyForRequest returns a proxy from the request context if available.
func (m *ProxyManager) ProxyForRequest(req *http.Request) (*url.URL, error) {
	ctx := req.Context()

	proxyURL, ok := ctx.Value(proxyCtxKey).(*url.URL)
	if !ok {
		return nil, nil
	}

	return proxyURL, nil
}

// SetProxyInRequest sets the proxy in the request context and returns the proxy used.
// If the proxyURL specified is nil, a random proxy is chosen.
// If no proxies are available, returns the same request and nil proxy.
func (m *ProxyManager) SetProxyInRequest(req *http.Request, proxyURL *url.URL) (*http.Request, *url.URL) {
	if strings.HasSuffix(req.Host, "amazonaws.com") {
		return req, nil
	}

	if proxyURL == nil && m.useProxyRotation {
		return req, nil
	}

	if proxyURL != nil {
		ctx := context.WithValue(req.Context(), proxyCtxKey, proxyURL)

		return req.WithContext(ctx), proxyURL
	}

	randomProxy := m.Get(req.Context())
	if randomProxy != nil {
		ctx := context.WithValue(req.Context(), proxyCtxKey, randomProxy)

		return req.WithContext(ctx), randomProxy
	}

	return req, nil
}

// SetProxiesFromEnv sets proxy urls from env
func (m *ProxyManager) SetProxiesFromEnv() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.proxyURLs = parseProxiesFromEnv()

	if len(m.proxyURLs) > 0 {
		m.nextPos = rand.Intn(len(m.proxyURLs)) // #nosec G404
	}
}
