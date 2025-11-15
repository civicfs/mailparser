package mailparser

import (
	"bufio"
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"
)

// mimeParser handles MIME message parsing
type mimeParser struct {
	reader  *textproto.Reader
	headers Headers
}

// newMimeParser creates a new MIME parser
func newMimeParser(r io.Reader) *mimeParser {
	return &mimeParser{
		reader: textproto.NewReader(bufio.NewReader(r)),
	}
}

// parseHeaders reads and parses email headers
func (m *mimeParser) parseHeaders() (Headers, error) {
	mimeHeader, err := m.reader.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}

	headers := make(Headers)
	for key, values := range mimeHeader {
		canonicalKey := canonicalHeader(key)
		headers[canonicalKey] = values
	}

	return headers, nil
}

// getContentType parses the Content-Type header and returns the media type and params
func getContentType(headers Headers) (string, map[string]string, error) {
	ct := headers.Get("content-type")
	if ct == "" {
		// Default to text/plain if no Content-Type header
		return "text/plain", map[string]string{"charset": "us-ascii"}, nil
	}

	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		// If parsing fails, try to extract just the media type
		parts := strings.Split(ct, ";")
		mediaType = strings.TrimSpace(parts[0])
		params = make(map[string]string)
	}

	// Ensure charset has a default
	if _, ok := params["charset"]; !ok {
		params["charset"] = "us-ascii"
	}

	return mediaType, params, nil
}

// getContentDisposition parses the Content-Disposition header
func getContentDisposition(headers Headers) (string, map[string]string, error) {
	cd := headers.Get("content-disposition")
	if cd == "" {
		return "", nil, nil
	}

	disposition, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return "", nil, err
	}

	return disposition, params, nil
}

// getTransferEncoding returns the Content-Transfer-Encoding value
func getTransferEncoding(headers Headers) string {
	encoding := headers.Get("content-transfer-encoding")
	if encoding == "" {
		return "7bit"
	}
	return strings.ToLower(strings.TrimSpace(encoding))
}

// isMultipart checks if the content type is a multipart type
func isMultipart(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "multipart/")
}

// parseMultipart parses a multipart message
func parseMultipart(r io.Reader, boundary string) ([]*mimePart, error) {
	mr := multipart.NewReader(r, boundary)
	var parts []*mimePart

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Read part headers
		headers := make(Headers)
		for key, values := range part.Header {
			canonicalKey := canonicalHeader(key)
			headers[canonicalKey] = values
		}

		// Read part body
		body, err := io.ReadAll(part)
		if err != nil {
			return nil, err
		}

		mimePart := &mimePart{
			headers: headers,
			body:    body,
		}

		parts = append(parts, mimePart)
	}

	return parts, nil
}

// mimePart represents a single MIME part
type mimePart struct {
	headers  Headers
	body     []byte
	children []*mimePart
}

// decodeHeader decodes MIME encoded-word headers (RFC 2047)
func decodeHeader(encoded string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(encoded)
	if err != nil {
		// If decoding fails, return the original
		return encoded
	}
	return decoded
}

// parseAddressList parses an address list header (From, To, Cc, etc.)
func parseAddressList(header string) ([]*Address, error) {
	if header == "" {
		return nil, nil
	}

	// Decode MIME encoded words first
	header = decodeHeader(header)

	// Use a simple address parser
	addresses := parseAddresses(header)
	return addresses, nil
}

// parseAddresses is a simple email address parser
func parseAddresses(input string) []*Address {
	var addresses []*Address

	// Split by comma, but respect quoted strings and angle brackets
	parts := splitAddresses(input)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		addr := parseAddress(part)
		if addr != nil {
			addresses = append(addresses, addr)
		}
	}

	return addresses
}

// parseAddress parses a single email address
func parseAddress(input string) *Address {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	// Check for "Name <email@example.com>" format
	if idx := strings.Index(input, "<"); idx >= 0 {
		endIdx := strings.Index(input, ">")
		if endIdx > idx {
			name := strings.TrimSpace(input[:idx])
			email := strings.TrimSpace(input[idx+1 : endIdx])

			// Remove quotes from name
			name = strings.Trim(name, "\"")

			return &Address{
				Name:    name,
				Address: email,
			}
		}
	}

	// Just an email address
	if strings.Contains(input, "@") {
		return &Address{
			Address: input,
		}
	}

	return nil
}

// splitAddresses splits an address list by commas, respecting quotes and brackets
func splitAddresses(input string) []string {
	var parts []string
	var current bytes.Buffer
	inQuotes := false
	inBrackets := false

	for i := 0; i < len(input); i++ {
		c := input[i]

		switch c {
		case '"':
			inQuotes = !inQuotes
			current.WriteByte(c)
		case '<':
			inBrackets = true
			current.WriteByte(c)
		case '>':
			inBrackets = false
			current.WriteByte(c)
		case ',':
			if !inQuotes && !inBrackets {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteByte(c)
			}
		default:
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// ensureMessageIDFormat ensures message IDs are wrapped in < >
func ensureMessageIDFormat(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}

	if !strings.HasPrefix(id, "<") {
		id = "<" + id
	}
	if !strings.HasSuffix(id, ">") {
		id = id + ">"
	}

	return id
}

// parseReferences parses the References header into a slice of message IDs
func parseReferences(header string) []string {
	if header == "" {
		return nil
	}

	header = decodeHeader(header)
	parts := strings.Fields(header)

	var refs []string
	for _, part := range parts {
		id := ensureMessageIDFormat(part)
		if id != "" {
			refs = append(refs, id)
		}
	}

	return refs
}
