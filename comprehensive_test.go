package mailparser

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// TestComplexMultipart tests complex nested multipart structures
func TestComplexMultipart(t *testing.T) {
	email := `From: sender@example.com
To: recipient@example.com
Subject: Complex Multipart Test
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="outer"

--outer
Content-Type: multipart/alternative; boundary="inner"

--inner
Content-Type: text/plain; charset=utf-8

Plain text version of the email.

--inner
Content-Type: text/html; charset=utf-8

<html><body><p>HTML version of the email.</p></body></html>

--inner--

--outer
Content-Type: image/png; name="test.png"
Content-Disposition: attachment; filename="test.png"
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==

--outer--
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify structure
	if mail.Subject != "Complex Multipart Test" {
		t.Errorf("Subject = %q, want %q", mail.Subject, "Complex Multipart Test")
	}

	if len(mail.From) != 1 || mail.From[0].Address != "sender@example.com" {
		t.Errorf("From = %v, want sender@example.com", mail.From)
	}

	if mail.Text == "" {
		t.Error("Expected text body to be extracted")
	}

	if mail.HTML == "" {
		t.Error("Expected HTML body to be extracted")
	}

	if len(mail.Attachments) != 1 {
		t.Errorf("Got %d attachments, want 1", len(mail.Attachments))
	} else {
		att := mail.Attachments[0]
		if att.Filename != "test.png" {
			t.Errorf("Attachment filename = %q, want %q", att.Filename, "test.png")
		}
		if att.ContentType != "image/png" {
			t.Errorf("Attachment content-type = %q, want %q", att.ContentType, "image/png")
		}
		// Verify decoded content (1x1 transparent PNG, around 67-70 bytes)
		if len(att.Content) < 60 || len(att.Content) > 80 {
			t.Errorf("Decoded attachment size = %d, want ~67", len(att.Content))
		}
	}
}

// TestInlineAttachments tests Content-ID and inline attachments
func TestInlineAttachments(t *testing.T) {
	email := `From: sender@example.com
To: recipient@example.com
Subject: Inline Image Test
MIME-Version: 1.0
Content-Type: multipart/related; boundary="boundary123"

--boundary123
Content-Type: text/html; charset=utf-8

<html><body><img src="cid:image001@example.com"></body></html>

--boundary123
Content-Type: image/png; name="logo.png"
Content-Disposition: inline
Content-ID: <image001@example.com>
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==

--boundary123--
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(mail.Attachments) != 1 {
		t.Fatalf("Got %d attachments, want 1", len(mail.Attachments))
	}

	att := mail.Attachments[0]
	if att.ContentID != "<image001@example.com>" {
		t.Errorf("ContentID = %q, want %q", att.ContentID, "<image001@example.com>")
	}

	if att.CID != "image001@example.com" {
		t.Errorf("CID = %q, want %q", att.CID, "image001@example.com")
	}

	if att.ContentDisposition != "inline" {
		t.Errorf("ContentDisposition = %q, want %q", att.ContentDisposition, "inline")
	}
}

// TestVariousCharsets tests different character encodings
func TestVariousCharsets(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		charset  string
		wantText string
	}{
		{
			name:    "UTF-8",
			charset: "utf-8",
			email: `Subject: Test
Content-Type: text/plain; charset=utf-8

Hello World! 你好世界`,
			wantText: "Hello World! 你好世界",
		},
		{
			name:    "ISO-8859-1 (Latin-1)",
			charset: "iso-8859-1",
			email: `Subject: Test
Content-Type: text/plain; charset=iso-8859-1
Content-Transfer-Encoding: quoted-printable

Caf=E9 `,
			wantText: "Café",
		},
		{
			name:    "Windows-1252",
			charset: "windows-1252",
			email: `Subject: Test
Content-Type: text/plain; charset=windows-1252

Regular ASCII text`,
			wantText: "Regular ASCII text",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mail, err := parser.ParseBytes([]byte(tt.email))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !strings.Contains(mail.Text, strings.TrimSpace(tt.wantText)) {
				t.Errorf("Text = %q, want to contain %q", mail.Text, tt.wantText)
			}
		})
	}
}

