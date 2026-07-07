package source

import (
	"fmt"
	"net/url"
	"strings"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ValidateURL(src *Source) error {
	if src.URL == "" {
		return fmt.Errorf("source URL cannot be empty")
	}

	parsed, err := url.Parse(src.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	if strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	return nil
}

func (s *Service) ValidateParserType(parserType string) error {
	validTypes := []string{"raw", "hosts", "dnsmasq", "regex", "auto", "txt", "json", "url-filter"}
	for _, t := range validTypes {
		if t == parserType {
			return nil
		}
	}
	return fmt.Errorf("invalid parser type: %s (valid: %s)", parserType, strings.Join(validTypes, ", "))
}
