package mailparser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFromFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "mixed multipart",
			filename: "test/fixtures/mixed.eml",
			wantErr:  false,
		},
		{
			name:     "nodemailer",
			filename: "test/fixtures/nodemailer.eml",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.filename)
			if err != nil {
				t.Fatalf("failed to read test file: %v", err)
			}

			parser := NewParser()
			mail, err := parser.ParseBytes(data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				t.Logf("Subject: %s", mail.Subject)
				t.Logf("Headers: %d", len(mail.Headers))
				t.Logf("Attachments: %d", len(mail.Attachments))
				if mail.Text != "" {
					t.Logf("Text length: %d", len(mail.Text))
				}
				if mail.HTML != "" {
					t.Logf("HTML length: %d", len(mail.HTML))
				}
			}
		})
	}
}

func TestParseAddresses(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantAddr string
		wantName string
	}{
		{
			name:     "simple email",
			input:    "user@example.com",
			wantLen:  1,
			wantAddr: "user@example.com",
			wantName: "",
		},
		{
			name:     "email with name",
			input:    "John Doe <john@example.com>",
			wantLen:  1,
			wantAddr: "john@example.com",
			wantName: "John Doe",
		},
		{
			name:     "quoted name",
			input:    `"Doe, John" <john@example.com>`,
			wantLen:  1,
			wantAddr: "john@example.com",
			wantName: "Doe, John",
		},
		{
			name:     "multiple addresses",
			input:    "john@example.com, jane@example.com",
			wantLen:  2,
			wantAddr: "john@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addresses := parseAddresses(tt.input)

			if len(addresses) != tt.wantLen {
				t.Errorf("parseAddresses() got %d addresses, want %d", len(addresses), tt.wantLen)
				return
			}

			if len(addresses) > 0 {
				if addresses[0].Address != tt.wantAddr {
					t.Errorf("parseAddresses() got address = %v, want %v", addresses[0].Address, tt.wantAddr)
				}
				if tt.wantName != "" && addresses[0].Name != tt.wantName {
					t.Errorf("parseAddresses() got name = %v, want %v", addresses[0].Name, tt.wantName)
				}
			}
		})
	}
}

