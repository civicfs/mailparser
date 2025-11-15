# Using mailparser in Go

## Installation

### Installing with a Specific Version

To use the mailparser library in your Go project, add it to your project using `go get`:

```bash
# Get the latest version
go get github.com/civicfs/mailparser@latest

# Get a specific version (using semantic version tag)
go get github.com/civicfs/mailparser@v1.2.0

# Get a specific commit
go get github.com/civicfs/mailparser@9634be1
```

### Specifying Version in go.mod

After running `go get`, your `go.mod` file will include the dependency. You can also manually edit it:

```go
module myproject

go 1.21

require (
    github.com/civicfs/mailparser v1.2.0  // Use specific version
    // or
    github.com/civicfs/mailparser latest  // Use latest
)
```

### Setting Default Version for Your Project

#### Using Go Modules (Recommended)

Go modules automatically manage versions. The version in your `go.mod` file is your project's default:

1. **To update to the latest version:**
   ```bash
   go get -u github.com/civicfs/mailparser
   go mod tidy
   ```

2. **To use a specific release:**
   ```bash
   go get github.com/civicfs/mailparser@v1.2.0
   ```

3. **To use the latest commit from main branch:**
   ```bash
   go get github.com/civicfs/mailparser@main
   ```

4. **To use the performance-optimized version (this branch):**
   ```bash
   go get github.com/civicfs/mailparser@copilot/identify-performance-enhancements
   ```

#### Vendor Dependencies (Optional)

If you want to vendor your dependencies for reproducible builds:

```bash
go mod vendor
```

This creates a `vendor/` directory with all dependencies, ensuring your builds use exactly these versions.

## Quick Start Example

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/civicfs/mailparser"
)

func main() {
    // Read email from file
    data, err := os.ReadFile("email.eml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create parser
    parser := mailparser.NewParser()
    
    // Parse email
    mail, err := parser.ParseBytes(data)
    if err != nil {
        log.Fatal(err)
    }
    
    // Access parsed data
    fmt.Println("Subject:", mail.Subject)
    fmt.Println("From:", mail.From)
    fmt.Println("To:", mail.To)
    fmt.Println("Date:", mail.Date)
    fmt.Println("\nPlain Text:")
    fmt.Println(mail.Text)
    
    // Process attachments
    fmt.Printf("\nAttachments: %d\n", len(mail.Attachments))
    for i, att := range mail.Attachments {
        fmt.Printf("  %d. %s (%s, %d bytes)\n", 
            i+1, att.Filename, att.ContentType, att.Size)
    }
}
```

## Parsing from io.Reader

```go
package main

import (
    "os"
    "github.com/civicfs/mailparser"
)

func main() {
    file, err := os.Open("email.eml")
    if err != nil {
        panic(err)
    }
    defer file.Close()
    
    parser := mailparser.NewParser()
    mail, err := parser.Parse(file)
    if err != nil {
        panic(err)
    }
    
    // Use mail...
}
```

## Parser Configuration

```go
parser := mailparser.NewParser()

// Set size limits
parser.MaxMessageSize = 50 * 1024 * 1024  // 50MB max email size
parser.MaxHTMLLength = 5 * 1024 * 1024    // 5MB max HTML

// Skip automatic conversions for better performance
parser.SkipHTMLToText = true   // Don't generate text from HTML
parser.SkipTextToHTML = true   // Don't generate TextAsHTML
parser.SkipTextLinks = true    // Don't linkify URLs in text
parser.SkipImageLinks = true   // Don't process CID images

// Use SHA256 instead of MD5 for checksums
parser.ChecksumAlgo = "sha256"

mail, err := parser.ParseBytes(emailData)
```

## Working with Attachments

```go
for _, att := range mail.Attachments {
    // Save attachment to disk
    err := os.WriteFile(att.Filename, att.Content, 0644)
    if err != nil {
        log.Printf("Error saving %s: %v", att.Filename, err)
        continue
    }
    
    fmt.Printf("Saved: %s\n", att.Filename)
    fmt.Printf("  Type: %s\n", att.ContentType)
    fmt.Printf("  Size: %d bytes\n", att.Size)
    fmt.Printf("  Checksum (%s): %s\n", att.ChecksumAlgo, att.Checksum)
    
    // Check if it's inline (related to HTML)
    if att.Related && att.CID != "" {
        fmt.Printf("  Content-ID: %s (inline image)\n", att.CID)
    }
}
```

## HTML Processing

```go
// Convert HTML to plain text
text, err := mailparser.HTMLToText(mail.HTML)
if err != nil {
    log.Printf("Error converting HTML: %v", err)
}

// Convert plain text to HTML with automatic URL linking
html := mailparser.TextToHTML(mail.Text, true)

// Sanitize HTML (remove dangerous tags)
safe, err := mailparser.SanitizeHTML(mail.HTML)
if err != nil {
    log.Printf("Error sanitizing: %v", err)
}

// Extract all links from HTML
links, err := mailparser.ParseHTMLLinks(mail.HTML)
if err != nil {
    log.Printf("Error extracting links: %v", err)
}
for _, link := range links {
    fmt.Println("Link:", link)
}
```

## Handling CID Images

```go
// Automatic conversion to data URIs
parser := mailparser.NewParser()
parser.KeepCIDLinks = false  // Convert cid: to data: URIs (default)
mail, _ := parser.Parse(reader)

// Or manually update with custom URLs
err := parser.UpdateImageLinks(mail, func(att *mailparser.Attachment) (string, error) {
    // Upload to CDN or storage
    url := fmt.Sprintf("https://cdn.example.com/%s", att.Checksum)
    // In production, you'd actually upload the att.Content here
    return url, nil
})
```

## Performance Tips

1. **Reuse Parser Instances**: Create a parser once and reuse it for multiple emails:
   ```go
   parser := mailparser.NewParser()
   for _, emailFile := range files {
       mail, err := parser.ParseBytes(emailData)
       // Process mail...
   }
   ```

2. **Skip Unnecessary Processing**: Disable features you don't need:
   ```go
   parser.SkipHTMLToText = true  // If you only need HTML
   parser.SkipTextToHTML = true  // If you only need plain text
   parser.SkipImageLinks = true  // If you don't use CID images
   ```

3. **Parallel Processing**: Parse multiple emails concurrently:
   ```go
   var wg sync.WaitGroup
   for _, emailData := range emails {
       wg.Add(1)
       go func(data []byte) {
           defer wg.Done()
           parser := mailparser.NewParser()
           mail, err := parser.ParseBytes(data)
           // Process mail...
       }(emailData)
   }
   wg.Wait()
   ```

## Version Management Best Practices

### For Applications

Lock to a specific version for stability:
```bash
go get github.com/civicfs/mailparser@v1.2.0
```

### For Libraries

Use minimum version constraints:
```go
// go.mod
require github.com/civicfs/mailparser v1.2.0
```

### Updating Dependencies

```bash
# Check for updates
go list -m -u all

# Update to latest patch version
go get -u=patch github.com/civicfs/mailparser

# Update to latest minor version
go get -u github.com/civicfs/mailparser

# View available versions
go list -m -versions github.com/civicfs/mailparser
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/civicfs/mailparser"
)