// TestQuotedPrintableEncoding tests quoted-printable decoding
func TestQuotedPrintableEncoding(t *testing.T) {
	email := `Subject: Test
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

This is a line with a soft break=
 that continues here.

This is a line with special characters: =C3=A9 =C3=A0 =C3=A7
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should decode soft line breaks
	if !strings.Contains(mail.Text, "soft break that continues") {
		t.Errorf("Soft line break not decoded correctly: %q", mail.Text)
	}

	// Should decode UTF-8 characters
	if !strings.Contains(mail.Text, "é") || !strings.Contains(mail.Text, "à") || !strings.Contains(mail.Text, "ç") {
		t.Errorf("UTF-8 characters not decoded correctly: %q", mail.Text)
	}
}

// TestComplexAddresses tests various email address formats
func TestComplexAddresses(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    []*Address
	}{
		{
			name:   "Single address",
			header: "user@example.com",
			want: []*Address{
				{Address: "user@example.com"},
			},
		},
		{
			name:   "Address with name",
			header: "John Doe <john@example.com>",
			want: []*Address{
				{Name: "John Doe", Address: "john@example.com"},
			},
		},
		{
			name:   "Quoted name with comma",
			header: `"Doe, John" <john@example.com>`,
			want: []*Address{
				{Name: "Doe, John", Address: "john@example.com"},
			},
		},
		{
			name:   "Multiple addresses",
			header: "john@example.com, jane@example.com, bob@example.com",
			want: []*Address{
				{Address: "john@example.com"},
				{Address: "jane@example.com"},
				{Address: "bob@example.com"},
			},
		},
		{
			name:   "Mixed formats",
			header: `John Doe <john@example.com>, jane@example.com, "Bob Smith" <bob@example.com>`,
			want: []*Address{
				{Name: "John Doe", Address: "john@example.com"},
				{Address: "jane@example.com"},
				{Name: "Bob Smith", Address: "bob@example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAddressList(tt.header)
			if err != nil {
				t.Fatalf("parseAddressList() error = %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %d addresses, want %d", len(got), len(tt.want))
			}

			for i, addr := range got {
				if addr.Name != tt.want[i].Name {
					t.Errorf("Address[%d].Name = %q, want %q", i, addr.Name, tt.want[i].Name)
				}
				if addr.Address != tt.want[i].Address {
					t.Errorf("Address[%d].Address = %q, want %q", i, addr.Address, tt.want[i].Address)
				}
			}
		})
	}
}

// TestDateParsing tests various date formats
func TestDateParsing(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		wantErr  bool
		validate func(time.Time) bool
	}{
		{
			name:    "RFC1123Z",
			dateStr: "Mon, 02 Jan 2006 15:04:05 -0700",
			wantErr: false,
			validate: func(t time.Time) bool {
				return t.Year() == 2006 && t.Month() == time.January && t.Day() == 2
			},
		},
		{
			name:    "RFC822",
			dateStr: "02 Jan 06 15:04 MST",
			wantErr: false,
			validate: func(t time.Time) bool {
				return t.Month() == time.January && t.Day() == 2
			},
		},
		{
			name:    "Common email format",
			dateStr: "Thu, 13 Oct 2016 11:39:48 +0000",
			wantErr: false,
			validate: func(t time.Time) bool {
				return t.Year() == 2016 && t.Month() == time.October && t.Day() == 13
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.dateStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !tt.validate(got) {
				t.Errorf("parseDate() = %v, validation failed", got)
			}
		})
	}
}

// TestAttachmentChecksums tests checksum calculation
func TestAttachmentChecksums(t *testing.T) {
	email := `Subject: Attachment Test
Content-Type: multipart/mixed; boundary="bound"

--bound
Content-Type: text/plain

Body text

--bound
Content-Type: application/octet-stream; name="data.bin"
Content-Disposition: attachment; filename="data.bin"
Content-Transfer-Encoding: base64

SGVsbG8gV29ybGQ=

--bound--
`

	parser := NewParser()
	parser.ChecksumAlgo = "md5"
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(mail.Attachments) != 1 {
		t.Fatalf("Got %d attachments, want 1", len(mail.Attachments))
	}

	att := mail.Attachments[0]

	// Verify content
	expectedContent := "Hello World"
	if string(att.Content) != expectedContent {
		t.Errorf("Content = %q, want %q", string(att.Content), expectedContent)
	}

	// Verify MD5 checksum
	expectedHash := md5.Sum([]byte(expectedContent))
	expectedChecksum := hex.EncodeToString(expectedHash[:])

	if att.Checksum != expectedChecksum {
		t.Errorf("Checksum = %q, want %q", att.Checksum, expectedChecksum)
	}

	if att.ChecksumAlgo != "md5" {
		t.Errorf("ChecksumAlgo = %q, want %q", att.ChecksumAlgo, "md5")
	}

	if att.Size != int64(len(expectedContent)) {
		t.Errorf("Size = %d, want %d", att.Size, len(expectedContent))
	}
}

// TestMIMEWordDecoding tests RFC 2047 encoded-word decoding
func TestMIMEWordDecoding(t *testing.T) {
	tests := []struct {
		name     string
		encoded  string
		expected string
	}{
		{
			name:     "UTF-8 encoding",
			encoded:  "=?UTF-8?Q?Hello_World?=",
			expected: "Hello World",
		},
		{
			name:     "UTF-8 with special chars",
			encoded:  "=?UTF-8?Q?Caf=C3=A9?=",
			expected: "Café",
		},
		{
			name:     "Base64 encoding",
			encoded:  "=?UTF-8?B?SGVsbG8gV29ybGQ=?=",
			expected: "Hello World",
		},
		{
			name:     "ISO-8859-1",
			encoded:  "=?ISO-8859-1?Q?Caf=E9?=",
			expected: "Café",
		},
		{
			name:     "Multiple encoded words",
			encoded:  "=?UTF-8?Q?Hello?= =?UTF-8?Q?World?=",
			expected: "HelloWorld",
		},
		{
			name:     "Not encoded",
			encoded:  "Plain text",
			expected: "Plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeHeader(tt.encoded)
			if got != tt.expected {
				t.Errorf("decodeHeader(%q) = %q, want %q", tt.encoded, got, tt.expected)
			}
		})
	}
}

// TestReferences tests parsing of References header
func TestReferences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "Single reference",
			input: "<msg1@example.com>",
			want:  []string{"<msg1@example.com>"},
		},
		{
			name:  "Multiple references",
			input: "<msg1@example.com> <msg2@example.com> <msg3@example.com>",
			want: []string{
				"<msg1@example.com>",
				"<msg2@example.com>",
				"<msg3@example.com>",
			},
		},
		{
			name:  "Missing brackets",
			input: "msg1@example.com msg2@example.com",
			want: []string{
				"<msg1@example.com>",
				"<msg2@example.com>",
			},
		},
		{
			name:  "Mixed formats",
			input: "<msg1@example.com> msg2@example.com <msg3@example.com>",
			want: []string{
				"<msg1@example.com>",
				"<msg2@example.com>",
				"<msg3@example.com>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseReferences(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d references, want %d", len(got), len(tt.want))
			}

			for i, ref := range got {
				if ref != tt.want[i] {
					t.Errorf("Reference[%d] = %q, want %q", i, ref, tt.want[i])
				}
			}
		})
	}
}

// TestLargeAttachment tests handling of larger attachments
func TestLargeAttachment(t *testing.T) {
	// Create a 1KB base64 encoded attachment
	// 768 bytes of 'A' -> ~1KB base64
	encoded := make([]byte, 1024)
	// Simple base64 simulation (all 'A's encode to 'QUFB' pattern)
	for i := 0; i < len(encoded); i += 4 {
		copy(encoded[i:], []byte("QUFB"))
	}

	email := `Subject: Large Attachment
Content-Type: multipart/mixed; boundary="bound"

--bound
Content-Type: text/plain

Body

--bound
Content-Type: application/octet-stream; name="large.bin"
Content-Disposition: attachment; filename="large.bin"
Content-Transfer-Encoding: base64

` + string(encoded) + `

--bound--
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(mail.Attachments) != 1 {
		t.Fatalf("Got %d attachments, want 1", len(mail.Attachments))
	}

	att := mail.Attachments[0]
	if att.Filename != "large.bin" {
		t.Errorf("Filename = %q, want %q", att.Filename, "large.bin")
	}

	// Content should be decoded
	if len(att.Content) == 0 {
		t.Error("Attachment content is empty")
	}

	if att.Size != int64(len(att.Content)) {
		t.Errorf("Size = %d, want %d", att.Size, len(att.Content))
	}
}

