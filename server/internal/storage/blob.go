package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	blobAPIBase    = "https://blob.vercel-storage.com"
	blobAPIVersion = "7"
)

// clientTokenPayload mirrors the JSON payload signed by @vercel/blob when it
// issues a browser upload token. Field names must match exactly so Vercel Blob
// can validate the token server-side.
type clientTokenPayload struct {
	Pathname            string   `json:"pathname"`
	AllowedContentTypes []string `json:"allowedContentTypes,omitempty"`
	MaximumSizeInBytes  int64    `json:"maximumSizeInBytes,omitempty"`
	AddRandomSuffix     bool     `json:"addRandomSuffix"`
	ValidUntil          int64    `json:"validUntil"`
}

// GenerateBlobClientToken reproduces @vercel/blob's client-token algorithm in Go.
//
// The browser uploads the video file straight to Vercel Blob (bypassing our API
// and Vercel's 4.5MB request-body limit). To authorize that direct upload, the
// browser needs a short-lived client token derived from the store's read/write
// token. The format is:
//
//	vercel_blob_client_<storeId>_<base64( hmacHex + "." + base64(payload) )>
//
// where hmacHex = HMAC-SHA256(readWriteToken, base64(payload)).
func GenerateBlobClientToken(rwToken, pathname string, allowedContentTypes []string, maxBytes int64, validFor time.Duration) (string, error) {
	rwToken = strings.TrimSpace(rwToken)
	if rwToken == "" {
		return "", fmt.Errorf("blob read/write token is not configured")
	}
	// vercel_blob_rw_<storeId>_<secret> -> storeId is the 4th underscore field.
	parts := strings.Split(rwToken, "_")
	if len(parts) < 4 || parts[3] == "" {
		return "", fmt.Errorf("invalid blob read/write token format")
	}
	storeID := parts[3]

	pathname = strings.Trim(strings.TrimPrefix(pathname, "/"), "/")
	if pathname == "" {
		return "", fmt.Errorf("blob pathname is required")
	}

	payload := clientTokenPayload{
		Pathname:            pathname,
		AllowedContentTypes: allowedContentTypes,
		MaximumSizeInBytes:  maxBytes,
		AddRandomSuffix:     true,
		ValidUntil:          time.Now().Add(validFor).UnixMilli(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.StdEncoding.EncodeToString(raw)

	mac := hmac.New(sha256.New, []byte(rwToken))
	_, _ = mac.Write([]byte(payloadB64))
	securedKey := hex.EncodeToString(mac.Sum(nil))

	combined := base64.StdEncoding.EncodeToString([]byte(securedKey + "." + payloadB64))
	return fmt.Sprintf("vercel_blob_client_%s_%s", storeID, combined), nil
}

// statBlob issues a HEAD request to a public blob URL to read authoritative
// size and content type after a direct browser upload.
func statBlob(ctx context.Context, blobURL string) (size int64, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, blobURL, nil)
	if err != nil {
		return 0, "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return 0, "", fmt.Errorf("blob head failed: status %d", res.StatusCode)
	}
	return res.ContentLength, strings.TrimSpace(res.Header.Get("Content-Type")), nil
}

type blobPutResponse struct {
	URL string `json:"url"`
}

type blobDeleteRequest struct {
	URLs []string `json:"urls"`
}

// uploadToBlob streams src into Vercel Blob and returns the public blob URL.
func uploadToBlob(ctx context.Context, token, pathname, contentType string, src io.Reader) (string, error) {
	pathname = strings.Trim(strings.TrimPrefix(pathname, "/"), "/")
	if pathname == "" {
		return "", fmt.Errorf("blob pathname is required")
	}

	endpoint, err := url.Parse(blobAPIBase)
	if err != nil {
		return "", err
	}
	endpoint.Path = pathname

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), src)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-api-version", blobAPIVersion)
	req.Header.Set("x-add-random-suffix", "false")
	if contentType != "" {
		req.Header.Set("x-content-type", contentType)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return "", fmt.Errorf("blob upload failed: status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var out blobPutResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode blob upload response: %w", err)
	}
	if out.URL == "" {
		return "", fmt.Errorf("blob upload returned empty url")
	}
	return out.URL, nil
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
