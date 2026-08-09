package storage

// Legacy Vercel Blob support.
//
// New uploads go to Cloudflare R2 (see r2.go). Videos uploaded before that
// migration still have a Vercel Blob URL in their `storage_key`, so they keep
// playing through the normal redirect. The only operation still needed here is
// deletion, which requires the original store's read/write token.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	blobAPIBase    = "https://blob.vercel-storage.com"
	blobAPIVersion = "7"
	blobHostSuffix = ".blob.vercel-storage.com"
)

// isLegacyBlobHost reports whether a hostname belongs to Vercel Blob.
func isLegacyBlobHost(hostname string) bool {
	return strings.HasSuffix(strings.ToLower(hostname), blobHostSuffix)
}

type blobDeleteRequest struct {
	URLs []string `json:"urls"`
}

func deleteFromBlob(ctx context.Context, token, blobURL string) error {
	payload, err := json.Marshal(blobDeleteRequest{URLs: []string{blobURL}})
	if err != nil {
		return err
	}

	endpoint, err := url.Parse(blobAPIBase)
	if err != nil {
		return err
	}
	endpoint.Path = "/delete"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-api-version", blobAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("blob delete failed: status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
