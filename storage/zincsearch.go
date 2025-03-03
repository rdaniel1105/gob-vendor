package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rdaniel1105/gob-vendor/client"
)

var (
	errNewReq      = errors.New("error creating request")
	errDoReq       = errors.New("error doing request")
	errReadingBody = errors.New("error reading body")

	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
			DisableCompression:  false,
			ForceAttemptHTTP2:   true,
		},
	}
)

type ZincSearch struct {
	zincSearchURL string
	admin         string
	password      string
}

func NewZincSearchClient(zincSearchURL, admin, password string) *ZincSearch {
	return &ZincSearch{
		zincSearchURL: zincSearchURL,
		admin:         admin,
		password:      password,
	}
}

func (z *ZincSearch) Save(ctx context.Context, body string) error {
	req, err := http.NewRequest(http.MethodPost, z.zincSearchURL, bytes.NewReader([]byte(body)))
	if err != nil {
		return fmt.Errorf("%w: %w", errNewReq, err)
	}

	req.SetBasicAuth(z.admin, z.password)
	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header.Set("Connection", "keep-alive")

	newClient, err := client.New()
	if err != nil {
		return fmt.Errorf("creating new client: %w", err)
	}

	resp, err := newClient.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bulk request failed with status %d", resp.StatusCode)
	}

	return nil
}
