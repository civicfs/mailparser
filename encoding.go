package mailparser

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime/quotedprintable"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// decodeTransferEncoding decodes content based on Content-Transfer-Encoding
func decodeTransferEncoding(data []byte, encoding string) ([]byte, error) {
	encoding = strings.ToLower(strings.TrimSpace(encoding))

	switch encoding {
	case "base64":
		return decodeBase64(data)
	case "quoted-printable":
		return decodeQuotedPrintable(data)
	case "7bit", "8bit", "binary", "":
		// No decoding needed
		return data, nil
	default:
		// Unknown encoding, return as-is
		return data, nil
	}
}

// decodeBase64 decodes base64 encoded data
func decodeBase64(data []byte) ([]byte, error) {
	// Remove whitespace which is allowed in base64 email content
	cleaned := removeWhitespace(data)

	decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(cleaned))
	decoded, err := io.ReadAll(decoder)
	if err != nil {
		// Try with RawStdEncoding (no padding)
		decoder = base64.NewDecoder(base64.RawStdEncoding, bytes.NewReader(cleaned))
		decoded, err = io.ReadAll(decoder)
		if err != nil {
			// If still fails, return original
			return data, err
		}
	}
	return decoded, nil
}

// decodeQuotedPrintable decodes quoted-printable encoded data
func decodeQuotedPrintable(data []byte) ([]byte, error) {
	reader := quotedprintable.NewReader(bytes.NewReader(data))
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return data, err
	}
	return decoded, nil
}

// removeWhitespace removes whitespace characters from byte slice
func removeWhitespace(data []byte) []byte {
	var result []byte
	for _, b := range data {
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
			result = append(result, b)
		}
	}
	return result
}

// decodeCharset decodes text from the specified charset to UTF-8
func decodeCharset(data []byte, charset string) (string, error) {
	charset = strings.ToLower(strings.TrimSpace(charset))

	// Already UTF-8 or ASCII
	if charset == "utf-8" || charset == "utf8" || charset == "us-ascii" || charset == "ascii" {
		return string(data), nil
	}

	decoder := getCharsetDecoder(charset)
	if decoder == nil {
		// Unknown charset, return as UTF-8 (might have mojibake)
		return string(data), nil
	}

	// Decode using the charset decoder
	reader := transform.NewReader(bytes.NewReader(data), decoder.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		// If decoding fails, return as-is
		return string(data), err
	}

	return string(decoded), nil
}

// getCharsetDecoder returns the appropriate encoding.Encoding for a charset name
// Prioritizes Latin-based encodings
func getCharsetDecoder(charset string) encoding.Encoding {
	charset = strings.ToLower(strings.TrimSpace(charset))
	charset = strings.ReplaceAll(charset, "_", "-")

	switch charset {
	// UTF variants
	case "utf-8", "utf8":
		return unicode.UTF8
	case "utf-16", "utf16":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	case "utf-16be", "utf16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case "utf-16le", "utf16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)

	// Latin encodings (PRIORITY)
	case "iso-8859-1", "latin1", "iso8859-1", "iso88591":
		return charmap.ISO8859_1
	case "iso-8859-2", "latin2", "iso8859-2", "iso88592":
		return charmap.ISO8859_2
	case "iso-8859-3", "latin3", "iso8859-3", "iso88593":
		return charmap.ISO8859_3
	case "iso-8859-4", "latin4", "iso8859-4", "iso88594":
		return charmap.ISO8859_4
	case "iso-8859-5", "iso8859-5", "iso88595":
		return charmap.ISO8859_5
	case "iso-8859-6", "iso8859-6", "iso88596":
		return charmap.ISO8859_6
	case "iso-8859-7", "iso8859-7", "iso88597":
		return charmap.ISO8859_7
	case "iso-8859-8", "iso8859-8", "iso88598":
		return charmap.ISO8859_8
	case "iso-8859-9", "latin5", "iso8859-9", "iso88599":
		return charmap.ISO8859_9
	case "iso-8859-10", "latin6", "iso8859-10", "iso885910":
		return charmap.ISO8859_10
	case "iso-8859-13", "iso8859-13", "iso885913":
		return charmap.ISO8859_13
	case "iso-8859-14", "iso8859-14", "iso885914":
		return charmap.ISO8859_14
	case "iso-8859-15", "latin9", "iso8859-15", "iso885915":
		return charmap.ISO8859_15
	case "iso-8859-16", "iso8859-16", "iso885916":
		return charmap.ISO8859_16

	// Windows code pages (common in Latin emails)
	case "windows-1250", "cp1250":
		return charmap.Windows1250
	case "windows-1251", "cp1251":
		return charmap.Windows1251
	case "windows-1252", "cp1252":
		return charmap.Windows1252
	case "windows-1253", "cp1253":
		return charmap.Windows1253
	case "windows-1254", "cp1254":
		return charmap.Windows1254
	case "windows-1255", "cp1255":
		return charmap.Windows1255
	case "windows-1256", "cp1256":
		return charmap.Windows1256
	case "windows-1257", "cp1257":
		return charmap.Windows1257
	case "windows-1258", "cp1258":
		return charmap.Windows1258

	// Other common encodings
	case "koi8-r":
		return charmap.KOI8R
	case "koi8-u":
		return charmap.KOI8U
	case "macintosh", "mac-roman":
		return charmap.Macintosh

	// For other encodings, we'd need to add more packages
	// For now, return nil for unsupported encodings
	default:
		return nil
	}
}

// normalizeLineEndings converts CRLF to LF
func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