// TestEmptyParts tests handling of empty parts
func TestEmptyParts(t *testing.T) {
	email := `Subject: Empty Parts
Content-Type: multipart/alternative; boundary="bound"

--bound
Content-Type: text/plain


--bound
Content-Type: text/html

<html></html>

--bound--
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if mail.Subject != "Empty Parts" {
		t.Errorf("Subject = %q, want %q", mail.Subject, "Empty Parts")
	}

	// Should handle empty parts gracefully
	t.Logf("Text: %q", mail.Text)
	t.Logf("HTML: %q", mail.HTML)
}

// TestMalformedEmail tests resilience against malformed emails
func TestMalformedEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		shouldErr bool
	}{
		{
			name: "Missing boundary in multipart",
			email: `Content-Type: multipart/mixed

This is not properly formatted
`,
			shouldErr: true,
		},
		{
			name: "Only headers",
			email: `From: sender@example.com
To: recipient@example.com
Subject: Only headers
`,
			shouldErr: false,
		},
		{
			name: "Invalid base64",
			email: `Content-Transfer-Encoding: base64

This is not valid base64!!!
`,
			shouldErr: false, // Should not error, just return garbled data
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseBytes([]byte(tt.email))
			if (err != nil) != tt.shouldErr {
				t.Errorf("Parse() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}

// TestMaxMessageSize tests size limits
func TestMaxMessageSize(t *testing.T) {
	parser := NewParser()
	parser.MaxMessageSize = 100 // 100 bytes limit

	// Small email should work
	smallEmail := `Subject: Small
From: test@example.com

Body
`
	_, err := parser.ParseBytes([]byte(smallEmail))
	if err != nil {
		t.Errorf("Small email failed: %v", err)
	}

	// Large email should fail
	largeEmail := strings.Repeat("A", 200)
	_, err = parser.ParseBytes([]byte(largeEmail))
	if err == nil {
		t.Error("Expected error for large email, got nil")
	}
}

// TestUnicodeInHeaders tests unicode in various headers
func TestUnicodeInHeaders(t *testing.T) {
	email := `From: =?UTF-8?B?44GC44GE44GG44GI44GK?= <sender@example.com>
To: recipient@example.com
Subject: =?UTF-8?Q?Test_=E2=9C=94_Unicode?=

Body
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Subject should contain checkmark
	if !strings.Contains(mail.Subject, "✔") {
		t.Errorf("Subject = %q, expected to contain ✔", mail.Subject)
	}

	// From should be decoded
	if len(mail.From) == 0 {
		t.Fatal("No From address")
	}

	t.Logf("From name: %q", mail.From[0].Name)
	t.Logf("Subject: %q", mail.Subject)
}
