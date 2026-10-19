package netutil

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	bareDomainPrefixRegex = regexp.MustCompile(`^[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+`)

	defaultShortlinkDomains = map[string]bool{
		"bit.ly":      true,
		"tinyurl.com": true,
		"t.co":        true,
		"lnkd.in":     true,
		"goo.gl":      true,
		"is.gd":       true,
		"buff.ly":     true,
		"ow.ly":       true,
		"rebrand.ly":  true,
		"cutt.ly":     true,
		"shorturl.at": true,
		"rb.gy":       true,
		"s.id":        true,
	}

	defaultSpamAffiliateDomains = []string{
		"shopee.ee",
		"s.shopee.vn",
		"shopee.vn",
		"lazada.vn",
		"s.lazada.vn",
		"tiki.vn",
		"tokopedia.com",
		"aliexpress.com",
		"temu.com",
	}

	freeHostingSuffixes = []string{
		".vercel.app",
		".lovable.app",
		".netlify.app",
		".pages.dev",
		".railway.app",
		".onrender.com",
		".fly.dev",
		".replit.app",
		".github.io",
		".web.app",
		".firebaseapp.com",
		".glitch.me",
		".ngrok-free.app",
	}
)

// NormalizeHTTPURL prefixes https:// if protocol is missing on a bare domain string.
func NormalizeHTTPURL(urlStr string) string {
	trimmed := strings.TrimSpace(urlStr)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return trimmed
	}
	if bareDomainPrefixRegex.MatchString(trimmed) {
		return "https://" + trimmed
	}
	return trimmed
}

// IsValidHTTPURL verifies if the given string is a valid HTTP/HTTPS URL.
func IsValidHTTPURL(urlStr string) bool {
	normalized := NormalizeHTTPURL(urlStr)
	u, err := url.ParseRequestURI(normalized)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// UnwrapRedirect parses Facebook redirects (l.facebook.com / lm.facebook.com),
// strips tracking parameters (fbclid, utm_*, ref, h), and returns the clean URL.
func UnwrapRedirect(rawURL string) string {
	u := strings.TrimSpace(rawURL)
	u = strings.TrimRight(u, ".,;!)]}")
	if u == "" {
		return ""
	}

	if strings.Contains(u, "l.facebook.com/l.php") || strings.Contains(u, "lm.facebook.com/l.php") {
		if parsed, err := url.Parse(u); err == nil {
			if target := parsed.Query().Get("u"); target != "" {
				if decoded, err := url.QueryUnescape(target); err == nil {
					u = decoded
				} else {
					u = target
				}
			}
		}
	}

	u = strings.TrimRight(strings.TrimSpace(u), ".,;!)]}/")
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return ""
	}

	parsed, err := url.Parse(u)
	if err != nil || parsed.Host == "" || !strings.Contains(parsed.Host, ".") {
		return ""
	}

	q := parsed.Query()
	hasTracking := false
	for k := range q {
		lowerK := strings.ToLower(k)
		if lowerK == "fbclid" || lowerK == "h" || lowerK == "ref" || lowerK == "ref_source" || strings.HasPrefix(lowerK, "utm_") {
			q.Del(k)
			hasTracking = true
		}
	}

	if hasTracking {
		if len(q) == 0 {
			parsed.RawQuery = ""
		} else {
			parsed.RawQuery = q.Encode()
		}
	}

	res := parsed.String()
	res = strings.TrimRight(res, "/")
	if parsed.Path == "" || parsed.Path == "/" {
		if parsed.RawQuery == "" {
			res = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
		}
	}

	return res
}

// IsShortlinkDomain checks if the host is a known shortlink domain.
func IsShortlinkDomain(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(strings.TrimPrefix(parsed.Hostname(), "www."))
	return defaultShortlinkDomains[host]
}

// ResolveShortlink issues a HEAD (with GET fallback) request following up to 3 redirects
// to uncover the canonical target URL, with a 3-second timeout and SSRF guard.
func ResolveShortlink(ctx context.Context, rawURL string, allowPrivate bool) string {
	cleanURL := UnwrapRedirect(rawURL)
	if cleanURL == "" {
		return rawURL
	}

	parsed, err := url.Parse(cleanURL)
	if err != nil || parsed.Host == "" {
		return cleanURL
	}

	if !IsShortlinkDomain(cleanURL) {
		return cleanURL
	}

	isSafe := func(host string) bool {
		return ValidateSafeProbeURL(ctx, "https://"+strings.TrimSpace(host), allowPrivate) == nil
	}

	if !isSafe(parsed.Hostname()) {
		return cleanURL
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			if !isSafe(req.URL.Hostname()) {
				return fmt.Errorf("blocked redirect to unsafe host: %s", req.URL.Hostname())
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, cleanURL, nil)
	if err != nil {
		return cleanURL
	}
	req.Header.Set("User-Agent", "CoreKit-LinkResolver/1.0")

	resp, err := client.Do(req)
	if err != nil {
		reqGet, errGet := http.NewRequestWithContext(ctx, http.MethodGet, cleanURL, nil)
		if errGet == nil {
			reqGet.Header.Set("User-Agent", "CoreKit-LinkResolver/1.0")
			if respGet, errGetDo := client.Do(reqGet); errGetDo == nil {
				defer func() { _ = respGet.Body.Close() }()
				if respGet.Request != nil && respGet.Request.URL != nil {
					return UnwrapRedirect(respGet.Request.URL.String())
				}
			}
		}
		return cleanURL
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.Request != nil && resp.Request.URL != nil {
		return UnwrapRedirect(resp.Request.URL.String())
	}

	return cleanURL
}

// IsSpamDomain checks if the URL belongs to an affiliate or ecommerce marketplace.
func IsSpamDomain(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	for _, spam := range defaultSpamAffiliateDomains {
		if host == spam || strings.HasSuffix(host, "."+spam) {
			return true
		}
	}
	return false
}

// ContainsOnlySpamLinks checks whether all non-empty links in the slice are spam/affiliate domains.
func ContainsOnlySpamLinks(links []string) bool {
	var validCount int
	for _, l := range links {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		validCount++
		if !IsSpamDomain(trimmed) {
			return false
		}
	}
	return validCount > 0
}

// IsCustomDomain returns true if rawURL uses a custom paid domain rather than a free PaaS subdomain or localhost.
func IsCustomDomain(rawURL string) bool {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return false
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	if host == "" || host == "localhost" || strings.HasPrefix(host, "127.") || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") {
		return false
	}
	for _, suffix := range freeHostingSuffixes {
		if strings.HasSuffix(host, suffix) {
			return false
		}
	}
	return strings.Contains(host, ".")
}

// ParseGitHubOwnerRepo extracts (owner, repo) from a GitHub URL.
func ParseGitHubOwnerRepo(githubURL string) (string, string, bool) {
	trimmed := strings.TrimSpace(githubURL)
	if trimmed == "" {
		return "", "", false
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "http://") && !strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		trimmed = "https://" + trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", "", false
	}
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	if host != "github.com" {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", false
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSuffix(strings.TrimSpace(parts[1]), ".git")
	if owner == "" || repo == "" {
		return "", "", false
	}
	return owner, repo, true
}
