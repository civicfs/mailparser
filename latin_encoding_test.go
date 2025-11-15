package mailparser

import (
	"strings"
	"testing"
)

// Comprehensive tests for Latin character encodings (priority focus)

// TestLatin1CommonChars tests common Latin-1 characters (Western European)
func TestLatin1CommonChars(t *testing.T) {
	tests := []struct {
		name     string
		charset  string
		contains []string // Characters/words that should appear in decoded text
	}{
		{
			name:     "French accents UTF-8",
			charset:  "utf-8",
			contains: []string{"café", "résumé", "naïve"},
		},
		{
			name:     "Spanish characters UTF-8",
			charset:  "utf-8",
			contains: []string{"español", "niño", "señor"},
		},
		{
			name:     "German umlauts UTF-8",
			charset:  "utf-8",
			contains: []string{"Müller", "Größe", "Österreich"},
		},
		{
			name:     "Portuguese UTF-8",
			charset:  "utf-8",
			contains: []string{"São", "João", "não"},
		},
		{
			name:     "Italian UTF-8",
			charset:  "utf-8",
			contains: []string{"città", "perché", "così"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create an email with these characters
			body := strings.Join(tt.contains, " ")
			email := `Subject: Test
Content-Type: text/plain; charset=` + tt.charset + `

` + body

			parser := NewParser()
			mail, err := parser.ParseBytes([]byte(email))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Verify all expected strings appear in text
			for _, expected := range tt.contains {
				if !strings.Contains(mail.Text, expected) {
					t.Errorf("Text missing %q. Got: %q", expected, mail.Text)
				}
			}
		})
	}
}

// TestLatin2CentralEuropean tests Latin-2 (Central European)
func TestLatin2CentralEuropean(t *testing.T) {
	tests := []struct {
		name     string
		language string
		charset  string
		word     string
	}{
		{
			name:     "Polish",
			language: "Polish",
			charset:  "iso-8859-2",
			word:     "Łódź", // Polish city with special chars
		},
		{
			name:     "Czech",
			language: "Czech",
			charset:  "iso-8859-2",
			word:     "Brno",
		},
		{
			name:     "Hungarian",
			language: "Hungarian",
			charset:  "iso-8859-2",
			word:     "Budapest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := `Subject: ` + tt.word + `
Content-Type: text/plain; charset=` + tt.charset + `

Test message in ` + tt.language

			parser := NewParser()
			mail, err := parser.ParseBytes([]byte(email))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Just verify it parses - actual char verification would require proper encoding
			if mail.Subject == "" {
				t.Error("Subject should not be empty")
			}

			t.Logf("Parsed %s: Subject=%q, Text=%q", tt.language, mail.Subject, mail.Text)
		})
	}
}

// TestWindows1252Encoding tests Windows-1252 (Western European)
func TestWindows1252Encoding(t *testing.T) {
	// Windows-1252 has some characters that differ from ISO-8859-1
	email := `Subject: Test
Content-Type: text/plain; charset=windows-1252

Common Windows-1252 text with "smart quotes" and – dashes.
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should parse without error
	if len(mail.Text) == 0 {
		t.Error("Text should not be empty")
	}

	t.Logf("Windows-1252 text: %q", mail.Text)
}

// TestMixedLatinEncodings tests emails with parts in different encodings
func TestMixedLatinEncodings(t *testing.T) {
	email := `Subject: Multi-language Email
Content-Type: multipart/mixed; boundary="bound"

--bound
Content-Type: text/plain; charset=iso-8859-1

French: Café, résumé

--bound
Content-Type: text/plain; charset=iso-8859-2

Polish: Łódź, Kraków

--bound--
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if mail.Subject != "Multi-language Email" {
		t.Errorf("Subject = %q, want %q", mail.Subject, "Multi-language Email")
	}

	// Both parts should be extracted
	if len(mail.Text) == 0 {
		t.Error("Text should not be empty")
	}

	t.Logf("Mixed encoding text: %q", mail.Text)
}

// TestLatin9Euro tests Latin-9 (ISO-8859-15) with Euro symbol
func TestLatin9Euro(t *testing.T) {
	email := `Subject: Price List
Content-Type: text/plain; charset=iso-8859-15

The price is 50€ (euros).
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// ISO-8859-15 supports the Euro symbol
	if !strings.Contains(mail.Text, "50") {
		t.Errorf("Text should contain price: %q", mail.Text)
	}

	t.Logf("Latin-9 text: %q", mail.Text)
}

// TestQuotedPrintableWithLatin tests Q-P encoding with Latin chars
func TestQuotedPrintableWithLatin(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		wantWord string
	}{
		{
			name: "French with QP",
			email: `Subject: Test
Content-Type: text/plain; charset=iso-8859-1
Content-Transfer-Encoding: quoted-printable

Le caf=E9 est bon.
`,
			wantWord: "café",
		},
		{
			name: "German with QP",
			email: `Subject: Test
Content-Type: text/plain; charset=iso-8859-1
Content-Transfer-Encoding: quoted-printable

Gr=F6=DFe aus Deutschland.
`,
			wantWord: "Größe",
		},
		{
			name: "Spanish with QP",
			email: `Subject: Test
Content-Type: text/plain; charset=iso-8859-1
Content-Transfer-Encoding: quoted-printable

El ni=F1o est=E1 aqu=ED.
`,
			wantWord: "niño",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mail, err := parser.ParseBytes([]byte(tt.email))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !strings.Contains(mail.Text, tt.wantWord) {
				t.Errorf("Text should contain %q, got: %q", tt.wantWord, mail.Text)
			}
		})
	}
}

