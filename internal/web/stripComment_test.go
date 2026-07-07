package web

import "testing"

func TestStripComment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"# comment", ""},
		{"  # comment", ""},
		{"104.18.35.41/32  # comment", "104.18.35.41/32"},
		{"104.18.35.41/32", "104.18.35.41/32"},
		{"", ""},
		{"  ", ""},
		{"# API и CDN IP-адреса (конкретные хосты)", ""},
		{"  # comment", ""},
		{"  \xc2\xa0# comment", ""},
		{"\xef\xbb\xbf# comment", ""},
		{"\xef\xbb\xbf  # comment", ""},
	}
	for _, tc := range tests {
		got := stripComment(tc.input)
		if got != tc.expected {
			t.Errorf("stripComment(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}
