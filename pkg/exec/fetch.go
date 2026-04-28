package exec

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Fetch downloads a payload over TLS and returns the response body.
// Caller must close the returned ReadCloser.
func Fetch(url string) (io.ReadCloser, error) {
	timeout := 30 * time.Second
	if t := os.Getenv("TIMEOUT"); t != "" {
		if secs, err := strconv.Atoi(t); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}

	tlsCfg := &tls.Config{
		InsecureSkipVerify: os.Getenv("INSECURE") == "1", //nolint:gosec
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}

	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch: server returned %d", resp.StatusCode)
	}
	return resp.Body, nil
}
