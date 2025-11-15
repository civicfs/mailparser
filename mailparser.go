// Package mailparser provides advanced email parsing capabilities for Go.
// It handles MIME multipart messages, character encoding, attachments, and more.
package mailparser

import (
	"strings"
	"time"
)

// Mail represents a parsed email message with all its components.
type Mail struct {
	// Headers contains all email headers as a map
	Headers Headers

	// Subject is the decoded email subject
	Subject string

	// From contains the sender address(es)
	From []*Address

	// To contains the recipient address(es)
	To []*Address

	// Cc contains the carbon copy address(es)
	Cc []*Address

	// Bcc contains the blind carbon copy address(es)
	Bcc []*Address

	// ReplyTo contains the reply-to address(es)
	ReplyTo []*Address

	// Date is the parsed email date
	Date time.Time

	// MessageID is the unique message identifier
	MessageID string

	// InReplyTo references the message this is replying to
	InReplyTo string

	// References contains message IDs this email references
	References []string

	// Text is the plain text body
	Text string

	// HTML is the HTML body
	HTML string

	// TextAsHTML is the plain text converted to HTML
	TextAsHTML string

	// Attachments contains all file attachments
	Attachments []*Attachment

	// Priority indicates email priority (high, normal, low)
	Priority string
}

// Address represents an email address with optional display name.
type Address struct {
	Name    string // Display name (e.g., "John Doe")
	Address string // Email address (e.g., "john@example.com")
}

// String returns the formatted address string.
func (a *Address) String() string {
	if a.Name != "" {
		return a.Name + " <" + a.Address + ">"
	}
	return a.Address
}

// Attachment represents a file attachment in an email.
type Attachment struct {
	// Filename is the name of the attached file
	Filename string

	// ContentType is the MIME type of the attachment
	ContentType string

	// ContentDisposition is "attachment" or "inline"
	ContentDisposition string

	// ContentID is the Content-ID header value
	ContentID string

	// CID is the cleaned content ID (without < >)
	CID string

	// Content is the decoded attachment data
	Content []byte

	// Size is the size of the attachment in bytes
	Size int64

	// Checksum is the hash of the attachment content
	Checksum string

	// ChecksumAlgo is the algorithm used (e.g., "md5", "sha256")
	ChecksumAlgo string

	// PartID identifies the MIME part
	PartID string

	// Related indicates if attachment is related to HTML content
	Related bool

	// Headers contains the MIME part headers
	Headers Headers
}

// Headers represents email headers as a map of header names to values.
// Header names are case-insensitive and stored in lowercase.
type Headers map[string][]string

// Get returns the first value for the given header name.
// Returns empty string if the header is not present.
func (h Headers) Get(name string) string {
	values := h.GetAll(name)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// GetAll returns all values for the given header name.
func (h Headers) GetAll(name string) []string {
	// Headers are stored in lowercase
	return h[canonicalHeader(name)]
}

// Set sets a header to a single value, replacing any existing values.
func (h Headers) Set(name, value string) {
	h[canonicalHeader(name)] = []string{value}
}

// Add adds a value to the header, preserving existing values.
func (h Headers) Add(name, value string) {
	key := canonicalHeader(name)
	h[key] = append(h[key], value)
}

// Has returns true if the header exists.
func (h Headers) Has(name string) bool {
	_, exists := h[canonicalHeader(name)]
	return exists
}

// canonicalHeader converts header names to lowercase for consistent lookup.
func canonicalHeader(name string) string {
	// Simple lowercase conversion - email headers are case-insensitive
	return toLower(name)
}

// toLower converts a string to lowercase.
func toLower(s string) string {
	// Use stdlib for better performance - it's optimized with assembly
	return strings.ToLower(s)
}

// Parser is the main email parser.
type Parser struct {
	// MaxMessageSize limits the maximum email size in bytes (0 = unlimited)
	MaxMessageSize int64

	// MaxHTMLLength limits HTML parsing to prevent DoS (default: 10MB)
	MaxHTMLLength int64

	// SkipHTMLToText skips HTML to text conversion
	SkipHTMLToText bool

	// SkipTextToHTML skips text to HTML conversion
	SkipTextToHTML bool

	// SkipTextLinks skips linkification in text
	SkipTextLinks bool

	// SkipImageLinks skips CID image link processing
	SkipImageLinks bool

	// KeepCIDLinks keeps cid: links instead of converting to data URIs
	KeepCIDLinks bool

	// KeepDeliveryStatus keeps message/delivery-status as separate parts
	KeepDeliveryStatus bool

	// ChecksumAlgo specifies hash algorithm for attachments (default: "md5")
	ChecksumAlgo string
}

// NewParser creates a new email parser with default settings.
func NewParser() *Parser {
	return &Parser{
		MaxHTMLLength: 10 * 1024 * 1024, // 10MB default
		ChecksumAlgo:  "md5",
	}
}

// Parse, ParseBytes, and SimpleParser are implemented in parser.go
