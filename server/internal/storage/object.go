package storage

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// headObject asks a public object URL for its size and content type without
// downloading a single byte of the file. It is how the API confirms that a
// browser upload really landed in the bucket before writing a database row.
func headObject(ctx context.Context, objectURL string) (size int64, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, objectURL, nil)
	if err != nil {
		return 0, "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return 0, "", fmt.Errorf("object head failed: status %d", res.StatusCode)
	}
	return res.ContentLength, strings.TrimSpace(res.Header.Get("Content-Type")), nil
}
