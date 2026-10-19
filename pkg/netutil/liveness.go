package netutil

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dangnmh/core-kit/pkg/strutil"
)

var (
	ogImageMetaRegex = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']`)
	ogImageAltRegex  = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`)
	metaDescRegex    = regexp.MustCompile(`(?i)<meta[^>]+(?:name|property)=["'](?:description|og:description)["'][^>]+content=["']([^"']+)["']`)
	metaDescAltRegex = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']+)["'][^>]+(?:name|property)=["'](?:description|og:description)["']`)
	htmlLangRegex    = regexp.MustCompile(`(?i)<html[^>]+lang=["']([a-zA-Z-]+)["']`)
	htmlTitleRegex   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	i18nMarkerRegex  = regexp.MustCompile(`(?i)(?:hreflang=["']en|next-intl|react-i18next|i18next|["']/en(?:/|["'])|data-locale=["']en|lang-switcher|language-selector|\bEN\s*[|/]\s*VI\b|\bVI\s*[|/]\s*EN\b)`)
)

// DemoProbeResult holds the liveness, iframe security, OG metadata, and Global/English UI signals for a probed URL.
type DemoProbeResult struct {
	IsAlive         bool   `json:"is_alive"`
	StatusCode      int    `json:"status_code"`
	CanEmbedIframe  bool   `json:"can_embed_iframe"`
	OGImage         string `json:"og_image,omitempty"`
	Title           string `json:"title,omitempty"`
	MetaDescription string `json:"meta_description,omitempty"`
	HTMLLang        string `json:"html_lang,omitempty"`
	HasI18nMarkers  bool   `json:"has_i18n_markers,omitempty"`
	HasEnglishUI    bool   `json:"has_english_ui,omitempty"`
	SupportsGlobal  bool   `json:"supports_global,omitempty"`
}

// ProbeDemoURL checks whether demoURL is alive, allows iframe embedding, and extracts OG meta tags & Global language signals.
// Blocks SSRF to private/loopback/metadata IP ranges unless allowPrivate is true.
func ProbeDemoURL(ctx context.Context, client *http.Client, demoURL string, allowPrivate bool) (*DemoProbeResult, error) {
	trimmed := strings.TrimSpace(demoURL)
	if trimmed == "" {
		return &DemoProbeResult{}, nil
	}

	if err := ValidateSafeProbeURL(ctx, trimmed, allowPrivate); err != nil {
		return &DemoProbeResult{IsAlive: false, CanEmbedIframe: false}, err
	}

	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimmed, nil)
	if err != nil {
		return &DemoProbeResult{}, err
	}
	req.Header.Set("User-Agent", "CoreKit-LivenessProbe/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return &DemoProbeResult{
			IsAlive:        false,
			CanEmbedIframe: false,
		}, err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	htmlSnippet := string(bodyBytes)

	isAlive := resp.StatusCode >= 200 && resp.StatusCode <= 399
	canEmbed := isAlive

	if canEmbed {
		xfo := strings.ToUpper(resp.Header.Get("X-Frame-Options"))
		if strings.Contains(xfo, "DENY") || strings.Contains(xfo, "SAMEORIGIN") {
			canEmbed = false
		}

		csp := strings.ToLower(resp.Header.Get("Content-Security-Policy"))
		if strings.Contains(csp, "frame-ancestors 'none'") ||
			strings.Contains(csp, "frame-ancestors 'self'") ||
			strings.Contains(csp, "frame-ancestors none") ||
			strings.Contains(csp, "frame-ancestors self") {
			canEmbed = false
		}
	}

	ogImage := ""
	if m := ogImageMetaRegex.FindStringSubmatch(htmlSnippet); len(m) > 1 {
		ogImage = strings.TrimSpace(m[1])
	} else if m := ogImageAltRegex.FindStringSubmatch(htmlSnippet); len(m) > 1 {
		ogImage = strings.TrimSpace(m[1])
	}

	title := ""
	if m := htmlTitleRegex.FindStringSubmatch(htmlSnippet); len(m) > 1 {
		title = strings.TrimSpace(m[1])
	}

	metaDesc := ""
	if m := metaDescRegex.FindStringSubmatch(htmlSnippet); len(m) > 1 {
		metaDesc = strings.TrimSpace(m[1])
	} else if m := metaDescAltRegex.FindStringSubmatch(htmlSnippet); len(m) > 1 {
		metaDesc = strings.TrimSpace(m[1])
	}

	htmlLang := ""
	if m := htmlLangRegex.FindStringSubmatch(htmlSnippet); len(m) > 1 {
		htmlLang = strings.ToLower(strings.TrimSpace(m[1]))
	}

	hasI18n := i18nMarkerRegex.MatchString(htmlSnippet)
	combinedMeta := strings.TrimSpace(title + " " + metaDesc)
	hasEnglishUI := strutil.IsEnglishText(combinedMeta)

	return &DemoProbeResult{
		IsAlive:         isAlive,
		StatusCode:      resp.StatusCode,
		CanEmbedIframe:  canEmbed,
		OGImage:         ogImage,
		Title:           title,
		MetaDescription: metaDesc,
		HTMLLang:        htmlLang,
		HasI18nMarkers:  hasI18n,
		HasEnglishUI:    hasEnglishUI,
		SupportsGlobal:  hasI18n || hasEnglishUI,
	}, nil
}
