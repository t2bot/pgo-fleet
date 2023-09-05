package pgo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// CollectorEndpoint is an Endpoint compatible with pgo-fleet's `collector` process.
type CollectorEndpoint struct {
	Endpoint
	client  *http.Client
	request *http.Request
}

// NewCollectorEndpoint creates a new Endpoint compatible with pgo-fleet's `collector`
// process. The API URL is the URL to HTTP POST the profile to upon submission, and the
// authKey will be provided in the Authentication header.
//
// An error is only returned if the given URL is invalid in some way.
func NewCollectorEndpoint(apiUrl string, authKey string) (*CollectorEndpoint, error) {
	parsed, err := url.Parse(apiUrl)
	if err != nil {
		return nil, err
	}
	return &CollectorEndpoint{
		client: &http.Client{},
		request: &http.Request{
			URL:    parsed,
			Method: http.MethodPost,
			Header: map[string][]string{
				"Authorization": {fmt.Sprintf("Bearer %s", authKey)},
			},
		},
	}, nil
}

func (e *CollectorEndpoint) Submit(profile io.Reader) error {
	req := e.request.Clone(context.Background())
	req.Body = io.NopCloser(profile)
	res, err := e.client.Do(req)
	if res != nil {
		defer func(body io.ReadCloser) {
			_ = body.Close()
		}(res.Body)
	}
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code %d (%s)", res.StatusCode, res.Status)
	}
	return nil
}
