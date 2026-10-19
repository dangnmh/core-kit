package r2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInferMediaMimeAndExt(t *testing.T) {
	tests := []struct {
		url      string
		ct       string
		mtype    string
		wantMime string
		wantExt  string
	}{
		{"https://example.com/a.webp", "image/webp", "image", "image/webp", ".webp"},
		{"https://example.com/a.png", "image/png; charset=utf-8", "image", "image/png", ".png"},
		{"https://example.com/a.gif", "", "image", "image/gif", ".gif"},
		{"https://example.com/a.svg", "", "image", "image/svg+xml", ".svg"},
		{"https://example.com/a.jpg", "application/octet-stream", "image", "image/jpeg", ".jpg"},
		{"https://example.com/v.webm", "video/webm", "video", "video/webm", ".webm"},
		{"https://example.com/v.mov", "video/quicktime", "video", "video/quicktime", ".mov"},
		{"https://example.com/v.mp4", "", "video", "video/mp4", ".mp4"},
	}

	for _, tc := range tests {
		mime, ext := InferMediaMimeAndExt(tc.url, tc.ct, tc.mtype)
		if mime != tc.wantMime || ext != tc.wantExt {
			t.Fatalf("InferMediaMimeAndExt(%q, %q, %q) = (%q, %q), want (%q, %q)",
				tc.url, tc.ct, tc.mtype, mime, ext, tc.wantMime, tc.wantExt)
		}
	}
}

func TestClient_UploadBytesAndFromURL(t *testing.T) {
	var unconfigured *Client
	if unconfigured.IsConfigured() {
		t.Fatal("expected nil client to be unconfigured")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/source.png" {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-png-bytes"))
			return
		}
		if r.Method == http.MethodPut {
			if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClient(Config{
		AccessKeyID:     "AKIA_TEST",
		SecretAccessKey: "SECRET_TEST",
		BucketName:      "test-bucket",
		PublicBaseURL:   "https://cdn.example.com",
		Endpoint:        srv.URL,
	}, srv.Client())

	if !client.IsConfigured() {
		t.Fatal("expected client to be configured")
	}
	if !client.IsPermanentCDNURL("https://cdn.example.com/media/image/2026/10/abc.png") {
		t.Fatal("expected CDN URL to be recognized as permanent")
	}

	cdnURL, err := client.UploadFromURL(context.Background(), srv.URL+"/source.png", "image")
	if err != nil {
		t.Fatalf("UploadFromURL failed: %v", err)
	}
	if !strings.HasPrefix(cdnURL, "https://cdn.example.com/media/image/") || !strings.HasSuffix(cdnURL, ".png") {
		t.Fatalf("unexpected CDN URL: %s", cdnURL)
	}
}
