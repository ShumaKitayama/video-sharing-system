package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Cloudflare R2 speaks the S3 API, so every request has to be signed with AWS
// Signature Version 4. All of that math is kept inside this one file: the rest
// of the codebase only ever sees ordinary https URLs.
const (
	r2Region        = "auto"
	r2Service       = "s3"
	sigV4Algorithm  = "AWS4-HMAC-SHA256"
	sigV4Terminator = "aws4_request"

	// The browser sends the video file straight to R2, so this server never sees
	// the bytes and cannot hash them in advance.
	sigV4UnsignedPayload = "UNSIGNED-PAYLOAD"

	// AWS SigV4 refuses presigned URLs that stay valid for longer than a week.
	maxPresignExpiry = 7 * 24 * time.Hour
)

// R2Config holds the five values Cloudflare shows when you create a bucket and
// an R2 API token. Leaving every field empty keeps video storage on local disk,
// which is what classroom LAN and Docker runs use.
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBaseURL   string // e.g. https://pub-xxxxxxxx.r2.dev
}

// R2Client issues presigned URLs and performs server-side object operations.
type R2Client struct {
	cfg          R2Config
	endpointHost string   // <accountID>.r2.cloudflarestorage.com
	publicBase   *url.URL // where finished objects are readable from
}

// NewR2Client returns nil when no R2 settings are present. A half-filled
// configuration is reported as an error instead of being silently ignored,
// because a single typo would otherwise look like "storage is not configured".
func NewR2Client(cfg R2Config) (*R2Client, error) {
	cfg = cfg.trimmed()

	settings := []struct {
		env   string
		value string
	}{
		{"R2_ACCOUNT_ID", cfg.AccountID},
		{"R2_ACCESS_KEY_ID", cfg.AccessKeyID},
		{"R2_SECRET_ACCESS_KEY", cfg.SecretAccessKey},
		{"R2_BUCKET", cfg.Bucket},
		{"R2_PUBLIC_BASE_URL", cfg.PublicBaseURL},
	}

	var missing []string
	filled := 0
	for _, s := range settings {
		if s.value == "" {
			missing = append(missing, s.env)
			continue
		}
		filled++
	}
	if filled == 0 {
		return nil, nil
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("R2 storage is only half configured; please also set %s", strings.Join(missing, ", "))
	}

	base, err := url.Parse(cfg.PublicBaseURL)
	if err != nil || base.Scheme != "https" || base.Host == "" {
		return nil, fmt.Errorf("R2_PUBLIC_BASE_URL must be an https URL, got %q", cfg.PublicBaseURL)
	}
	base.Path = strings.TrimSuffix(base.Path, "/")
	base.RawQuery = ""
	base.Fragment = ""

	return &R2Client{
		cfg:          cfg,
		endpointHost: cfg.AccountID + ".r2.cloudflarestorage.com",
		publicBase:   base,
	}, nil
}

func (c R2Config) trimmed() R2Config {
	return R2Config{
		AccountID:       strings.TrimSpace(c.AccountID),
		AccessKeyID:     strings.TrimSpace(c.AccessKeyID),
		SecretAccessKey: strings.TrimSpace(c.SecretAccessKey),
		Bucket:          strings.TrimSpace(c.Bucket),
		PublicBaseURL:   strings.TrimSpace(c.PublicBaseURL),
	}
}

// PublicURL is the address a browser can play the finished object from.
func (c *R2Client) PublicURL(objectKey string) string {
	return c.publicBase.String() + "/" + uriEncodePath(strings.TrimPrefix(objectKey, "/"))
}

// ObjectKeyFromURL turns a stored public URL back into a bucket key, and
// reports false for any URL that does not belong to this bucket.
func (c *R2Client) ObjectKeyFromURL(raw string) (string, bool) {
	if c == nil {
		return "", false
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" {
		return "", false
	}
	if !strings.EqualFold(parsed.Host, c.publicBase.Host) {
		return "", false
	}

	prefix := c.publicBase.Path + "/"
	if !strings.HasPrefix(parsed.Path, prefix) {
		return "", false
	}
	key, err := url.PathUnescape(strings.TrimPrefix(parsed.Path, prefix))
	if err != nil || key == "" || strings.Contains(key, "..") {
		return "", false
	}
	return key, true
}

// OwnsURL reports whether a URL points at an object in this bucket.
func (c *R2Client) OwnsURL(raw string) bool {
	_, ok := c.ObjectKeyFromURL(raw)
	return ok
}

// PresignPut builds a one-shot upload URL. Whoever holds the URL may store
// exactly one object, at exactly this key, with exactly this content type,
// until it expires — no Cloudflare credentials ever reach the browser.
func (c *R2Client) PresignPut(objectKey, contentType string, expires time.Duration) (string, error) {
	return c.presign(http.MethodPut, objectKey, contentType, expires, time.Now().UTC())
}

