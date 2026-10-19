package netutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ResolveViaCloudflareDoH resolves an IPv4 A record via Cloudflare DNS-over-HTTPS (1.1.1.1)
// when local ISP DNS blocks a target hostname.
func ResolveViaCloudflareDoH(ctx context.Context, host string) (string, error) {
	dohURL := "https://1.1.1.1/dns-query?name=" + url.QueryEscape(host) + "&type=A"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dohURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/dns-json")

	dohClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := dohClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var dnsResp struct {
		Answer []struct {
			Type int    `json:"type"`
			Data string `json:"data"`
		} `json:"Answer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dnsResp); err != nil {
		return "", err
	}
	for _, ans := range dnsResp.Answer {
		if ans.Type == 1 && net.ParseIP(strings.TrimSpace(ans.Data)) != nil {
			return strings.TrimSpace(ans.Data), nil
		}
	}
	return "", fmt.Errorf("no A record found via DoH for %s", host)
}

// NewDoHFallbackHTTPClient returns an *http.Client that automatically falls back to Cloudflare DoH (1.1.1.1)
// when standard system DNS resolution or dialing fails.
func NewDoHFallbackHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, addr)
		if err == nil {
			return conn, nil
		}
		host, port, splitErr := net.SplitHostPort(addr)
		if splitErr != nil || net.ParseIP(host) != nil {
			return nil, err
		}
		resolvedIP, dohErr := ResolveViaCloudflareDoH(ctx, host)
		if dohErr != nil {
			return nil, err
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(resolvedIP, port))
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
