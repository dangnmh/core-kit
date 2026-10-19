package r2

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dangnmh/core-kit/pkg/strutil"
)

// Config holds Cloudflare R2 S3-compatible storage credentials and endpoint settings.
type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicBaseURL   string
	Endpoint        string
}

// Client uploads and mirrors binary media streams to Cloudflare R2 using AWS Signature Version 4 (AWS4-HMAC-SHA256).
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient constructs a new Cloudflare R2 Client.
func NewClient(cfg Config, httpClient *http.Client) *Client {
	cfg.AccountID = strings.TrimSpace(cfg.AccountID)
	cfg.AccessKeyID = strings.TrimSpace(cfg.AccessKeyID)
	cfg.SecretAccessKey = strings.TrimSpace(cfg.SecretAccessKey)
	cfg.BucketName = strings.TrimSpace(cfg.BucketName)
	if cfg.BucketName == "" {
		cfg.BucketName = "utilgarden-media"
	}
	cfg.PublicBaseURL = strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/")
	cfg.Endpoint = strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		cfg:        cfg,
		httpClient: httpClient,
	}
}

// IsConfigured returns true when R2 credentials and bucket/account/endpoint are configured.
func (c *Client) IsConfigured() bool {
	if c == nil || c.cfg.AccessKeyID == "" || c.cfg.SecretAccessKey == "" || c.cfg.BucketName == "" {
		return false
	}
	return c.cfg.AccountID != "" || c.cfg.Endpoint != ""
}

func (c *Client) resolveEndpoint() string {
	if c.cfg.Endpoint != "" {
		return c.cfg.Endpoint
	}
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.cfg.AccountID)
}

// IsPermanentCDNURL returns true if rawURL already points to the configured R2 public CDN or an r2.dev domain.
func (c *Client) IsPermanentCDNURL(rawURL string) bool {
	lower := strings.ToLower(strings.TrimSpace(rawURL))
	if c != nil && c.cfg.PublicBaseURL != "" && strings.HasPrefix(lower, strings.ToLower(c.cfg.PublicBaseURL)) {
		return true
	}
	return strings.Contains(lower, ".r2.dev/") ||
		strings.Contains(lower, ".r2.cloudflarestorage.com/")
}

// InferMediaMimeAndExt infers the canonical Content-Type and file extension for an image or video asset.
func InferMediaMimeAndExt(sourceURL, contentTypeHeader, mediaType string) (string, string) {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentTypeHeader, ";")[0]))
	lowerURL := strings.ToLower(sourceURL)

	if mediaType == "video" || strings.HasPrefix(ct, "video/") {
		switch {
		case strings.Contains(ct, "webm") || strings.Contains(lowerURL, ".webm"):
			return "video/webm", ".webm"
		case strings.Contains(ct, "quicktime") || strings.Contains(lowerURL, ".mov"):
			return "video/quicktime", ".mov"
		default:
			return "video/mp4", ".mp4"
		}
	}

	switch {
	case strings.Contains(ct, "webp") || strings.Contains(lowerURL, ".webp"):
		return "image/webp", ".webp"
	case strings.Contains(ct, "png") || strings.Contains(lowerURL, ".png"):
		return "image/png", ".png"
	case strings.Contains(ct, "gif") || strings.Contains(lowerURL, ".gif"):
		return "image/gif", ".gif"
	case strings.Contains(ct, "svg") || strings.Contains(lowerURL, ".svg"):
		return "image/svg+xml", ".svg"
	default:
		return "image/jpeg", ".jpg"
	}
}

// UploadBytes uploads raw media bytes directly to Cloudflare R2 using S3 Signature Version 4 (AWS4-HMAC-SHA256).
func (c *Client) UploadBytes(ctx context.Context, payload []byte, contentType, ext, mediaType string) (string, error) {
	if !c.IsConfigured() {
		return "", nil
	}
	if len(payload) == 0 {
		return "", fmt.Errorf("empty media payload")
	}

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := strutil.SHA256Hex(payload)

	folder := "image"
	if mediaType == "video" {
		folder = "video"
	}
	objectKey := fmt.Sprintf("media/%s/%s/%s/%s%s",
		folder,
		now.Format("2006"),
		now.Format("01"),
		payloadHash[:24],
		ext,
	)

	endpoint := c.resolveEndpoint()
	targetURL := fmt.Sprintf("%s/%s/%s", endpoint, c.cfg.BucketName, objectKey)
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("invalid R2 target URL: %w", err)
	}

	canonicalURI := parsedURL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	host := parsedURL.Host

	canonicalHeaders := fmt.Sprintf(
		"content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		contentType,
		host,
		payloadHash,
		amzDate,
	)
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{
		http.MethodPut,
		canonicalURI,
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	region := "auto"
	service := "s3"
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		strutil.SHA256HexString(canonicalRequest),
	}, "\n")

	kDate := strutil.HMACSHA256([]byte("AWS4"+c.cfg.SecretAccessKey), dateStamp)
	kRegion := strutil.HMACSHA256(kDate, region)
	kService := strutil.HMACSHA256(kRegion, service)
	kSigning := strutil.HMACSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(strutil.HMACSHA256(kSigning, stringToSign))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.cfg.AccessKeyID,
		credentialScope,
		signedHeaders,
		signature,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, targetURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(payload))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Cache-Control", "public, max-age=31536000, immutable")
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("R2 PUT request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("R2 PUT returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	return fmt.Sprintf("%s/%s", c.cfg.PublicBaseURL, objectKey), nil
}

// UploadFromURL downloads a remote image or video stream and uploads it directly to Cloudflare R2.
func (c *Client) UploadFromURL(ctx context.Context, mediaURL, mediaType string) (string, error) {
	if !c.IsConfigured() {
		return "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download source media %s: %w", mediaURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("source media %s returned HTTP %d", mediaURL, resp.StatusCode)
	}

	maxBytes := int64(25 * 1024 * 1024)
	if mediaType == "video" {
		maxBytes = int64(200 * 1024 * 1024)
	}

	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return "", fmt.Errorf("failed to read source media stream: %w", err)
	}

	contentType, ext := InferMediaMimeAndExt(mediaURL, resp.Header.Get("Content-Type"), mediaType)
	return c.UploadBytes(ctx, payload, contentType, ext, mediaType)
}
