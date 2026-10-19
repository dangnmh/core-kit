package netutil

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dangnmh/core-kit/pkg/strutil"
)

// ExtractCleanTextAndLinks extracts clean text and deduplicated external URLs from HTML using github.com/PuerkitoBio/goquery.
func ExtractCleanTextAndLinks(htmlContent string) (string, []string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", nil, err
	}

	var links []string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			cleaned := UnwrapRedirect(href)
			if cleaned != "" && strings.HasPrefix(cleaned, "http") {
				links = append(links, cleaned)
			}
		}
	})

	return strings.TrimSpace(doc.Text()), strutil.UniqueStrings(links), nil
}