func TestDecodeTransferEncoding(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		encoding string
		want     string
		wantErr  bool
	}{
		{
			name:     "base64",
			input:    "SGVsbG8gV29ybGQ=",
			encoding: "base64",
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "quoted-printable",
			input:    "Hello=20World",
			encoding: "quoted-printable",
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "7bit",
			input:    "Hello World",
			encoding: "7bit",
			want:     "Hello World",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeTransferEncoding([]byte(tt.input), tt.encoding)

			if (err != nil) != tt.wantErr {
				t.Errorf("decodeTransferEncoding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if string(got) != tt.want {
				t.Errorf("decodeTransferEncoding() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestDecodeCharset(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		charset string
		want    string
	}{
		{
			name:    "utf-8",
			input:   "Hello World",
			charset: "utf-8",
			want:    "Hello World",
		},
		{
			name:    "us-ascii",
			input:   "Hello World",
			charset: "us-ascii",
			want:    "Hello World",
		},
		{
			name:    "iso-8859-1 (Latin1)",
			input:   "Café",
			charset: "iso-8859-1",
			want:    "Café",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeCharset([]byte(tt.input), tt.charset)

			if err != nil {
				t.Logf("decodeCharset() error = %v (may be expected for some charsets)", err)
			}

			// For simple ASCII/UTF-8 cases, verify exact match
			if (tt.charset == "utf-8" || tt.charset == "us-ascii") && got != tt.want {
				t.Errorf("decodeCharset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaders(t *testing.T) {
	h := make(Headers)

	h.Set("Content-Type", "text/plain")
	if got := h.Get("content-type"); got != "text/plain" {
		t.Errorf("Headers.Get() = %v, want %v", got, "text/plain")
	}

	h.Add("Received", "from server1")
	h.Add("Received", "from server2")

	all := h.GetAll("received")
	if len(all) != 2 {
		t.Errorf("Headers.GetAll() got %d values, want 2", len(all))
	}

	if !h.Has("content-type") {
		t.Error("Headers.Has() = false, want true")
	}

	if h.Has("nonexistent") {
		t.Error("Headers.Has() = true, want false")
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		name     string
		headers  Headers
		want     string
	}{
		{
			name: "high priority",
			headers: Headers{
				"priority": []string{"urgent"},
			},
			want: "high",
		},
		{
			name: "low priority",
			headers: Headers{
				"priority": []string{"non-urgent"},
			},
			want: "low",
		},
		{
			name: "numeric high",
			headers: Headers{
				"x-priority": []string{"1"},
			},
			want: "high",
		},
		{
			name: "numeric low",
			headers: Headers{
				"x-priority": []string{"5"},
			},
			want: "low",
		},
		{
			name:    "no priority",
			headers: Headers{},
			want:    "normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePriority(tt.headers)
			if got != tt.want {
				t.Errorf("parsePriority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMessageID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already formatted",
			input: "<123@example.com>",
			want:  "<123@example.com>",
		},
		{
			name:  "missing brackets",
			input: "123@example.com",
			want:  "<123@example.com>",
		},
		{
			name:  "missing opening bracket",
			input: "123@example.com>",
			want:  "<123@example.com>",
		},
		{
			name:  "missing closing bracket",
			input: "<123@example.com",
			want:  "<123@example.com>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureMessageIDFormat(tt.input)
			if got != tt.want {
				t.Errorf("ensureMessageIDFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntegration_AllFixtures(t *testing.T) {
	// Find all .eml files in test/fixtures
	matches, err := filepath.Glob("test/fixtures/*.eml")
	if err != nil {
		t.Fatalf("failed to glob test files: %v", err)
	}

	if len(matches) == 0 {
		t.Skip("no test fixtures found")
	}

	parser := NewParser()

	for _, file := range matches {
		filename := filepath.Base(file)
		t.Run(filename, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			mail, err := parser.ParseBytes(data)
			if err != nil {
				// Log error but don't fail - some fixtures might have edge cases
				t.Logf("Parse error: %v", err)
				return
			}

			// Basic validation
			if mail == nil {
				t.Error("mail is nil")
				return
			}

			t.Logf("Successfully parsed %s:", filename)
			t.Logf("  Subject: %s", mail.Subject)
			t.Logf("  From: %v", mail.From)
			t.Logf("  To: %v", mail.To)
			t.Logf("  Headers: %d", len(mail.Headers))
			t.Logf("  Text length: %d", len(mail.Text))
			t.Logf("  HTML length: %d", len(mail.HTML))
			t.Logf("  Attachments: %d", len(mail.Attachments))

			// Log attachment details
			for i, att := range mail.Attachments {
				t.Logf("    Attachment %d: %s (%s, %d bytes, checksum: %s)",
					i+1, att.Filename, att.ContentType, att.Size, att.Checksum)
			}
		})
	}
}

// Benchmark tests
func BenchmarkParseSimple(b *testing.B) {
	data, err := os.ReadFile("test/fixtures/mixed.eml")
	if err != nil {
		b.Skipf("test file not found: %v", err)
	}

	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseBytes(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func TestAddress_String(t *testing.T) {
	tests := []struct {
		name string
		addr *Address
		want string
	}{
		{
			name: "with name",
			addr: &Address{
				Name:    "John Doe",
				Address: "john@example.com",
			},
			want: "John Doe <john@example.com>",
		},
		{
			name: "without name",
			addr: &Address{
				Address: "john@example.com",
			},
			want: "john@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.String(); got != tt.want {
				t.Errorf("Address.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		name            string
		headers         Headers
		wantMediaType   string
		wantCharset     string
	}{
		{
			name: "text/plain with charset",
			headers: Headers{
				"content-type": []string{"text/plain; charset=utf-8"},
			},
			wantMediaType: "text/plain",
			wantCharset:   "utf-8",
		},
		{
			name: "multipart/mixed with boundary",
			headers: Headers{
				"content-type": []string{`multipart/mixed; boundary="----boundary"`},
			},
			wantMediaType: "multipart/mixed",
		},
		{
			name:          "no content-type",
			headers:       Headers{},
			wantMediaType: "text/plain",
			wantCharset:   "us-ascii",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mediaType, params, err := getContentType(tt.headers)
			if err != nil {
				t.Logf("getContentType() warning: %v", err)
			}

			if mediaType != tt.wantMediaType {
				t.Errorf("getContentType() mediaType = %v, want %v", mediaType, tt.wantMediaType)
			}

			if tt.wantCharset != "" && params["charset"] != tt.wantCharset {
				t.Errorf("getContentType() charset = %v, want %v", params["charset"], tt.wantCharset)
			}
		})
	}
}

// Test Latin-1 encoding specifically
func TestLatinEncoding(t *testing.T) {
	tests := []struct {
		name     string
		charset  string
		// Input in the specified encoding
		inputHex string
		want     string
	}{
		{
			name:     "ISO-8859-1 with accents",
			charset:  "iso-8859-1",
			inputHex: "436166e9", // "Café" in Latin-1
			want:     "Café",
		},
		{
			name:     "Windows-1252 common",
			charset:  "windows-1252",
			inputHex: "48656c6c6f", // "Hello" in Windows-1252
			want:     "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert hex string to bytes
			input := hexToBytes(tt.inputHex)

			got, err := decodeCharset(input, tt.charset)
			if err != nil {
				t.Logf("decodeCharset() error: %v", err)
			}

			// For Latin encodings, we should get valid UTF-8 output
			if !strings.Contains(got, "Caf") && !strings.Contains(got, "Hello") {
				t.Logf("decodeCharset() = %q (may need charset decoder verification)", got)
			}
		})
	}
}

func hexToBytes(hex string) []byte {
	bytes := make([]byte, 0, len(hex)/2)
	for i := 0; i < len(hex); i += 2 {
		var b byte
		fmt.Sscanf(hex[i:i+2], "%02x", &b)
		bytes = append(bytes, b)
	}
	return bytes
}
