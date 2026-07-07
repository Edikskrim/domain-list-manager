package parser

import (
	"strings"
)

// HostsParser parses /etc/hosts format content.
// Examples:
//   0.0.0.0 example.com
//   127.0.0.1 ads.example.com
//   ::1 example.com
//   # Comment line
type HostsParser struct{}

var hostsLines = []string{
	"0.0.0.0",
	"127.0.0.1",
	"127.0.0.0",
	"192.168.0.1",
	"192.168.1.1",
	"::1",
	"::",
}

// Parse converts hosts format to domain entries.
func (p *HostsParser) Parse(content []byte) ([]*Entry, error) {
	if !p.CanParse(content) {
		return nil, ErrUnknownFormat
	}

	lines := strings.Split(string(content), "\n")
	var entries []*Entry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// Extract comment (everything after IP and domain)
		var comment string
		if len(parts) > 2 {
			comment = strings.Join(parts[2:], " ")
		}

		// Parse all domains from this line
		for _, domain := range parts[1:] {
			// Skip IP addresses
			if isIP(domain) {
				continue
			}
			entries = append(entries, &Entry{
				Domain:  strings.ToLower(domain),
				Comment: comment,
				Type:    "hosts",
				RawLine: line,
			})
		}
	}

	return entries, nil
}

// CanDetect checks if the content looks like hosts format.
func (p *HostsParser) CanParse(content []byte) bool {
	lines := strings.Split(string(content), "\n")
	hostCount := 0
	lineCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lineCount++

		if lineCount > 10 {
			break
		}

		for _, prefix := range hostsLines {
			if strings.HasPrefix(line, prefix) {
				hostCount++
				break
			}
		}
	}

	return hostCount > 0 && lineCount > 0
}

func isIP(s string) bool {
	// Simple check for IP-like strings
	if len(s) < 7 || len(s) > 39 {
		return false
	}
	for _, c := range s {
		if c != '.' && c != ':' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}
