package parser

import (
	"regexp"
	"strings"
)

// RawParser parses raw domain list format (one domain per line).
// Handles:
//   - example.com
//   - www.example.com
//   - *.example.com
//   - +.example.com
type RawParser struct {
	domainRegex *regexp.Regexp
}

// NewRawParser creates a new RawParser.
func NewRawParser() *RawParser {
	return &RawParser{
		domainRegex: regexp.MustCompile(`^[*+\.]?([a-z0-9][a-z0-9\-]*\.[a-z0-9][a-z0-9\-]*)$`),
	}
}

// Parse converts raw format to domain entries.
func (p *RawParser) Parse(content []byte) ([]*Entry, error) {
	lines := strings.Split(string(content), "\n")
	var entries []*Entry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove wildcard prefixes and common subdomains
		domain := strings.TrimLeft(line, "*+.")
		domain = strings.TrimPrefix(domain, "www.")
		domain = strings.TrimPrefix(domain, "mail.")
		domain = strings.TrimPrefix(domain, "api.")
		domain = strings.TrimPrefix(domain, "cdn.")
		domain = strings.TrimPrefix(domain, "static.")
		domain = strings.TrimPrefix(domain, "img.")
		domain = strings.TrimPrefix(domain, "media.")
		domain = strings.TrimPrefix(domain, "assets.")
		domain = strings.TrimPrefix(domain, "app.")
		domain = strings.TrimPrefix(domain, "dev.")
		
		// Validate domain
		if !p.isValidDomain(domain) {
			continue
		}

		entries = append(entries, &Entry{
			Domain:  strings.ToLower(domain),
			Type:    "raw",
			RawLine: line,
		})
	}

	return entries, nil
}

// CanDetect checks if the content looks like raw format.
func (p *RawParser) CanParse(content []byte) bool {
	lines := strings.Split(string(content), "\n")
	domainCount := 0
	lineCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lineCount++

		if lineCount > 20 {
			break
		}

		domain := strings.TrimLeft(line, "*+.")
		if p.isValidDomain(domain) {
			domainCount++
		}
	}

	return domainCount > 0 && lineCount > 0
}

func (p *RawParser) isValidDomain(s string) bool {
	// Remove prefixes
	domain := strings.TrimLeft(s, "*+.")
	
	// Basic domain validation
	if len(domain) < 3 || len(domain) > 253 {
		return false
	}

	// Must contain a dot
	if !strings.Contains(domain, ".") {
		return false
	}

	// Check for valid characters
	for _, c := range domain {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '.') {
			return false
		}
	}

	// Must not start or end with hyphen
	if domain[0] == '-' || domain[len(domain)-1] == '-' {
		return false
	}

	return true
}