func main() {
    // Configure parser
    parser := mailparser.NewParser()
    parser.MaxMessageSize = 50 * 1024 * 1024
    parser.ChecksumAlgo = "sha256"
    
    // Open email file
    file, err := os.Open("example.eml")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    
    // Parse email
    mail, err := parser.Parse(file)
    if err != nil {
        log.Fatal(err)
    }
    
    // Display header information
    fmt.Printf("From: %s\n", formatAddresses(mail.From))
    fmt.Printf("To: %s\n", formatAddresses(mail.To))
    fmt.Printf("Subject: %s\n", mail.Subject)
    fmt.Printf("Date: %s\n", mail.Date.Format("2006-01-02 15:04:05"))
    fmt.Printf("Priority: %s\n", mail.Priority)
    
    // Display message IDs
    if mail.MessageID != "" {
        fmt.Printf("Message-ID: %s\n", mail.MessageID)
    }
    if mail.InReplyTo != "" {
        fmt.Printf("In-Reply-To: %s\n", mail.InReplyTo)
    }
    if len(mail.References) > 0 {
        fmt.Printf("References: %d\n", len(mail.References))
    }
    
    // Display content
    fmt.Println("\n--- Plain Text ---")
    fmt.Println(mail.Text)
    
    if mail.HTML != "" {
        fmt.Println("\n--- HTML Preview ---")
        fmt.Printf("HTML length: %d bytes\n", len(mail.HTML))
    }
    
    // Process attachments
    if len(mail.Attachments) > 0 {
        fmt.Printf("\n--- Attachments (%d) ---\n", len(mail.Attachments))
        for i, att := range mail.Attachments {
            fmt.Printf("%d. %s\n", i+1, att.Filename)
            fmt.Printf("   Type: %s\n", att.ContentType)
            fmt.Printf("   Size: %d bytes\n", att.Size)
            fmt.Printf("   SHA256: %s\n", att.Checksum)
            
            // Save attachment
            outPath := fmt.Sprintf("attachment_%d_%s", i+1, att.Filename)
            if err := os.WriteFile(outPath, att.Content, 0644); err != nil {
                log.Printf("Error saving attachment: %v", err)
            } else {
                fmt.Printf("   Saved to: %s\n", outPath)
            }
        }
    }
}

func formatAddresses(addrs []*mailparser.Address) string {
    if len(addrs) == 0 {
        return ""
    }
    result := ""
    for i, addr := range addrs {
        if i > 0 {
            result += ", "
        }
        result += addr.String()
    }
    return result
}
```

## Error Handling

The parser is designed to be resilient:

```go
mail, err := parser.ParseBytes(data)
if err != nil {
    // Critical parsing error - email is malformed
    log.Printf("Parse error: %v", err)
    return
}

// Parser handles gracefully:
// - Invalid base64/quoted-printable (uses original data)
// - Unknown character sets (tries UTF-8 fallback)
// - Missing MIME boundaries (returns error only if critical)
// - Empty or malformed parts (skips them)
```

## Testing

```go
package mypackage

import (
    "testing"
    "github.com/civicfs/mailparser"
)

func TestEmailParsing(t *testing.T) {
    email := `From: test@example.com
To: recipient@example.com
Subject: Test Email

This is a test email body.
`
    
    parser := mailparser.NewParser()
    mail, err := parser.ParseBytes([]byte(email))
    if err != nil {
        t.Fatalf("Parse failed: %v", err)
    }
    
    if mail.Subject != "Test Email" {
        t.Errorf("Expected subject 'Test Email', got '%s'", mail.Subject)
    }
    
    if len(mail.From) != 1 || mail.From[0].Address != "test@example.com" {
        t.Errorf("Unexpected From address")
    }
}
```

## Additional Resources

- [GitHub Repository](https://github.com/civicfs/mailparser)
- [Go Package Documentation](https://pkg.go.dev/github.com/civicfs/mailparser)
- [RFC 2822 - Internet Message Format](https://tools.ietf.org/html/rfc2822)
- [RFC 2045-2049 - MIME](https://tools.ietf.org/html/rfc2045)

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
- Submit a pull request
- Check existing documentation and examples
