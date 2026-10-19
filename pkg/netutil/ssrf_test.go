package netutil_test

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/dangnmh/core-kit/pkg/netutil"
)

func TestIsPrivateOrLocalIP(t *testing.T) {
	t.Parallel()

	privateIPs := []string{
		"127.0.0.1",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"169.254.169.254",
		"::1",
		"fc00::1",
		"fe80::1",
		"0.0.0.0",
	}
	for _, raw := range privateIPs {
		ip := net.ParseIP(raw)
		if !netutil.IsPrivateOrLocalIP(ip) {
			t.Fatalf("expected %s to be private/local IP", raw)
		}
	}

	publicIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
	}
	for _, raw := range publicIPs {
		ip := net.ParseIP(raw)
		if netutil.IsPrivateOrLocalIP(ip) {
			t.Fatalf("expected %s to be public IP", raw)
		}
	}

	if netutil.IsPrivateOrLocalIP(nil) {
		t.Fatal("expected nil IP to return false")
	}
}

func TestIsPrivateOrLocalHost(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for _, h := range []string{"localhost", "127.0.0.1", "169.254.169.254", "service.internal", "mybox.local"} {
		if !netutil.IsPrivateOrLocalHost(ctx, h) {
			t.Fatalf("expected host %s to be private/local", h)
		}
	}
	if netutil.IsPrivateOrLocalHost(ctx, "8.8.8.8") {
		t.Fatal("expected 8.8.8.8 to be public")
	}
}

func TestValidateSafeProbeURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	blocked := []string{
		"http://127.0.0.1:8080/probe",
		"http://localhost:3000",
		"http://169.254.169.254/latest/meta-data",
		"http://10.0.0.1/admin",
		"http://192.168.1.1/internal",
		"http://172.16.0.1/secret",
		"http://[::1]:8080",
	}
	for _, u := range blocked {
		err := netutil.ValidateSafeProbeURL(ctx, u, false)
		if err == nil || !strings.Contains(err.Error(), "SSRF blocked") {
			t.Fatalf("expected SSRF blocked error for %s, got %v", u, err)
		}
	}

	if err := netutil.ValidateSafeProbeURL(ctx, "http://127.0.0.1:8080/probe", true); err != nil {
		t.Fatalf("expected allowPrivate=true to permit loopback, got %v", err)
	}

	if err := netutil.ValidateSafeProbeURL(ctx, "ftp://127.0.0.1:21", true); err == nil || !strings.Contains(err.Error(), "unsupported URL scheme") {
		t.Fatalf("expected unsupported URL scheme error, got %v", err)
	}
}
