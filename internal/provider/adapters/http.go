package adapters

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	requestTimeout   = 2 * time.Minute
	maxResponseBytes = 8 << 20
)

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: requestTimeout}
}

// readBody reads a bounded response body and fails instead of truncating.
func readBody(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxResponseBytes)
	}
	return data, nil
}
