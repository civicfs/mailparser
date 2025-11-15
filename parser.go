package mailparser

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"
)

// Parse implements the main parsing logic
func (p *Parser) Parse(r io.Reader) (*Mail, error) {
	// Read all data
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Check size limit
	if p.MaxMessageSize > 0 && int64(len(data)) > p.MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds limit %d", len(data), p.MaxMessageSize)
	}

	return p.ParseBytes(data)
}

// ParseBytes parses an email from a byte slice
func (p *Parser) ParseBytes(data []byte) (*Mail, error) {
	// Find the double newline that separates headers from body
	// Headers end with \r\n\r\n or \n\n
	bodyStart := 0
	headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
	if headerEnd >= 0 {
		bodyStart = headerEnd + 4 // Skip past \r\n\r\n
	} else {
		headerEnd = bytes.Index(data, []byte("\n\n"))
		if headerEnd >= 0 {
			bodyStart = headerEnd + 2 // Skip past \n\n
		}
	}

	// Extract header and body sections
	var headerData, body []byte
	if headerEnd >= 0 {
		headerData = data[:headerEnd]
		body = data[bodyStart:]
	} else {
		// No body separator found, treat entire content as headers
		headerData = data
		body = []byte{}
	}

	// Ensure header data ends with double newline for proper MIME header parsing
	if len(headerData) > 0 {
		// Add blank line if not present
		if !bytes.HasSuffix(headerData, []byte("\r\n\r\n")) && !bytes.HasSuffix(headerData, []byte("\n\n")) {
			if bytes.HasSuffix(headerData, []byte("\r\n")) {
				headerData = append(headerData, []byte("\r\n")...)
			} else if bytes.HasSuffix(headerData, []byte("\n")) {
				headerData = append(headerData, '\n')
			} else {
				headerData = append(headerData, []byte("\n\n")...)
			}
		}
	}

	// Parse headers
	mp := newMimeParser(bytes.NewReader(headerData))
	headers, err := mp.parseHeaders()
	if err != nil {
		return nil, fmt.Errorf("failed to parse headers: %w", err)
	}

	// Create mail structure
	mail := &Mail{
		Headers:     headers,
		Attachments: make([]*Attachment, 0),
	}

	// Extract common headers
	p.extractCommonHeaders(mail, headers)

	// Parse the body
	if len(body) > 0 {
		err = p.parseBody(mail, headers, body)
		if err != nil {
			return nil, fmt.Errorf("failed to parse body: %w", err)
		}
	}

	return mail, nil
}

// extractCommonHeaders extracts commonly used headers into Mail fields
func (p *Parser) extractCommonHeaders(mail *Mail, headers Headers) {
	// Subject
	mail.Subject = decodeHeader(headers.Get("subject"))

	// Date
	dateStr := headers.Get("date")
	if dateStr != "" {
		date, err := parseDate(dateStr)
		if err == nil {
			mail.Date = date
		}
	}

	// From
	if from := headers.Get("from"); from != "" {
		mail.From, _ = parseAddressList(from)
	}

	// To
	if to := headers.Get("to"); to != "" {
		mail.To, _ = parseAddressList(to)
	}

	// Cc
	if cc := headers.Get("cc"); cc != "" {
		mail.Cc, _ = parseAddressList(cc)
	}

	// Bcc
	if bcc := headers.Get("bcc"); bcc != "" {
		mail.Bcc, _ = parseAddressList(bcc)
	}

	// Reply-To
	if replyTo := headers.Get("reply-to"); replyTo != "" {
		mail.ReplyTo, _ = parseAddressList(replyTo)
	}

	// Message-ID
	mail.MessageID = ensureMessageIDFormat(headers.Get("message-id"))

	// In-Reply-To
	mail.InReplyTo = ensureMessageIDFormat(headers.Get("in-reply-to"))

	// References
	mail.References = parseReferences(headers.Get("references"))

	// Priority
	mail.Priority = parsePriority(headers)
}

// parseBody parses the email body based on content type
func (p *Parser) parseBody(mail *Mail, headers Headers, body []byte) error {
	contentType, params, _ := getContentType(headers)

	if isMultipart(contentType) {
		boundary := params["boundary"]
		if boundary == "" {
			return fmt.Errorf("multipart message missing boundary")
		}

		return p.parseMultipartBody(mail, body, boundary, contentType)
	}

	// Single part message
	return p.parseSinglePart(mail, headers, body, true)
}

