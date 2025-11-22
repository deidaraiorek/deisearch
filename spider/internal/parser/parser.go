package parser

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Page struct {
	URL         string
	Title       string
	Description string
	Content     string
	StatusCode  int
}

type Link struct {
	URL string
}

type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(resp *http.Response, baseURL string) (*Page, []Link, error) {
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if !isEnglish(resp, doc) {
		return nil, nil, nil
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())
	description, _ := doc.Find("meta[name=description]").Attr("content")

	links := p.extractLinks(doc, baseURL)

	content := p.extractContent(doc)

	page := &Page{
		URL:         baseURL,
		Title:       title,
		Description: description,
		Content:     content,
		StatusCode:  resp.StatusCode,
	}

	return page, links, nil
}

func (p *Parser) extractLinks(doc *goquery.Document, baseURL string) []Link {
	var links []Link

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		absoluteURL := resolveURL(baseURL, href)
		if absoluteURL == "" || !isValidURL(absoluteURL) {
			return
		}

		links = append(links, Link{
			URL: absoluteURL,
		})
	})

	return links
}

func (p *Parser) extractContent(doc *goquery.Document) string {
	doc.Find("script, style, nav, header, footer, aside, iframe, noscript").Remove()

	content := doc.Find("body").Text()

	content = strings.TrimSpace(content)
	content = strings.Join(strings.Fields(content), " ")

	if len(content) > 1000000 {
		content = content[:1000000]
	}

	return content
}

func resolveURL(base, href string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}

	relURL, err := url.Parse(href)
	if err != nil {
		return ""
	}

	absoluteURL := baseURL.ResolveReference(relURL)

	absoluteURL.Fragment = ""

	normalizedURL := normalizeURL(absoluteURL)
	return normalizedURL
}

func normalizeURL(u *url.URL) string {
	trackingParams := map[string]bool{
		"utm_source":   true,
		"utm_medium":   true,
		"utm_campaign": true,
		"utm_term":     true,
		"utm_content":  true,
		"fbclid":       true,
		"gclid":        true,
		"source":       true,
		"ref":          true,
		"ssrc":         true,
	}

	if u.RawQuery != "" {
		query := u.Query()
		cleanQuery := url.Values{}

		for key, values := range query {
			if !trackingParams[key] {
				cleanQuery[key] = values
			}
		}

		if len(cleanQuery) > 0 {
			u.RawQuery = cleanQuery.Encode()
		} else {
			u.RawQuery = ""
		}
	}
	return u.String()
}

func isValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	path := strings.ToLower(u.Path)
	skipExtensions := []string{
		".pdf", ".jpg", ".jpeg", ".png", ".gif", ".svg",
		".css", ".js", ".zip", ".tar", ".gz",
		".exe", ".dmg", ".iso",
		".mp4", ".avi", ".mov",
		".mp3", ".wav",
	}

	for _, ext := range skipExtensions {
		if strings.HasSuffix(path, ext) {
			return false
		}
	}
	return true
}

func ExtractDomain(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return u.Host
}

func isEnglish(resp *http.Response, doc *goquery.Document) bool {
	contentLang := resp.Header.Get("Content-Language")
	if contentLang != "" {
		lang := strings.ToLower(strings.Split(contentLang, ",")[0])
		lang = strings.TrimSpace(strings.Split(lang, "-")[0])
		if lang != "en" {
			return false
		}
	}

	htmlLang, exists := doc.Find("html").Attr("lang")
	if exists {
		lang := strings.ToLower(strings.Split(htmlLang, "-")[0])
		if lang != "" && lang != "en" {
			return false
		}
	}
	return true
}
