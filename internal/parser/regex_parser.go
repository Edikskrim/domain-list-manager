package parser

import (
	"regexp"
	"strings"
)

// RegexParser parses custom regex patterns to extract domains.
// Supports patterns like:
//   /domain_pattern/
//   regex:domain_pattern
//   pattern:domain_pattern
type RegexParser struct {
	patternRegex *regexp.Regexp
}

// NewRegexParser creates a new RegexParser.
func NewRegexParser() *RegexParser {
	return &RegexParser{
		patternRegex: regexp.MustCompile(`^(/|regex:|pattern:)(.+)$`),
	}
}

// Parse extracts domains using regex patterns.
func (p *RegexParser) Parse(content []byte) ([]*Entry, error) {
	lines := strings.Split(string(content), "\n")
	var entries []*Entry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := p.patternRegex.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		pattern := matches[2]
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}

		// Search for matches in the entire content
		matched := re.FindAllString(string(content), -1)
		for _, m := range matched {
			// Extract domain-like pattern
			if strings.Contains(m, ".") {
				entries = append(entries, &Entry{
					Domain:  strings.ToLower(m),
					Type:    "regex",
					RawLine: line,
				})
			}
		}
	}

	return entries, nil
}

// CanDetect checks if the content contains regex patterns.
func (p *RegexParser) CanParse(content []byte) bool {
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if p.patternRegex.MatchString(line) {
			return true
		}
	}

	return false
}