// parseMultipartBody parses a multipart message
func (p *Parser) parseMultipartBody(mail *Mail, body []byte, boundary, multipartType string) error {
	parts, err := parseMultipart(bytes.NewReader(body), boundary)
	if err != nil {
		return err
	}

	// Determine how to handle parts based on multipart type
	switch strings.ToLower(multipartType) {
	case "multipart/alternative":
		return p.handleAlternativeParts(mail, parts)
	case "multipart/mixed":
		return p.handleMixedParts(mail, parts)
	case "multipart/related":
		return p.handleRelatedParts(mail, parts)
	default:
		// Treat as mixed
		return p.handleMixedParts(mail, parts)
	}
}

// handleAlternativeParts handles multipart/alternative
func (p *Parser) handleAlternativeParts(mail *Mail, parts []*mimePart) error {
	// In alternative parts, prefer HTML over text
	var textPart, htmlPart *mimePart

	for _, part := range parts {
		ct, params, _ := getContentType(part.headers)

		// Check for nested multipart
		if isMultipart(ct) {
			boundary := params["boundary"]
			if boundary != "" {
				if err := p.parseMultipartBody(mail, part.body, boundary, ct); err != nil {
					return err
				}
				continue
			}
		}

		if ct == "text/plain" {
			textPart = part
		} else if ct == "text/html" {
			htmlPart = part
		}
	}

	// Process text part
	if textPart != nil {
		if err := p.parseSinglePart(mail, textPart.headers, textPart.body, true); err != nil {
			return err
		}
	}

	// Process HTML part
	if htmlPart != nil {
		if err := p.parseSinglePart(mail, htmlPart.headers, htmlPart.body, true); err != nil {
			return err
		}
	}

	return nil
}

// handleMixedParts handles multipart/mixed
func (p *Parser) handleMixedParts(mail *Mail, parts []*mimePart) error {
	for _, part := range parts {
		ct, params, _ := getContentType(part.headers)

		// Check for nested multipart
		if isMultipart(ct) {
			boundary := params["boundary"]
			if boundary != "" {
				if err := p.parseMultipartBody(mail, part.body, boundary, ct); err != nil {
					return err
				}
				continue
			}
		}

		// Check if this is an attachment
		disposition, _, _ := getContentDisposition(part.headers)
		isAttachment := disposition == "attachment" ||
			(!strings.HasPrefix(ct, "text/") && !isMultipart(ct))

		if err := p.parseSinglePart(mail, part.headers, part.body, !isAttachment); err != nil {
			return err
		}
	}

	return nil
}

// handleRelatedParts handles multipart/related
func (p *Parser) handleRelatedParts(mail *Mail, parts []*mimePart) error {
	// Related parts usually have a root part and related resources (images, etc.)
	for i, part := range parts {
		ct, params, _ := getContentType(part.headers)

		// Check for nested multipart
		if isMultipart(ct) {
			boundary := params["boundary"]
			if boundary != "" {
				if err := p.parseMultipartBody(mail, part.body, boundary, ct); err != nil {
					return err
				}
				continue
			}
		}

		// First part is usually the main content
		isMainContent := i == 0
		if err := p.parseSinglePart(mail, part.headers, part.body, isMainContent); err != nil {
			return err
		}
	}

	return nil
}

