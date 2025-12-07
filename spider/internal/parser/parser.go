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
	description := p.extractDescription(doc)

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

func (p *Parser) ParseHTML(htmlContent string, baseURL string) (*Page, []Link, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, nil, err
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())
	description := p.extractDescription(doc)

	links := p.extractLinks(doc, baseURL)

	content := p.extractContent(doc)

	page := &Page{
		URL:         baseURL,
		Title:       title,
		Description: description,
		Content:     content,
		StatusCode:  200,
	}

	return page, links, nil
}

func (p *Parser) extractDescription(doc *goquery.Document) string {
	metaSelectors := []string{
		"meta[name='description']",
		"meta[property='og:description']",
		"meta[name='twitter:description']",
		"meta[property='description']",
	}

	for _, selector := range metaSelectors {
		if content, exists := doc.Find(selector).Attr("content"); exists && strings.TrimSpace(content) != "" {
			return strings.TrimSpace(content)
		}
	}

	return ""
}

func (p *Page) HasSufficientContent() bool {
	allText := p.Description + " " + p.Content
	allText = strings.TrimSpace(allText)

	return len(allText) >= 100
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
	contentDoc := doc.Clone()

	// Remove non-content elements
	contentDoc.Find("script, style, nav, header, footer, aside, iframe, noscript, form, button").Remove()

	// Strategy 1: Try semantic HTML5 elements first (article, main)
	var content string

	// Try <article> tag
	if article := contentDoc.Find("article").First(); article.Length() > 0 {
		content = article.Text()
	}

	// Try <main> tag if article didn't work well
	if len(strings.TrimSpace(content)) < 100 {
		if main := contentDoc.Find("main").First(); main.Length() > 0 {
			content = main.Text()
		}
	}

	// Strategy 2: Try common content class/id selectors
	if len(strings.TrimSpace(content)) < 100 {
		contentSelectors := []string{
			"#content", ".content", "#main-content", ".main-content",
			"#article", ".article", "#post", ".post",
			".entry-content", ".post-content", ".article-content",
			"[role='main']", ".page-content", "#page-content",
		}

		for _, selector := range contentSelectors {
			if elem := contentDoc.Find(selector).First(); elem.Length() > 0 {
				text := elem.Text()
				if len(strings.TrimSpace(text)) > len(strings.TrimSpace(content)) {
					content = text
				}
			}
		}
	}

	// Strategy 3: Aggregate all paragraph tags
	if len(strings.TrimSpace(content)) < 100 {
		var paragraphs []string
		contentDoc.Find("p").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if len(text) > 20 { // Only include substantial paragraphs
				paragraphs = append(paragraphs, text)
			}
		})
		if len(paragraphs) > 0 {
			content = strings.Join(paragraphs, " ")
		}
	}

	// Strategy 4: Fall back to body if nothing else worked
	if len(strings.TrimSpace(content)) < 100 {
		content = contentDoc.Find("body").Text()
	}

	// Clean up whitespace
	content = strings.TrimSpace(content)
	content = strings.Join(strings.Fields(content), " ")

	// Limit size
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
	u.RawQuery = ""

	if u.Path == "/" {
		u.Path = ""
	} else if strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	return u.String()
}

func NormalizeURLString(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	return normalizeURL(u)
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
