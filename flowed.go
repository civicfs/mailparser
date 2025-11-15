package mailparser

import (
	"strings"
)

// FlowedDecoder decodes format=flowed text (RFC 3676)
type FlowedDecoder struct {
	DelSp bool // Whether to delete space before flowing
}

// NewFlowedDecoder creates a new flowed text decoder
func NewFlowedDecoder(delSp bool) *FlowedDecoder {
	return &FlowedDecoder{
		DelSp: delSp,
	}
}

// Decode decodes format=flowed text
func (fd *FlowedDecoder) Decode(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	var result []string
	var currentPara []string
	var quoteDepth int

	for i, line := range lines {
		// Handle CRLF
		line = strings.TrimSuffix(line, "\r")

		// Detect quote depth
		depth, unquoted := getQuoteDepth(line)

		// Check if this is a flowed line (ends with space)
		isFlowed := len(unquoted) > 0 && unquoted[len(unquoted)-1] == ' '

		// Check if quote depth changed
		if depth != quoteDepth && len(currentPara) > 0 {
			// Flush current paragraph
			result = append(result, joinFlowed(currentPara, quoteDepth, fd.DelSp))
			currentPara = nil
		}

		quoteDepth = depth

		if isFlowed {
			// Remove trailing space if delsp=yes
			if fd.DelSp && len(unquoted) > 0 {
				unquoted = unquoted[:len(unquoted)-1]
			}
			currentPara = append(currentPara, buildQuoted(depth, unquoted))
		} else {
			// Not flowed - add this line and any accumulated lines
			if len(currentPara) > 0 {
				currentPara = append(currentPara, buildQuoted(depth, unquoted))
				result = append(result, joinFlowed(currentPara, quoteDepth, fd.DelSp))
				currentPara = nil
			} else {
				result = append(result, line)
			}
		}

		// Last line
		if i == len(lines)-1 && len(currentPara) > 0 {
			result = append(result, joinFlowed(currentPara, quoteDepth, fd.DelSp))
		}
	}

	return strings.Join(result, "\n")
}

// getQuoteDepth returns the quote depth and the unquoted line
func getQuoteDepth(line string) (int, string) {
	depth := 0
	for i := 0; i < len(line); i++ {
		if line[i] == '>' {
			depth++
		} else if line[i] == ' ' && i > 0 && line[i-1] == '>' {
			// Skip space after >
			continue
		} else {
			return depth, line[i:]
		}
	}
	return depth, ""
}

// buildQuoted builds a quoted line with the given depth
func buildQuoted(depth int, text string) string {
	if depth == 0 {
		return text
	}

	prefix := strings.Repeat("> ", depth)
	return prefix + text
}

// joinFlowed joins flowed lines into a single line
func joinFlowed(lines []string, quoteDepth int, delSp bool) string {
	if len(lines) == 0 {
		return ""
	}

	if len(lines) == 1 {
		return lines[0]
	}

	// Remove quote prefixes for joining
	var unquoted []string
	for _, line := range lines {
		_, text := getQuoteDepth(line)
		unquoted = append(unquoted, text)
	}

	// Join the lines
	joined := strings.Join(unquoted, "")

	// Add quote prefix back
	return buildQuoted(quoteDepth, joined)
}

// IsFlowedText checks if text appears to be format=flowed
func IsFlowedText(text string) bool {
	lines := strings.Split(text, "\n")
	flowedCount := 0

	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		_, unquoted := getQuoteDepth(line)

		if len(unquoted) > 0 && unquoted[len(unquoted)-1] == ' ' {
			flowedCount++
		}
	}

	// If more than 20% of lines are flowed, it's probably format=flowed
	return len(lines) > 0 && float64(flowedCount)/float64(len(lines)) > 0.2
}

// UnwrapFlowed removes soft line breaks from flowed text
func UnwrapFlowed(text string, delSp bool) string {
	decoder := NewFlowedDecoder(delSp)
	return decoder.Decode(text)
}

// WrapFlowed wraps text in format=flowed format
func WrapFlowed(text string, width int, delSp bool) string {
	if width <= 0 {
		width = 78 // Default RFC 3676 width
	}

	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if len(line) <= width {
			result = append(result, line)
			continue
		}

		// Wrap long lines
		wrapped := wrapLine(line, width, delSp)
		result = append(result, wrapped...)
	}

	return strings.Join(result, "\n")
}

// wrapLine wraps a single line
func wrapLine(line string, width int, delSp bool) []string {
	if len(line) <= width {
		return []string{line}
	}

	var lines []string
	words := strings.Fields(line)
	var current strings.Builder

	for _, word := range words {
		testLen := current.Len()
		if testLen > 0 {
			testLen++ // For space
		}
		testLen += len(word)

		if testLen > width && current.Len() > 0 {
			// Flush current line
			lineText := current.String()
			if delSp {
				// Add trailing space for flow
				lineText += " "
			} else {
				lineText += " " // Always add space for hard wrap
			}
			lines = append(lines, lineText)
			current.Reset()
			current.WriteString(word)
		} else {
			if current.Len() > 0 {
				current.WriteString(" ")
			}
			current.WriteString(word)
		}
	}

	if current.Len() > 0 {
		lines = append(lines, current.String())
	}

	return lines
}
