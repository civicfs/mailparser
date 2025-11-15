package mailparser

import (
	"os"
	"testing"
)

// Benchmark tests to ensure performance is acceptable

func BenchmarkParseSimpleEmail(b *testing.B) {
	email := `From: sender@example.com
To: recipient@example.com
Subject: Test Email
MIME-Version: 1.0
Content-Type: text/plain; charset=utf-8

This is a simple test email body.
`

	parser := NewParser()
	data := []byte(email)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseBytes(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParseMultipart(b *testing.B) {
	email := `From: sender@example.com
To: recipient@example.com
Subject: Multipart Email
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="bound"

--bound
Content-Type: text/plain; charset=utf-8

Plain text version

--bound
Content-Type: text/html; charset=utf-8

<html><body>HTML version</body></html>

--bound--
`

	parser := NewParser()
	data := []byte(email)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseBytes(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParseWithAttachment(b *testing.B) {
	email := `From: sender@example.com
To: recipient@example.com
Subject: Email with Attachment
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="bound"

--bound
Content-Type: text/plain

Body text

--bound
Content-Type: application/octet-stream; name="file.txt"
Content-Disposition: attachment; filename="file.txt"
Content-Transfer-Encoding: base64

SGVsbG8gV29ybGQhIFRoaXMgaXMgYSB0ZXN0IGF0dGFjaG1lbnQuCg==

--bound--
`

	parser := NewParser()
	data := []byte(email)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseBytes(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParseBase64Decoding(b *testing.B) {
	// Large base64 encoded content
	email := `Content-Type: text/plain
Content-Transfer-Encoding: base64

SGVsbG8gV29ybGQhIFRoaXMgaXMgYSB0ZXN0IG1lc3NhZ2UgdGhhdCB3aWxsIGJlIHJlcGVhdGVk
IGFuZCByZXBlYXRlZCB0byBtYWtlIGl0IGxvbmdlciBhbmQgdGVzdCBwZXJmb3JtYW5jZS4KSGVs
bG8gV29ybGQhIFRoaXMgaXMgYSB0ZXN0IG1lc3NhZ2UgdGhhdCB3aWxsIGJlIHJlcGVhdGVkIGFu
ZCByZXBlYXRlZCB0byBtYWtlIGl0IGxvbmdlciBhbmQgdGVzdCBwZXJmb3JtYW5jZS4K
`

	parser := NewParser()
	data := []byte(email)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseBytes(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParseQuotedPrintable(b *testing.B) {
	email := `Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

This is a test message with quoted-printable encoding.
It contains special characters: =C3=A9 =C3=A0 =C3=A7
And soft line breaks=
 that continue here.
`

	parser := NewParser()
	data := []byte(email)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseBytes(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParseAddressList(b *testing.B) {
	addresses := `John Doe <john@example.com>, "Jane Smith" <jane@example.com>, bob@example.com, "Company, Inc" <info@company.com>`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = parseAddressList(addresses)
	}
}

func BenchmarkDecodeHeader(b *testing.B) {
	encoded := "=?UTF-8?Q?Hello_World_=E2=9C=94?="

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = decodeHeader(encoded)
	}
}

func BenchmarkParseRealEmail(b *testing.B) {
	data, err := os.ReadFile("test/fixtures/mixed.eml")
	if err != nil {
		b.Skipf("Test file not found: %v", err)
	}

	parser := NewParser()

	// Verify it parses before benchmarking
	_, err = parser.ParseBytes(data)
	if err != nil {
		b.Skipf("Test file cannot be parsed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseBytes(data)
	}
}

func BenchmarkParseLargeEmail(b *testing.B) {
	data, err := os.ReadFile("test/fixtures/spam.eml")
	if err != nil {
		b.Skipf("Test file not found: %v", err)
	}

	parser := NewParser()

	// Verify it parses before benchmarking
	_, err = parser.ParseBytes(data)
	if err != nil {
		b.Skipf("Test file cannot be parsed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseBytes(data)
	}
}

func BenchmarkCharsetDecoding(b *testing.B) {
	data := []byte("Café résumé naïve")
	charset := "utf-8"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = decodeCharset(data, charset)
	}
}

// Parallel benchmarks
func BenchmarkParseSimpleEmailParallel(b *testing.B) {
	email := `From: sender@example.com
To: recipient@example.com
Subject: Test Email
Content-Type: text/plain; charset=utf-8

This is a simple test email body.
`

	data := []byte(email)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		parser := NewParser()
		for pb.Next() {
			_, err := parser.ParseBytes(data)
			if err != nil {
				b.Fatalf("Parse failed: %v", err)
			}
		}
	})
}

func BenchmarkParseMultipartParallel(b *testing.B) {
	email := `From: sender@example.com
To: recipient@example.com
Subject: Multipart Email
Content-Type: multipart/alternative; boundary="bound"

--bound
Content-Type: text/plain

Plain text

--bound
Content-Type: text/html

<html>HTML</html>

--bound--
`

	data := []byte(email)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		parser := NewParser()
		for pb.Next() {
			_, err := parser.ParseBytes(data)
			if err != nil {
				b.Fatalf("Parse failed: %v", err)
			}
		}
	})
}
