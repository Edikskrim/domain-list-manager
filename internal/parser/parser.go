package parser

// Entry represents a parsed domain entry with optional metadata.
type Entry struct {
	Domain   string
	Comment  string
	Type     string // "hosts", "dnsmasq", etc.
	RawLine  string
}

// Parser defines the interface for parsing domain lists.
type Parser interface {
	Parse(content []byte) ([]*Entry, error)
	CanParse(content []byte) bool
}

// AutoParser detects the format and delegates to the appropriate parser.
type AutoParser struct {
	parsers []Parser
}

// NewAutoParser creates an AutoParser with all registered parsers.
func NewAutoParser() *AutoParser {
	ap := &AutoParser{
		parsers: []Parser{
			&HostsParser{},
			&DnsmasqParser{},
			&RawParser{},
			NewRegexParser(),
		},
	}
	return ap
}

// Parse detects the format and parses the content.
func (ap *AutoParser) Parse(content []byte) ([]*Entry, error) {
	for _, p := range ap.parsers {
		if p.CanParse(content) {
			return p.Parse(content)
		}
	}
	return nil, ErrUnknownFormat
}

// CanParse checks if the content can be parsed.
func (ap *AutoParser) CanParse(content []byte) bool {
	for _, p := range ap.parsers {
		if p.CanParse(content) {
			return true
		}
	}
	return false
}

// DetectType returns the parser type name that would handle the content.
func (ap *AutoParser) DetectType(content []byte) string {
	for _, p := range ap.parsers {
		if p.CanParse(content) {
			switch p.(type) {
			case *HostsParser:
				return "hosts"
			case *DnsmasqParser:
				return "dnsmasq"
			case *RawParser:
				return "raw"
			case *RegexParser:
				return "regex"
			default:
				return "unknown"
			}
		}
	}
	return "none"
}

// Register adds a parser to the auto-detection list.
func (ap *AutoParser) Register(p Parser) {
	ap.parsers = append([]Parser{p}, ap.parsers...)
}
