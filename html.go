package mailparser

import (
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

// normalizeWhitespace cleans up excessive whitespace
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	s = regexp.MustCompile(`[^\S\n]+`).ReplaceAllString(s, " ")
	// Replace more than 2 newlines with 2
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	// Remove trailing spaces before newlines
	s = regexp.MustCompile(` +\n`).ReplaceAllString(s, "\n")
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

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// linkifyText detects and converts URLs and email addresses to links
func linkifyText(text string) string {
	// URL pattern - detect http(s):// and common URLs
	urlPattern := regexp.MustCompile(`\b(https?://[^\s<>"{}|\\^` + "`" + `\[\]]+)`)
	text = urlPattern.ReplaceAllStringFunc(text, func(url string) string {
		// Clean up trailing punctuation
		url = strings.TrimRight(url, ".,;:!?)")
		return `<a href="` + url + `">` + url + `</a>`
	})

	// Email pattern
	emailPattern := regexp.MustCompile(`\b([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	text = emailPattern.ReplaceAllStringFunc(text, func(email string) string {
		// Don't linkify if already in a link
		return `<a href="mailto:` + email + `">` + email + `</a>`
	})

	// www. URLs (without http://)
	wwwPattern := regexp.MustCompile(`\b(www\.[^\s<>"{}|\\^` + "`" + `\[\]]+)`)
	text = wwwPattern.ReplaceAllStringFunc(text, func(url string) string {
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
	cidMap := make(map[string]*Attachment)
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
	cidPattern := regexp.MustCompile(`cid:([^'"\s<>]+)`)

	var lastErr error
	html = cidPattern.ReplaceAllStringFunc(html, func(match string) string {
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
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	var result strings.Builder
	result.Grow((len(data) + 2) / 3 * 4)

	for i := 0; i < len(data); i += 3 {
		b := [3]byte{}
		n := 0
		for j := 0; j < 3 && i+j < len(data); j++ {
			b[j] = data[i+j]
			n++
		}

		result.WriteByte(base64Table[(b[0]&0xFC)>>2])
		result.WriteByte(base64Table[((b[0]&0x03)<<4)|((b[1]&0xF0)>>4)])

		if n > 1 {
			result.WriteByte(base64Table[((b[1]&0x0F)<<2)|((b[2]&0xC0)>>6)])
		} else {
			result.WriteByte('=')
		}

		if n > 2 {
			result.WriteByte(base64Table[b[2]&0x3F])
		} else {
			result.WriteByte('=')
		}
	}

	return result.String()
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
		return regexp.MustCompile(`<[^>]*>`).ReplaceAllString(htmlContent, "")
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
