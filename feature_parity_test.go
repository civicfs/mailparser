package mailparser

import (
	"strings"
	"testing"
)

// TestHTMLToText tests HTML to text conversion
func TestHTMLToText(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains []string
	}{
		{
			name:     "Simple paragraph",
			html:     "<p>Hello World</p>",
			contains: []string{"Hello World"},
		},
		{
			name:     "Multiple paragraphs",
			html:     "<p>First paragraph</p><p>Second paragraph</p>",
			contains: []string{"First paragraph", "Second paragraph"},
		},
		{
			name:     "Line breaks",
			html:     "<p>Line 1<br/>Line 2</p>",
			contains: []string{"Line 1", "Line 2"},
		},
		{
			name:     "Lists",
			html:     "<ul><li>Item 1</li><li>Item 2</li></ul>",
			contains: []string{"Item 1", "Item 2"},
		},
		{
			name:     "Nested tags",
			html:     "<div><p>Inside <strong>bold</strong> text</p></div>",
			contains: []string{"Inside", "bold", "text"},
		},
		{
			name:     "Table",
			html:     "<table><tr><td>Cell 1</td><td>Cell 2</td></tr></table>",
			contains: []string{"Cell 1", "Cell 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := HTMLToText(tt.html)
			if err != nil {
				t.Fatalf("HTMLToText() error = %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(text, expected) {
					t.Errorf("Text missing %q. Got: %q", expected, text)
				}
			}
		})
	}
}

// TestTextToHTML tests text to HTML conversion
func TestTextToHTML(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		linkify  bool
		contains []string
	}{
		{
			name:     "Single line",
			text:     "Hello World",
			linkify:  false,
			contains: []string{"<p>", "Hello World", "</p>"},
		},
		{
			name:     "Multiple paragraphs",
			text:     "First paragraph\n\nSecond paragraph",
			linkify:  false,
			contains: []string{"<p>First paragraph</p>", "<p>Second paragraph</p>"},
		},
		{
			name:     "Line breaks within paragraph",
			text:     "Line 1\nLine 2",
			linkify:  false,
			contains: []string{"<br/>"},
		},
		{
			name:     "URL linkification",
			text:     "Visit https://example.com for more info",
			linkify:  true,
			contains: []string{"<a href=", "https://example.com", "</a>"},
		},
		{
			name:     "Email linkification",
			text:     "Contact us at support@example.com",
			linkify:  true,
			contains: []string{"<a href=", "mailto:", "support@example.com", "</a>"},
		},
		{
			name:     "www linkification",
			text:     "Visit www.example.com",
			linkify:  true,
			contains: []string{"<a href=", "www.example.com", "</a>"},
		},
		{
			name:     "HTML escaping",
			text:     "Use <tags> & \"quotes\"",
			linkify:  false,
			contains: []string{"&lt;tags&gt;", "&amp;", "&quot;quotes&quot;"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := TextToHTML(tt.text, tt.linkify)

			for _, expected := range tt.contains {
				if !strings.Contains(html, expected) {
					t.Errorf("HTML missing %q. Got: %q", expected, html)
				}
			}
		})
	}
}

// TestFlowedDecoder tests format=flowed text decoding
func TestFlowedDecoder(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		delSp  bool
		output string
	}{
		{
			name:   "Basic flowing (delsp=no)",
			input:  "This is a long line \nthat continues here.",
			delSp:  false,
			output: "This is a long line that continues here.",
		},
		{
			name:   "Basic flowing (delsp=yes)",
			input:  "This is a long line \nthat continues here.",
			delSp:  true,
			output: "This is a long linethat continues here.",
		},
		{
			name:   "Hard breaks",
			input:  "Line 1\nLine 2",
			delSp:  false,
			output: "Line 1\nLine 2",
		},
		{
			name:   "Quoted text",
			input:  "> This is quoted \n> and flows.",
			delSp:  false,
			output: "> This is quoted and flows.",
		},
		{
			name:   "Mixed quote depths",
			input:  "> Level 1 \n> Level 1 continues\n>> Level 2",
			delSp:  false,
			output: "> Level 1 Level 1 continues\n>> Level 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewFlowedDecoder(tt.delSp)
			output := decoder.Decode(tt.input)

			// Normalize whitespace for comparison
			outputNorm := strings.TrimSpace(output)
			expectedNorm := strings.TrimSpace(tt.output)

			if outputNorm != expectedNorm {
				t.Errorf("Decode() = %q, want %q", outputNorm, expectedNorm)
			}
		})
	}
}

