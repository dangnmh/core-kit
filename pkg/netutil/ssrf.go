package netutil

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var privateCIDRs = []string{
	"127.0.0.0/8",    // Loopback
	"10.0.0.0/8",     // RFC1918 Private
	"172.16.0.0/12",  // RFC1918 Private
	"192.168.0.0/16", // RFC1918 Private
	"169.254.0.0/16", // Link-local / Cloud Metadata (169.254.169.254)
	"::1/128",        // IPv6 Loopback
	"fc00::/7",       // IPv6 Unique Local
	"fe80::/10",      // IPv6 Link-Local
}

// IsPrivateOrLocalIP checks whether an IP is private, loopback, unspecified, link-local, or cloud metadata.
func IsPrivateOrLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return true
	}
	for _, cidr := range privateCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil && ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// IsPrivateOrLocalHost checks whether a hostname or IP literal resolves to a private, loopback, or local address.
func IsPrivateOrLocalHost(ctx context.Context, host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "" {
		return false
	}
	if h == "localhost" || strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".internal") {
		return true
	}
	if directIP := net.ParseIP(h); directIP != nil {
		return IsPrivateOrLocalIP(directIP)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", h)
	if err != nil || len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		if IsPrivateOrLocalIP(ip) {
			return true
		}
	}
	return false
}

// ValidateSafeProbeURL verifies that rawURL uses http/https and, when allowPrivate is false,
// does not resolve to a private, loopback, link-local, or cloud metadata address.
func ValidateSafeProbeURL(ctx context.Context, rawURL string, allowPrivate bool) error {
	trimmed := strings.TrimSpace(rawURL)
	u, err := url.Parse(trimmed)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
	if allowPrivate {
		return nil
	}

	hostname := strings.TrimSpace(u.Hostname())
	if hostname == "" {
		return fmt.Errorf("SSRF blocked: empty hostname")
	}
	if directIP := net.ParseIP(hostname); directIP != nil {
		if IsPrivateOrLocalIP(directIP) {
			return fmt.Errorf("SSRF blocked: direct IP %s is in a private/restricted range", directIP)
		}
		return nil
	}

	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".local") || strings.HasSuffix(lowerHost, ".internal") {
		return fmt.Errorf("SSRF blocked: host %s resolves to private/restricted IP", hostname)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return fmt.Errorf("SSRF blocked: host %s resolved to no IP addresses", hostname)
	}
	for _, ip := range ips {
		if IsPrivateOrLocalIP(ip) {
			return fmt.Errorf("SSRF blocked: host %s resolves to private/restricted IP %s", hostname, ip)
		}
	}
	return nil
}
