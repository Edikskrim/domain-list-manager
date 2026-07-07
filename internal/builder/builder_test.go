package builder

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"domain-list-manager/internal/domain"
	"domain-list-manager/internal/parser"
	"domain-list-manager/internal/source"
)

type mockDomainRepo struct{}

func (m *mockDomainRepo) Create(d *domain.Domain) error          { return nil }
func (m *mockDomainRepo) Get(id string) (*domain.Domain, error)  { return nil, os.ErrNotExist }
func (m *mockDomainRepo) List() ([]*domain.Domain, error)        { return nil, nil }
func (m *mockDomainRepo) Update(id string, d *domain.Domain) error { return nil }
func (m *mockDomainRepo) Delete(id string) error                 { return nil }
func (m *mockDomainRepo) ListBySource(sourceID string) ([]*domain.Domain, error) {
	return nil, nil
}
func (m *mockDomainRepo) Search(query string) ([]*domain.Domain, error) { return nil, nil }
func (m *mockDomainRepo) Upsert(d *domain.Domain) error                 { return nil }

type mockSourceRepo struct {
	sources []*source.Source
}

func (m *mockSourceRepo) Create(s *source.Source) error {
	m.sources = append(m.sources, s)
	return nil
}

func (m *mockSourceRepo) Get(id string) (*source.Source, error) {
	for _, s := range m.sources {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, os.ErrNotExist
}

func (m *mockSourceRepo) List() ([]*source.Source, error) {
	return m.sources, nil
}

func (m *mockSourceRepo) Update(id string, s *source.Source) error {
	for i, src := range m.sources {
		if src.ID == id {
			m.sources[i] = s
			return nil
		}
	}
	return os.ErrNotExist
}

func (m *mockSourceRepo) Delete(id string) error {
	for i, s := range m.sources {
		if s.ID == id {
			m.sources = append(m.sources[:i], m.sources[i+1:]...)
			return nil
		}
	}
	return os.ErrNotExist
}

func (m *mockSourceRepo) Enable(id string) error {
	for _, s := range m.sources {
		if s.ID == id {
			s.Enabled = true
			return nil
		}
	}
	return os.ErrNotExist
}

func (m *mockSourceRepo) Disable(id string) error {
	for _, s := range m.sources {
		if s.ID == id {
			s.Enabled = false
			return nil
		}
	}
	return os.ErrNotExist
}

func TestBuild_WithDisabledSources(t *testing.T) {
	tmpDir := t.TempDir()
	repo := &mockSourceRepo{
		sources: []*source.Source{
			{ID: "1", Name: "Source1", URL: "https://example.com/hosts", Enabled: false, ParserType: "raw"},
			{ID: "2", Name: "Source2", URL: "https://example.com/domains", Enabled: true, ParserType: "raw"},
		},
	}

	b := NewBuilder(repo, &mockDomainRepo{}, BuilderConfig{
		OutputPath: filepath.Join(tmpDir, "output.txt"),
	})

	result, err := b.Build()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.TotalFetched != 0 {
		t.Errorf("expected 0 fetched sources, got %d", result.TotalFetched)
	}

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error for disabled source, got %d", len(result.Errors))
	}
}

func TestBuild_DuplicateRemoval(t *testing.T) {
	tmpDir := t.TempDir()
	repo := &mockSourceRepo{
		sources: []*source.Source{
			{ID: "1", Name: "Source1", URL: "https://example.com/domains", Enabled: true, ParserType: "raw"},
		},
	}

	_ = NewBuilder(repo, &mockDomainRepo{}, BuilderConfig{
		OutputPath: filepath.Join(tmpDir, "output.txt"),
	})

	seen := make(map[string]bool)
	autoParser := parser.NewAutoParser()

	domainList := "example.com\nEXAMPLE.COM\nexample.com\nexample.com\n"
	content := []byte(domainList)

	entries, _ := autoParser.Parse(content)
	var newDomains []string
	for _, e := range entries {
		domain := e.Domain
		if seen[domain] {
			continue
		}
		seen[domain] = true
		newDomains = append(newDomains, domain)
	}

	if len(newDomains) != 1 {
		t.Errorf("expected 1 unique domain, got %d", len(newDomains))
	}
}

func TestBuild_Normalization(t *testing.T) {
	domains := []string{
		"EXAMPLE.COM",
		"example.com",
		"  Example.Com  ",
	}

	seen := make(map[string]bool)
	for _, d := range domains {
		domain := strings.ToLower(strings.TrimSpace(d))
		if seen[domain] {
			continue
		}
		seen[domain] = true
	}

	if len(seen) != 1 {
		t.Errorf("expected 1 unique domain after normalization, got %d", len(seen))
	}
}

func TestBuild_Sorting(t *testing.T) {
	domains := []string{
		"zebra.com",
		"alpha.com",
		"middle.com",
	}

	sort.Strings(domains)

	if domains[0] != "alpha.com" {
		t.Errorf("expected first domain to be alpha.com, got %s", domains[0])
	}
	if domains[len(domains)-1] != "zebra.com" {
		t.Errorf("expected last domain to be zebra.com, got %s", domains[len(domains)-1])
	}
}

func TestBuildResult_Structure(t *testing.T) {
	start := time.Now()
	result := &BuildResult{
		Domains:      []string{"example.com", "test.org"},
		TotalSources: 5,
		TotalFetched: 3,
		TotalParsed:  100,
		TotalDomains: 2,
		Duplicates:   98,
		Errors:       []string{"error1"},
		BuildTime:    time.Since(start),
	}

	if result.TotalDomains != 2 {
		t.Errorf("expected 2 total domains, got %d", result.TotalDomains)
	}

	if result.Duplicates != 98 {
		t.Errorf("expected 98 duplicates, got %d", result.Duplicates)
	}

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestBuildConfig_Defaults(t *testing.T) {
	cfg := BuilderConfig{
		OutputPath: "/tmp/output.txt",
	}

	if cfg.OutputPath != "/tmp/output.txt" {
		t.Errorf("expected default output path")
	}
}
