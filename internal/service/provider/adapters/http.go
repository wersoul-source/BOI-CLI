package adapters

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultRequestTimeout = 2 * time.Minute
	maxResponseBytes      = 8 << 20
	// EnvRequestTimeout lets a slow local model take longer than the default
	// for one request; a value outside 10s..2h or not a duration is ignored.
	EnvRequestTimeout = "BOI_REQUEST_TIMEOUT"
)

func requestTimeout() time.Duration {
	value := strings.TrimSpace(os.Getenv(EnvRequestTimeout))
	if value == "" {
		return defaultRequestTimeout
	}
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout < 10*time.Second || timeout > 2*time.Hour {
		return defaultRequestTimeout
	}
	return timeout
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: requestTimeout()}
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
