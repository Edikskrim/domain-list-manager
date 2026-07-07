package parser

import (
	"strings"
)

// DnsmasqParser parses dnsmasq format content.
// Examples:
//   address=/example.com/127.0.0.1
//   server=/example.com/8.8.8.8
//   blacklist-domain=example.com
//   addn-hosts=/etc/dnsmasq.d/hosts
type DnsmasqParser struct{}

var dnsmasqPatterns = map[string]bool{
	"address=/":           true,
	"server=/":            true,
	"blacklist-domain=":  true,
	"whitelist-domain=":  true,
	"addn-hosts=":        true,
	"host-record=":       true,
	"dhcp-host=":         true,
}

// Parse converts dnsmasq format to domain entries.
func (p *DnsmasqParser) Parse(content []byte) ([]*Entry, error) {
	lines := strings.Split(string(content), "\n")
	var entries []*Entry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := strings.TrimSpace(parts[1])

		var domain string

		switch {
		case strings.HasPrefix(key, "blacklist-domain"):
			domain = value
		case strings.HasPrefix(key, "whitelist-domain"):
			domain = value
		case strings.HasPrefix(key, "host-record"):
			domain = value
		case key == "address":
			// address=/example.com/127.0.0.1
			addrParts := strings.Split(value, "/")
			if len(addrParts) >= 2 {
				domain = addrParts[1]
			}
		case key == "server":
			// server=/example.com/8.8.8.8
			serverParts := strings.Split(value, "/")
			if len(serverParts) >= 2 {
				domain = serverParts[1]
			}
		default:
			continue
		}

		if domain == "" {
			continue
		}

		entries = append(entries, &Entry{
			Domain:  strings.ToLower(domain),
			Type:    "dnsmasq",
			RawLine: line,
		})
	}

	return entries, nil
}

// CanDetect checks if the content looks like dnsmasq format.
func (p *DnsmasqParser) CanParse(content []byte) bool {
	lines := strings.Split(string(content), "\n")
	matchCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		for pattern := range dnsmasqPatterns {
			if strings.HasPrefix(line, pattern) {
				matchCount++
				break
			}
		}

		if matchCount >= 3 {
			return true
		}
	}

	return matchCount > 0
}