// PutObject streams an already-validated local file into the bucket. It is used
// by the multipart upload path, where the file arrives at this server first.
func (c *R2Client) PutObject(ctx context.Context, objectKey, contentType string, body io.Reader, size int64) error {
	signedURL, err := c.presign(http.MethodPut, objectKey, contentType, 30*time.Minute, time.Now().UTC())
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, signedURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	// Setting ContentLength keeps the transfer streaming instead of buffering.
	req.ContentLength = size

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("r2 upload failed: status %d: %s", res.StatusCode, strings.TrimSpace(string(detail)))
	}
	return nil
}

// DeleteObject removes one object from the bucket.
func (c *R2Client) DeleteObject(ctx context.Context, objectKey string) error {
	signedURL, err := c.presign(http.MethodDelete, objectKey, "", 5*time.Minute, time.Now().UTC())
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, signedURL, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// S3 reports a delete of a missing key as success; treat 404 the same way.
	if res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("r2 delete failed: status %d: %s", res.StatusCode, strings.TrimSpace(string(detail)))
	}
	return nil
}

// presign implements AWS Signature Version 4 "query string" authentication.
//
// The recipe is fixed by AWS and must be reproduced byte for byte:
//
//  1. build a canonical request (method, path, query, headers, payload hash)
//  2. hash it and wrap it in a "string to sign" together with the credential scope
//  3. derive a signing key from the secret through four chained HMACs
//  4. append the resulting signature to the query string
//
// now is passed in rather than read from the clock so the signature can be
// compared against a known-good reference in tests.
func (c *R2Client) presign(method, objectKey, contentType string, expires time.Duration, now time.Time) (string, error) {
	if c == nil {
		return "", errors.New("r2 storage is not configured")
	}
	objectKey = strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if objectKey == "" || strings.Contains(objectKey, "..") {
		return "", errors.New("invalid r2 object key")
	}
	if expires <= 0 || expires > maxPresignExpiry {
		return "", fmt.Errorf("r2 presign expiry must be between 1s and %s", maxPresignExpiry)
	}

	now = now.UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	credentialScope := strings.Join([]string{dateStamp, r2Region, r2Service, sigV4Terminator}, "/")

	// "host" is always signed. Signing "content-type" as well pins the object's
	// MIME type, so an upload slot handed out for an MP4 cannot be reused to
	// store something else.
	headers := map[string]string{"host": c.endpointHost}
	if contentType != "" {
		headers["content-type"] = contentType
	}
	headerNames := make([]string, 0, len(headers))
	for name := range headers {
		headerNames = append(headerNames, name)
	}
	sort.Strings(headerNames)

	var canonicalHeaders strings.Builder
	for _, name := range headerNames {
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteByte(':')
		canonicalHeaders.WriteString(headers[name])
		canonicalHeaders.WriteByte('\n')
	}
	signedHeaders := strings.Join(headerNames, ";")

	query := url.Values{}
	query.Set("X-Amz-Algorithm", sigV4Algorithm)
	query.Set("X-Amz-Credential", c.cfg.AccessKeyID+"/"+credentialScope)
	query.Set("X-Amz-Date", amzDate)
	query.Set("X-Amz-Expires", strconv.Itoa(int(expires.Seconds())))
	query.Set("X-Amz-SignedHeaders", signedHeaders)
	canonicalQuery := canonicalQueryString(query)

	canonicalURI := "/" + uriEncodePath(c.cfg.Bucket) + "/" + uriEncodePath(objectKey)

	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders.String(),
		signedHeaders,
		sigV4UnsignedPayload,
	}, "\n")

	stringToSign := strings.Join([]string{
		sigV4Algorithm,
		amzDate,
		credentialScope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	signature := hex.EncodeToString(hmacSHA256(c.signingKey(dateStamp), []byte(stringToSign)))

	return fmt.Sprintf("https://%s%s?%s&X-Amz-Signature=%s",
		c.endpointHost, canonicalURI, canonicalQuery, signature), nil
}

// signingKey derives the per-day, per-region, per-service key AWS requires.
func (c *R2Client) signingKey(dateStamp string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+c.cfg.SecretAccessKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(r2Region))
	kService := hmacSHA256(kRegion, []byte(r2Service))
	return hmacSHA256(kService, []byte(sigV4Terminator))
}

func hmacSHA256(key, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(message)
	return mac.Sum(nil)
}

func hexSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// canonicalQueryString sorts parameters by name and escapes them the way SigV4
// expects. Go's url.Values.Encode cannot be used because it writes spaces as
// "+" while AWS requires "%20".
func canonicalQueryString(values url.Values) string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		for _, value := range values[name] {
			parts = append(parts, rfc3986Escape(name)+"="+rfc3986Escape(value))
		}
	}
	return strings.Join(parts, "&")
}

// uriEncodePath escapes a path while leaving the "/" separators intact, which
// is how S3 builds the canonical URI.
func uriEncodePath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = rfc3986Escape(segment)
	}
	return strings.Join(segments, "/")
}

// rfc3986Escape percent-encodes everything outside the unreserved character set.
func rfc3986Escape(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		unreserved := (ch >= 'A' && ch <= 'Z') ||
			(ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '~'
		if unreserved {
			out.WriteByte(ch)
			continue
		}
		fmt.Fprintf(&out, "%%%02X", ch)
	}
	return out.String()
}