// TestCIDReplacement tests CID link replacement
func TestCIDReplacement(t *testing.T) {
	// Create an email with CID references
	email := `From: sender@example.com
Subject: Test
Content-Type: multipart/related; boundary="bound"

--bound
Content-Type: text/html

<html><img src="cid:image001"></html>

--bound
Content-Type: image/png
Content-ID: <image001>
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==

--bound--
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Replace CID links with data URIs
	err = parser.UpdateImageLinks(mail, nil)
	if err != nil {
		t.Fatalf("UpdateImageLinks() error = %v", err)
	}

	// Should have replaced cid: with data:
	if strings.Contains(mail.HTML, "cid:") {
		t.Error("CID link not replaced")
	}

	if !strings.Contains(mail.HTML, "data:image/png;base64,") {
		t.Errorf("Data URI not found in HTML: %q", mail.HTML)
	}
}

// TestJapaneseEncoding tests Japanese character encodings
func TestJapaneseEncoding(t *testing.T) {
	tests := []struct {
		name    string
		charset string
	}{
		{
			name:    "ISO-2022-JP",
			charset: "iso-2022-jp",
		},
		{
			name:    "EUC-JP",
			charset: "euc-jp",
		},
		{
			name:    "Shift-JIS",
			charset: "shift-jis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the charset is recognized
			decoder := getCharsetDecoder(tt.charset)
			if decoder == nil {
				t.Errorf("Charset %q not recognized", tt.charset)
			}

			// Test parsing an email with this charset
			email := `Subject: Test
Content-Type: text/plain; charset=` + tt.charset + `

Test message
`

			parser := NewParser()
			mail, err := parser.ParseBytes([]byte(email))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if mail.Text == "" {
				t.Error("Text should not be empty")
			}
		})
	}
}

// TestCJKEncodings tests Chinese and Korean encodings
func TestCJKEncodings(t *testing.T) {
	tests := []struct {
		name    string
		charset string
	}{
		{"GB2312", "gb2312"},
		{"GBK", "gbk"},
		{"GB18030", "gb18030"},
		{"Big5", "big5"},
		{"EUC-KR", "euc-kr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := getCharsetDecoder(tt.charset)
			if decoder == nil {
				t.Errorf("Charset %q not recognized", tt.charset)
			}
		})
	}
}

// TestHTMLToTextInEmail tests automatic HTML to text conversion
func TestHTMLToTextInEmail(t *testing.T) {
	email := `Subject: HTML Email
Content-Type: text/html; charset=utf-8

<html>
<body>
<h1>Welcome</h1>
<p>This is a <strong>test</strong> email.</p>
<p>Second paragraph.</p>
</body>
</html>
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have HTML
	if mail.HTML == "" {
		t.Error("HTML should not be empty")
	}

	// Should have automatically converted to text
	if mail.Text == "" {
		t.Error("Text should have been generated from HTML")
	}

	// Text should contain the content
	if !strings.Contains(mail.Text, "Welcome") {
		t.Errorf("Text missing 'Welcome': %q", mail.Text)
	}

	if !strings.Contains(mail.Text, "test") {
		t.Errorf("Text missing 'test': %q", mail.Text)
	}
}