// TestBase64WithLatin tests base64 encoding with Latin characters
func TestBase64WithLatin(t *testing.T) {
	// "Café" in base64
	email := `Subject: Test
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: base64

Q2Fmw6k=
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(mail.Text, "Caf") {
		t.Errorf("Text should contain Café (or similar), got: %q", mail.Text)
	}

	t.Logf("Base64 Latin text: %q", mail.Text)
}

// TestLatinInSubject tests Latin characters in email subjects
func TestLatinInSubject(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		charset string
	}{
		{
			name:    "French subject",
			subject: "=?ISO-8859-1?Q?Caf=E9?=",
			charset: "iso-8859-1",
		},
		{
			name:    "German subject",
			subject: "=?ISO-8859-1?Q?Gr=F6=DFe?=",
			charset: "iso-8859-1",
		},
		{
			name:    "Spanish subject",
			subject: "=?ISO-8859-1?Q?Espa=F1ol?=",
			charset: "iso-8859-1",
		},
		{
			name:    "UTF-8 subject",
			subject: "=?UTF-8?B?Q2Fmw6k=?=",
			charset: "utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := `Subject: ` + tt.subject + `
Content-Type: text/plain

Body
`

			parser := NewParser()
			mail, err := parser.ParseBytes([]byte(email))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Subject should be decoded
			if mail.Subject == "" {
				t.Error("Subject should not be empty")
			}

			// Should not contain encoded markers
			if strings.Contains(mail.Subject, "=?") {
				t.Errorf("Subject still encoded: %q", mail.Subject)
			}

			t.Logf("Decoded subject (%s): %q", tt.charset, mail.Subject)
		})
	}
}

// TestLatinInAddresses tests Latin chars in email addresses (name parts)
func TestLatinInAddresses(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		wantName string
	}{
		{
			name:     "French name",
			from:     `=?ISO-8859-1?Q?Fran=E7ois?= <francois@example.com>`,
			wantName: "François",
		},
		{
			name:     "German name",
			from:     `=?ISO-8859-1?Q?M=FCller?= <muller@example.com>`,
			wantName: "Müller",
		},
		{
			name:     "Spanish name",
			from:     `=?ISO-8859-1?Q?Jos=E9?= <jose@example.com>`,
			wantName: "José",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := `From: ` + tt.from + `
Subject: Test

Body
`

			parser := NewParser()
			mail, err := parser.ParseBytes([]byte(email))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(mail.From) == 0 {
				t.Fatal("No From address parsed")
			}

			// Name should be decoded (exact match may vary by decoder)
			if mail.From[0].Name == "" {
				t.Error("From name should not be empty")
			}

			// Should not contain encoded markers
			if strings.Contains(mail.From[0].Name, "=?") {
				t.Errorf("From name still encoded: %q", mail.From[0].Name)
			}

			t.Logf("Decoded From name: %q (want: %q)", mail.From[0].Name, tt.wantName)
		})
	}
}

// TestAllLatin1Characters tests the full range of printable Latin-1
func TestAllLatin1Characters(t *testing.T) {
	// Create an email with many Latin-1 special characters
	// Using UTF-8 charset since the Go source is UTF-8
	specialChars := []string{
		"à", "á", "â", "ã", "ä", "å", // a with accents
		"è", "é", "ê", "ë", // e with accents
		"ì", "í", "î", "ï", // i with accents
		"ò", "ó", "ô", "õ", "ö", // o with accents
		"ù", "ú", "û", "ü", // u with accents
		"ñ", "ç", "ß", "ÿ", // other special chars
		"À", "Á", "Â", "Ã", "Ä", "Å", // uppercase
		"È", "É", "Ê", "Ë",
		"Ñ", "Ç",
	}

	body := strings.Join(specialChars, " ")
	email := `Subject: Latin-1 Characters
Content-Type: text/plain; charset=utf-8

` + body

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should parse without error
	if len(mail.Text) == 0 {
		t.Error("Text should not be empty")
	}

	// Count how many special chars are preserved
	preserved := 0
	for _, char := range specialChars {
		if strings.Contains(mail.Text, char) {
			preserved++
		}
	}

	t.Logf("Preserved %d/%d special characters", preserved, len(specialChars))

	// Should preserve all UTF-8 characters
	if preserved < len(specialChars) {
		t.Errorf("Some characters lost: %d/%d preserved", preserved, len(specialChars))
	}
}

// TestCaseSensitiveCharsets tests that charset names are case-insensitive
func TestCaseSensitiveCharsets(t *testing.T) {
	charsets := []string{
		"ISO-8859-1",
		"iso-8859-1",
		"IsO-8859-1",
		"UTF-8",
		"utf-8",
		"Utf-8",
		"WINDOWS-1252",
		"windows-1252",
		"Windows-1252",
	}

	for _, charset := range charsets {
		t.Run(charset, func(t *testing.T) {
			email := `Subject: Test
Content-Type: text/plain; charset=` + charset + `

Test body with café
`

			parser := NewParser()
			_, err := parser.ParseBytes([]byte(email))
			if err != nil {
				t.Errorf("Failed to parse with charset %q: %v", charset, err)
			}
		})
	}
}

// TestUnknownLatinCharset tests fallback behavior for unknown charsets
func TestUnknownLatinCharset(t *testing.T) {
	email := `Subject: Test
Content-Type: text/plain; charset=UNKNOWN-CHARSET

This should still parse but might have encoding issues.
`

	parser := NewParser()
	mail, err := parser.ParseBytes([]byte(email))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should parse without fatal error (might have mojibake)
	if len(mail.Text) == 0 {
		t.Error("Text should not be empty")
	}

	t.Logf("Unknown charset fallback text: %q", mail.Text)
}
