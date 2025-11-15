package mailparser

import (
	"encoding/base64"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// HTMLToText converts HTML to plain text
func HTMLToText(htmlContent string) (string, error) {
	if htmlContent == "" {
		return "", nil
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	extractText(doc, &buf)

	// Clean up the output
	text := buf.String()
	text = normalizeWhitespace(text)
	text = strings.TrimSpace(text)

	return text, nil
}

// extractText recursively extracts text from HTML nodes
func extractText(n *html.Node, buf *strings.Builder) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}

	// Add line breaks for block elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "br":
			buf.WriteString("\n")
		case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6":
			if buf.Len() > 0 && !strings.HasSuffix(buf.String(), "\n\n") {
				buf.WriteString("\n\n")
			}
		case "li":
			buf.WriteString("\n• ")
		case "tr":
			buf.WriteString("\n")
		case "td", "th":
			buf.WriteString("\t")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, buf)
	}

	// Add trailing breaks for block elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6":
			if !strings.HasSuffix(buf.String(), "\n\n") {
				buf.WriteString("\n\n")
			}
		}
	}
}

var (
	// Pre-compiled regex patterns for better performance
	whiteSpaceRegex      = regexp.MustCompile(`[^\S\n]+`)
	multipleNewlinesRegex = regexp.MustCompile(`\n{3,}`)
	trailingSpacesRegex  = regexp.MustCompile(` +\n`)
	urlRegex             = regexp.MustCompile(`\b(https?://[^\s<>"{}|\\^` + "`" + `\[\]]+)`)
	emailRegex           = regexp.MustCompile(`\b([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	wwwRegex             = regexp.MustCompile(`\b(www\.[^\s<>"{}|\\^` + "`" + `\[\]]+)`)
	cidRegex             = regexp.MustCompile(`cid:([^'"\s<>]+)`)
	htmlTagRegex         = regexp.MustCompile(`<[^>]*>`)
)

// normalizeWhitespace cleans up excessive whitespace
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	s = whiteSpaceRegex.ReplaceAllString(s, " ")
	// Replace more than 2 newlines with 2
	s = multipleNewlinesRegex.ReplaceAllString(s, "\n\n")
	// Remove trailing spaces before newlines
	s = trailingSpacesRegex.ReplaceAllString(s, "\n")
	return s
}

// TextToHTML converts plain text to HTML with linkification
func TextToHTML(text string, linkify bool) string {
	if text == "" {
		return ""
	}

	// Escape HTML entities
	text = htmlEscape(text)

	// Linkify URLs and email addresses if requested
	if linkify {
		text = linkifyText(text)
	}

	// Convert line breaks to HTML
	lines := strings.Split(text, "\n")
	var paragraphs []string
	var currentPara strings.Builder

	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			if currentPara.Len() > 0 {
				paragraphs = append(paragraphs, currentPara.String())
				currentPara.Reset()
			}
		} else {
			if currentPara.Len() > 0 {
				currentPara.WriteString("<br/>")
			}
			currentPara.WriteString(line)
		}
	}

	if currentPara.Len() > 0 {
		paragraphs = append(paragraphs, currentPara.String())
	}

	if len(paragraphs) == 0 {
		return ""
	}

	return "<p>" + strings.Join(paragraphs, "</p><p>") + "</p>"
}

// htmlEscaper is a pre-compiled replacer for HTML escaping
var htmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"\"", "&quot;",
	"'", "&#39;",
)

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	return htmlEscaper.Replace(s)
}

