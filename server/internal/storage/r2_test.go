package storage

import (
	"testing"
	"time"
)

// The expected URLs below were produced by botocore (the AWS SDK's own SigV4
// implementation) using the same inputs. Cloudflare R2 rejects a signature that
// differs by a single byte, so these act as a contract test for r2.go.
const (
	testAccountID = "1234567890abcdef1234567890abcdef"
	testAccessKey = "0123456789abcdef0123456789abcdef"
	testSecretKey = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	testBucket    = "video-sharing-uploads"
	testPublicURL = "https://pub-abcdef0123456789.r2.dev"
)

func newTestClient(t *testing.T) *R2Client {
	t.Helper()
	client, err := NewR2Client(R2Config{
		AccountID:       testAccountID,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		Bucket:          testBucket,
		PublicBaseURL:   testPublicURL,
	})
	if err != nil {
		t.Fatalf("NewR2Client returned an error: %v", err)
	}
	if client == nil {
		t.Fatal("NewR2Client returned nil for a complete configuration")
	}
	return client
}

func TestPresignMatchesAWSReference(t *testing.T) {
	client := newTestClient(t)
	signedAt := time.Date(2026, 8, 9, 2, 27, 24, 0, time.UTC)

	tests := []struct {
		name        string
		method      string
		objectKey   string
		contentType string
		expires     time.Duration
		want        string
	}{
		{
			name:        "put mp4",
			method:      "PUT",
			objectKey:   "0f6c2b8e-1d4a-4a7f-9c31-3f2f8a5b7c10.mp4",
			contentType: "video/mp4",
			expires:     30 * time.Minute,
			want: "https://1234567890abcdef1234567890abcdef.r2.cloudflarestorage.com/video-sharing-uploads/0f6c2b8e-1d4a-4a7f-9c31-3f2f8a5b7c10.mp4" +
				"?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
				"&X-Amz-Credential=0123456789abcdef0123456789abcdef%2F20260809%2Fauto%2Fs3%2Faws4_request" +
				"&X-Amz-Date=20260809T022724Z&X-Amz-Expires=1800&X-Amz-SignedHeaders=content-type%3Bhost" +
				"&X-Amz-Signature=65b1b6c76ae9ada77b03cf53827ab9a03fe5e36fcf57326ae54c23472abaa4b8",
		},
		{
			name:        "put webm",
			method:      "PUT",
			objectKey:   "aa11bb22-cc33-dd44-ee55-ff6677889900.webm",
			contentType: "video/webm",
			expires:     15 * time.Minute,
			want: "https://1234567890abcdef1234567890abcdef.r2.cloudflarestorage.com/video-sharing-uploads/aa11bb22-cc33-dd44-ee55-ff6677889900.webm" +
				"?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
				"&X-Amz-Credential=0123456789abcdef0123456789abcdef%2F20260809%2Fauto%2Fs3%2Faws4_request" +
				"&X-Amz-Date=20260809T022724Z&X-Amz-Expires=900&X-Amz-SignedHeaders=content-type%3Bhost" +
				"&X-Amz-Signature=3cbae428a2418895322b72e6b529e387354e8cd4c967c8b1a319681fe7637463",
		},
		{
			name:      "delete without content type",
			method:    "DELETE",
			objectKey: "0f6c2b8e-1d4a-4a7f-9c31-3f2f8a5b7c10.mp4",
			expires:   5 * time.Minute,
			want: "https://1234567890abcdef1234567890abcdef.r2.cloudflarestorage.com/video-sharing-uploads/0f6c2b8e-1d4a-4a7f-9c31-3f2f8a5b7c10.mp4" +
				"?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
				"&X-Amz-Credential=0123456789abcdef0123456789abcdef%2F20260809%2Fauto%2Fs3%2Faws4_request" +
				"&X-Amz-Date=20260809T022724Z&X-Amz-Expires=300&X-Amz-SignedHeaders=host" +
				"&X-Amz-Signature=56104abf845d1c700a85b7b04142640dbffcd551492d15b28f0777c222c234ec",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := client.presign(tc.method, tc.objectKey, tc.contentType, tc.expires, signedAt)
			if err != nil {
				t.Fatalf("presign returned an error: %v", err)
			}
			if got != tc.want {
				t.Errorf("presigned URL does not match the AWS reference\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestPresignRejectsBadInput(t *testing.T) {
	client := newTestClient(t)
	signedAt := time.Date(2026, 8, 9, 2, 27, 24, 0, time.UTC)

	tests := []struct {
		name      string
		objectKey string
		expires   time.Duration
	}{
		{name: "empty key", objectKey: "", expires: time.Minute},
		{name: "path traversal", objectKey: "../secrets.mp4", expires: time.Minute},
		{name: "zero expiry", objectKey: "video.mp4", expires: 0},
		{name: "expiry beyond one week", objectKey: "video.mp4", expires: 8 * 24 * time.Hour},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := client.presign("PUT", tc.objectKey, "video/mp4", tc.expires, signedAt); err == nil {
				t.Error("expected an error but presign succeeded")
			}
		})
	}
}

func TestNewR2ClientConfiguration(t *testing.T) {
	t.Run("empty configuration disables R2", func(t *testing.T) {
		client, err := NewR2Client(R2Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client != nil {
			t.Error("expected nil client when no R2 settings are present")
		}
	})

	t.Run("partial configuration is an error", func(t *testing.T) {
		_, err := NewR2Client(R2Config{AccountID: testAccountID, Bucket: testBucket})
		if err == nil {
			t.Error("expected an error for a half-filled configuration")
		}
	})

	t.Run("public base URL must be https", func(t *testing.T) {
		_, err := NewR2Client(R2Config{
			AccountID:       testAccountID,
			AccessKeyID:     testAccessKey,
			SecretAccessKey: testSecretKey,
			Bucket:          testBucket,
			PublicBaseURL:   "http://pub-abcdef0123456789.r2.dev",
		})
		if err == nil {
			t.Error("expected an error for a non-https public base URL")
		}
	})
}

func TestObjectKeyFromURL(t *testing.T) {
	client := newTestClient(t)

	tests := []struct {
		name    string
		rawURL  string
		wantKey string
		wantOK  bool
	}{
		{
			name:    "object in this bucket",
			rawURL:  testPublicURL + "/0f6c2b8e-1d4a-4a7f-9c31-3f2f8a5b7c10.mp4",
			wantKey: "0f6c2b8e-1d4a-4a7f-9c31-3f2f8a5b7c10.mp4",
			wantOK:  true,
		},
		{name: "another host", rawURL: "https://evil.example.com/video.mp4"},
		{name: "legacy vercel blob url", rawURL: "https://abc.public.blob.vercel-storage.com/video.mp4"},
		{name: "plain http", rawURL: "http://pub-abcdef0123456789.r2.dev/video.mp4"},
		{name: "no object key", rawURL: testPublicURL + "/"},
		{name: "path traversal", rawURL: testPublicURL + "/../video.mp4"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, ok := client.ObjectKeyFromURL(tc.rawURL)
			if ok != tc.wantOK {
				t.Fatalf("ownership = %v, want %v", ok, tc.wantOK)
			}
			if key != tc.wantKey {
				t.Errorf("key = %q, want %q", key, tc.wantKey)
			}
		})
	}
}

func TestPublicURLRoundTrip(t *testing.T) {
	client := newTestClient(t)
	const objectKey = "aa11bb22-cc33-dd44-ee55-ff6677889900.webm"

	publicURL := client.PublicURL(objectKey)
	got, ok := client.ObjectKeyFromURL(publicURL)
	if !ok {
		t.Fatalf("URL generated by PublicURL was not recognised: %s", publicURL)
	}
	if got != objectKey {
		t.Errorf("round trip produced %q, want %q", got, objectKey)
	}
}