// TestTextToHTMLInEmail tests automatic text to HTML conversion
func TestTextToHTMLInEmail(t *testing.T) {
	email := `Subject: Plain Text Email
Content-Type: text/plain; charset=utf-8

This is a plain text email.

Visit https://example.com for more info.
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have text
	if mail.Text == "" {
		t.Error("Text should not be empty")
	}

	// Should have TextAsHTML
	if mail.TextAsHTML == "" {
		t.Error("TextAsHTML should have been generated")
	}

	// TextAsHTML should contain HTML tags
	if !strings.Contains(mail.TextAsHTML, "<p>") {
		t.Error("TextAsHTML missing paragraph tags")
	}

	// Should have linkified the URL
	if !strings.Contains(mail.TextAsHTML, "<a href=") {
		t.Error("TextAsHTML missing link")
	}
}

// TestFormatFlowedInEmail tests format=flowed parameter
func TestFormatFlowedInEmail(t *testing.T) {
	// Note: the first line has a trailing space to indicate it flows to the next line
	emailBody := "This is a long line \nthat should be joined together.\n\nThis is a new paragraph.\n"
	email := "Subject: Flowed Text\nContent-Type: text/plain; charset=utf-8; format=flowed; delsp=yes\n\n" + emailBody

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Lines should have been joined
	if !strings.Contains(mail.Text, "This is a long linethat should be joined together") {
		t.Errorf("Flowed text not properly decoded: %q", mail.Text)
	}
}

// TestStripHTML tests HTML stripping
func TestStripHTML(t *testing.T) {
	html := "<p>Hello <strong>World</strong></p>"
	text := StripHTML(html)

	if strings.Contains(text, "<") || strings.Contains(text, ">") {
		t.Errorf("HTML tags not stripped: %q", text)
	}

	if !strings.Contains(text, "Hello") || !strings.Contains(text, "World") {
		t.Errorf("Text content missing: %q", text)
	}
}

// TestSanitizeHTML tests HTML sanitization
func TestSanitizeHTML(t *testing.T) {
	dangerous := `<div>Safe content<script>alert('xss')</script></div>`

	safe, err := SanitizeHTML(dangerous)
	if err != nil {
		t.Fatalf("SanitizeHTML() error = %v", err)
	}

	if strings.Contains(safe, "<script") {
		t.Error("Script tag not removed")
	}

	if !strings.Contains(safe, "Safe content") {
		t.Error("Safe content was removed")
	}
}

// TestParseHTMLLinks tests link extraction
func TestParseHTMLLinks(t *testing.T) {
	html := `<html>
<a href="https://example.com">Link 1</a>
<a href="mailto:test@example.com">Email</a>
</html>`

	links, err := ParseHTMLLinks(html)
	if err != nil {
		t.Fatalf("ParseHTMLLinks() error = %v", err)
	}

	if len(links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(links))
	}

	hasHTTP := false
	hasMailto := false
	for _, link := range links {
		if strings.Contains(link, "https://example.com") {
			hasHTTP = true
		}
		if strings.Contains(link, "mailto:") {
			hasMailto = true
		}
	}

	if !hasHTTP {
		t.Error("HTTP link not found")
	}
	if !hasMailto {
		t.Error("Mailto link not found")
	}
}

// TestSimpleParser tests the SimpleParser convenience function
func TestSimpleParser(t *testing.T) {
	email := `From: sender@example.com
Subject: Test
Content-Type: multipart/related; boundary="bound"

--bound
Content-Type: text/html

<html><img src="cid:img1"></html>

--bound
Content-Type: image/png
Content-ID: <img1>
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==

--bound--
`

	mail, err := SimpleParser(strings.NewReader(email), false)
	if err != nil {
		t.Fatalf("SimpleParser() error = %v", err)
	}

	// Should have replaced CID links
	if strings.Contains(mail.HTML, "cid:") {
		t.Error("CID link should have been replaced")
	}

	if !strings.Contains(mail.HTML, "data:") {
		t.Error("Data URI not found")
	}
}

// TestSkipOptions tests skip options
func TestSkipOptions(t *testing.T) {
	email := `Subject: Test
Content-Type: text/html

<html><p>Test</p></html>
`

	t.Run("Skip HTML to text", func(t *testing.T) {
		parser := NewParser()
		parser.SkipHTMLToText = true

		mail, err := parser.ParseBytes([]byte(email))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if mail.Text != "" {
			t.Error("Text should be empty when SkipHTMLToText is true")
		}
	})

	t.Run("Skip text to HTML", func(t *testing.T) {
		plainEmail := `Subject: Test
Content-Type: text/plain

Plain text
`

		parser := NewParser()
		parser.SkipTextToHTML = true

		mail, err := parser.ParseBytes([]byte(plainEmail))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if mail.TextAsHTML != "" {
			t.Error("TextAsHTML should be empty when SkipTextToHTML is true")
		}
	})
}