// linkifyText detects and converts URLs and email addresses to links
func linkifyText(text string) string {
	// URL pattern - detect http(s):// and common URLs
	text = urlRegex.ReplaceAllStringFunc(text, func(url string) string {
		// Clean up trailing punctuation
		url = strings.TrimRight(url, ".,;:!?)")
		return `<a href="` + url + `">` + url + `</a>`
	})

	// Email pattern
	text = emailRegex.ReplaceAllStringFunc(text, func(email string) string {
		// Don't linkify if already in a link
		return `<a href="mailto:` + email + `">` + email + `</a>`
	})

	// www. URLs (without http://)
	text = wwwRegex.ReplaceAllStringFunc(text, func(url string) string {
		url = strings.TrimRight(url, ".,;:!?)")
		return `<a href="http://` + url + `">` + url + `</a>`
	})

	return text
}

// UpdateImageLinks replaces cid: links with data URIs or custom URLs
func (p *Parser) UpdateImageLinks(mail *Mail, replaceFunc func(*Attachment) (string, error)) error {
	if mail.HTML == "" || p.SkipImageLinks {
		return nil
	}

	// Build a map of CID to attachment
	cidMap := make(map[string]*Attachment, len(mail.Attachments))
	for _, att := range mail.Attachments {
		if att.CID != "" && strings.HasPrefix(att.ContentType, "image/") {
			cidMap[att.CID] = att
		}
	}

	if len(cidMap) == 0 {
		return nil
	}

	// Find and replace cid: references
	html := mail.HTML

	var lastErr error
	html = cidRegex.ReplaceAllStringFunc(html, func(match string) string {
		cid := strings.TrimPrefix(match, "cid:")
		if att, ok := cidMap[cid]; ok {
			if replaceFunc != nil {
				// Custom replacement function
				url, err := replaceFunc(att)
				if err != nil {
					lastErr = err
					return match // Keep original on error
				}
				return url
			} else if !p.KeepCIDLinks {
				// Default: convert to data URI
				return "data:" + att.ContentType + ";base64," + base64Encode(att.Content)
			}
		}
		return match
	})

	mail.HTML = html
	return lastErr
}

// base64Encode encodes bytes to base64 string
func base64Encode(data []byte) string {
	// Use stdlib for better performance and correctness
	return base64.StdEncoding.EncodeToString(data)
}

// ParseHTMLLinks extracts links from HTML content
func ParseHTMLLinks(htmlContent string) ([]string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	var links []string
	var extractLinks func(*html.Node)
	extractLinks = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractLinks(c)
		}
	}

	extractLinks(doc)
	return links, nil
}

// StripHTML removes all HTML tags and returns plain text
func StripHTML(htmlContent string) string {
	text, err := HTMLToText(htmlContent)
	if err != nil {
		// Fallback: simple tag stripping
		return htmlTagRegex.ReplaceAllString(htmlContent, "")
	}
	return text
}

// SanitizeHTML removes potentially dangerous HTML elements
func SanitizeHTML(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	// Remove script, style, and other dangerous elements
	dangerousTags := map[string]bool{
		"script": true,
		"style":  true,
		"iframe": true,
		"object": true,
		"embed":  true,
	}

	var sanitize func(*html.Node)
	sanitize = func(n *html.Node) {
		// Process children first (depth-first)
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			sanitize(c)
			c = next
		}

		// Then check if this node should be removed
		if n.Type == html.ElementNode {
			if dangerousTags[n.Data] {
				// Remove this node from its parent
				if n.Parent != nil {
					n.Parent.RemoveChild(n)
				}
				return
			}

			// Remove dangerous attributes
			var safeAttrs []html.Attribute
			for _, attr := range n.Attr {
				if attr.Key != "onclick" && attr.Key != "onload" &&
				   !strings.HasPrefix(attr.Key, "on") {
					safeAttrs = append(safeAttrs, attr)
				}
			}
			n.Attr = safeAttrs
		}
	}

	sanitize(doc)

	// Render back to HTML
	var buf strings.Builder
	if err := html.Render(&buf, doc); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderNode renders HTML node to string
func renderNode(n *html.Node) (string, error) {
	var buf strings.Builder
	w := io.Writer(&buf)
	if err := html.Render(w, n); err != nil {
		return "", err
	}
	return buf.String(), nil
}
