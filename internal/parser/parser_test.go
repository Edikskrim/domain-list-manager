package parser

import (
	"testing"
)

func TestHostsParser_Parse(t *testing.T) {
	content := []byte(`# hosts file
0.0.0.0 example.com
127.0.0.1 ads.example.com
::1 test.org
# Comment
0.0.0.0 block.net`)

	p := &HostsParser{}
	entries, err := p.Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(entries))
	}

	expected := []string{"example.com", "ads.example.com", "test.org", "block.net"}
	for i, e := range entries {
		if e.Domain != expected[i] {
			t.Errorf("entry %d: expected %s, got %s", i, expected[i], e.Domain)
		}
		t.Logf("[%d] %s (type=%s)", i, e.Domain, e.Type)
	}
}

func TestHostsParser_CanParse(t *testing.T) {
	p := &HostsParser{}

	if !p.CanParse([]byte("0.0.0.0 example.com\n")) {
		t.Error("should detect hosts format")
	}

	if p.CanParse([]byte("example.com\n")) {
		t.Error("should not detect raw format as hosts")
	}
}

func TestDnsmasqParser_Parse(t *testing.T) {
	content := []byte(`# dnsmasq config
address=/example.com/127.0.0.1
blacklist-domain=ads.net
server=/dns.example.com/8.8.8.8
whitelist-domain=trusted.org`)

	p := &DnsmasqParser{}
	entries, err := p.Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(entries))
	}

	expected := []string{"example.com", "ads.net", "dns.example.com", "trusted.org"}
	for i, e := range entries {
		if e.Domain != expected[i] {
			t.Errorf("entry %d: expected %s, got %s", i, expected[i], e.Domain)
		}
		t.Logf("[%d] %s (type=%s)", i, e.Domain, e.Type)
	}
}

func TestDnsmasqParser_CanParse(t *testing.T) {
	p := &DnsmasqParser{}

	if !p.CanParse([]byte("address=/example.com/127.0.0.1\n")) {
		t.Error("should detect dnsmasq format")
	}
}

func TestRawParser_Parse(t *testing.T) {
	content := []byte(`# Domain list
example.com
www.test.org
*.ads.net
+.block.com
# Comment
`)

	p := NewRawParser()
	entries, err := p.Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(entries))
	}

	expected := []string{"example.com", "test.org", "ads.net", "block.com"}
	for i, e := range entries {
		if e.Domain != expected[i] {
			t.Errorf("entry %d: expected %s, got %s", i, expected[i], e.Domain)
		}
		t.Logf("[%d] %s (type=%s)", i, e.Domain, e.Type)
	}
}

func TestRawParser_CanParse(t *testing.T) {
	p := NewRawParser()

	if !p.CanParse([]byte("example.com\nwww.test.org\n")) {
		t.Error("should detect raw format")
	}

	if p.CanParse([]byte("0.0.0.0 example.com\n")) {
		t.Error("should not detect hosts format as raw")
	}
}

func TestAutoParser(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "hosts",
			content:  "0.0.0.0 example.com\n127.0.0.1 test.org\n",
			expected: "hosts",
		},
		{
			name:     "raw",
			content:  "example.com\ntest.org\nads.net\n",
			expected: "raw",
		},
	}

	ap := NewAutoParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !ap.CanParse([]byte(tt.content)) {
				t.Errorf("should detect %s format", tt.name)
			}

			entries, err := ap.Parse([]byte(tt.content))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(entries) == 0 {
				t.Error("expected entries")
			}

			t.Logf("AutoParser detected: %d entries", len(entries))
		})
	}
}

func TestParser_Errors(t *testing.T) {
	ap := NewAutoParser()
	
	_, err := ap.Parse([]byte(""))
	if err == nil {
		t.Error("expected error for empty content")
	}
	
	t.Logf("Empty content error: %v", err)
}