// parseSinglePart parses a single MIME part
func (p *Parser) parseSinglePart(mail *Mail, headers Headers, body []byte, isMainContent bool) error {
	contentType, params, _ := getContentType(headers)
	charset := params["charset"]
	if charset == "" {
		charset = "us-ascii"
	}

	// Decode transfer encoding
	transferEncoding := getTransferEncoding(headers)
	decoded, err := decodeTransferEncoding(body, transferEncoding)
	if err != nil {
		// If transfer decoding fails, use original data (be resilient)
		decoded = body
	}

	// Check if this should be treated as an attachment
	disposition, dispParams, _ := getContentDisposition(headers)
	filename := dispParams["filename"]
	if filename == "" {
		filename = params["name"]
	}

	// Determine if this is an attachment
	isAttachment := disposition == "attachment" ||
		filename != "" ||
		(!isMainContent && !strings.HasPrefix(contentType, "text/"))

	if isAttachment {
		return p.addAttachment(mail, headers, decoded, contentType, filename, disposition)
	}

	// Main content - decode charset
	text, err := decodeCharset(decoded, charset)
	if err != nil {
		return err
	}

	text = normalizeLineEndings(text)

	// Check for format=flowed
	if params["format"] == "flowed" {
		delSp := params["delsp"] == "yes"
		decoder := NewFlowedDecoder(delSp)
		text = decoder.Decode(text)
	}

	// Store based on content type
	if contentType == "text/plain" {
		mail.Text = text

		// Generate TextAsHTML if needed
		if !p.SkipTextToHTML {
			mail.TextAsHTML = TextToHTML(text, !p.SkipTextLinks)
		}
	} else if contentType == "text/html" {
		mail.HTML = text

		// Generate text version from HTML if no text exists
		if !p.SkipHTMLToText && mail.Text == "" {
			if int64(len(text)) > p.MaxHTMLLength {
				return fmt.Errorf("HTML too long for parsing: %d bytes", len(text))
			}
			plainText, err := HTMLToText(text)
			if err != nil {
				// Don't fail, just skip HTML to text conversion
				mail.Text = ""
			} else {
				mail.Text = plainText
			}
		}
	}

	return nil
}

// addAttachment adds an attachment to the mail
func (p *Parser) addAttachment(mail *Mail, headers Headers, data []byte, contentType, filename, disposition string) error {
	attachment := &Attachment{
		Filename:           decodeHeader(filename),
		ContentType:        contentType,
		ContentDisposition: disposition,
		Content:            data,
		Size:               int64(len(data)),
		Headers:            headers,
		ChecksumAlgo:       p.ChecksumAlgo,
	}

	// Calculate checksum
	attachment.Checksum = calculateChecksum(data, p.ChecksumAlgo)

	// Extract Content-ID
	contentID := headers.Get("content-id")
	if contentID != "" {
		attachment.ContentID = contentID
		// Clean CID (remove < >)
		cid := strings.TrimSpace(contentID)
		cid = strings.TrimPrefix(cid, "<")
		cid = strings.TrimSuffix(cid, ">")
		attachment.CID = cid
	}

	mail.Attachments = append(mail.Attachments, attachment)
	return nil
}

// calculateChecksum calculates a hash of the data
func calculateChecksum(data []byte, algo string) string {
	switch strings.ToLower(algo) {
	case "sha256":
		hash := sha256.Sum256(data)
		return hex.EncodeToString(hash[:])
	case "md5":
		fallthrough
	default:
		hash := md5.Sum(data)
		return hex.EncodeToString(hash[:])
	}
}

// parseDate parses an email date header
func parseDate(dateStr string) (time.Time, error) {
	// Try common email date formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 MST",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// parsePriority extracts email priority from various headers
func parsePriority(headers Headers) string {
	// Check various priority headers
	priority := headers.Get("priority")
	if priority == "" {
		priority = headers.Get("x-priority")
	}
	if priority == "" {
		priority = headers.Get("importance")
	}

	if priority == "" {
		return "normal"
	}

	priority = strings.ToLower(strings.TrimSpace(priority))

	// Handle numeric priorities
	if len(priority) > 0 && priority[0] >= '1' && priority[0] <= '5' {
		switch priority[0] {
		case '1', '2':
			return "high"
		case '4', '5':
			return "low"
		default:
			return "normal"
		}
	}

	// Handle text priorities (check non-urgent before urgent to avoid false match)
	if strings.Contains(priority, "non-urgent") || strings.Contains(priority, "low") {
		return "low"
	}
	if strings.Contains(priority, "urgent") || strings.Contains(priority, "high") {
		return "high"
	}

	return "normal"
}


// SimpleParser parses an email with default settings and CID replacement
func SimpleParser(r io.Reader, keepCIDLinks bool) (*Mail, error) {
	parser := NewParser()
	parser.KeepCIDLinks = keepCIDLinks
	
	mail, err := parser.Parse(r)
	if err != nil {
		return nil, err
	}
	
	// Replace CID links with data URIs unless keepCIDLinks is true
	if !keepCIDLinks && mail.HTML != "" {
		_ = parser.UpdateImageLinks(mail, nil)
	}
	
	return mail, nil
}

