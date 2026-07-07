package intersection

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"domain-list-manager/internal/domain"
	"domain-list-manager/internal/fetcher"
	"domain-list-manager/internal/parser"
	"domain-list-manager/internal/source"
)

type Service struct {
	sourceRepo source.Repository
	fetcher    *fetcher.Fetcher
	domainRepo DomainRepo
}

type DomainRepo interface {
	List() ([]*domain.Domain, error)
	ListBySource(sourceID string) ([]*domain.Domain, error)
	FindByName(name string) ([]*domain.Domain, error)
}

type SourceRepo interface {
	Get(id string) (*source.Source, error)
}

func NewService(sourceRepo source.Repository, fetcher *fetcher.Fetcher, domainRepo DomainRepo) *Service {
	return &Service{
		sourceRepo: sourceRepo,
		fetcher:    fetcher,
		domainRepo: domainRepo,
	}
}

type DomainSet struct {
	Domains map[string]bool
}

func NewDomainSet(domains []string) *DomainSet {
	ds := make(map[string]bool)
	for _, d := range domains {
		ds[strings.ToLower(strings.TrimSpace(d))] = true
	}
	return &DomainSet{Domains: ds}
}

func (ds *DomainSet) Contains(domain string) bool {
	return ds.Domains[strings.ToLower(strings.TrimSpace(domain))]
}

func (ds *DomainSet) ToSlice() []string {
	result := make([]string, 0, len(ds.Domains))
	for d := range ds.Domains {
		result = append(result, d)
	}
	sort.Strings(result)
	return result
}

type IntersectionReport struct {
	IntersectingDomains []IntersectingDomain `json:"intersecting_domains"`
	UniqueDomains       []UniqueDomain       `json:"unique_domains"`
	Summary             ReportSummary        `json:"summary"`
	SourceDomains       []SourceDomainInfo   `json:"source_domains"`
	AnalyzedAt          string               `json:"analyzed_at"`
}

type IntersectingDomain struct {
	Domain      string   `json:"domain"`
	SourceCount int      `json:"source_count"`
	Sources     []string `json:"sources"`
}

type UniqueDomain struct {
	Domain string `json:"domain"`
	Source string `json:"source"`
}

type ReportSummary struct {
	TotalSources     int `json:"total_sources"`
	TotalDomains     int `json:"total_domains"`
	IntersectingCount int `json:"intersecting_count"`
	UniqueCount      int `json:"unique_count"`
}

type SourceDomainInfo struct {
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
	DomainCount int  `json:"domain_count"`
}

type IntersectionResult struct {
	Report *IntersectionReport
	DomainsPerSource map[string]*DomainSet
}

func (s *Service) GetDomainsByName(name string) ([]*domain.Domain, error) {
	return s.domainRepo.FindByName(name)
}

func (s *Service) Analyze() (*IntersectionResult, error) {
	domains, err := s.domainRepo.List()
	if err == nil && len(domains) > 0 {
		report := s.buildReportFromDB()
		return &IntersectionResult{
			Report:           report,
			DomainsPerSource: make(map[string]*DomainSet),
		}, nil
	}

	domainsPerSource := make(map[string]*DomainSet)
	sourceNameMap := make(map[string]string)

	sources, err := s.sourceRepo.List()
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}

	for _, src := range sources {
		if !src.Enabled {
			continue
		}

		sourceNameMap[src.ID] = src.Name
		domains, err := s.fetchSourceDomains(src)
		if err != nil {
			continue
		}
		if len(domains) == 0 {
			continue
		}

		domainsPerSource[src.ID] = NewDomainSet(domains)
	}

	report := s.buildReport(domainsPerSource, sourceNameMap)

	return &IntersectionResult{
		Report:           report,
		DomainsPerSource: domainsPerSource,
	}, nil
}

func (s *Service) loadFromDatabase() (map[string]*DomainSet, map[string]string) {
	domainsPerSource := make(map[string]*DomainSet)
	sourceNameMap := make(map[string]string)

	domains, err := s.domainRepo.List()
	if err != nil {
		return domainsPerSource, sourceNameMap
	}

	sourceDomainMap := make(map[string][]string)
	for _, d := range domains {
		if d.SourceID != nil {
			sourceDomainMap[*d.SourceID] = append(sourceDomainMap[*d.SourceID], d.Name)
		}
	}

	for sourceID, sourceDomains := range sourceDomainMap {
		if len(sourceDomains) > 0 {
			sourceNameMap[sourceID] = sourceID
			domainsPerSource[sourceID] = NewDomainSet(sourceDomains)
		}
	}

	return domainsPerSource, sourceNameMap
}

func (s *Service) buildReportFromDB() *IntersectionReport {
	domains, err := s.domainRepo.List()
	if err != nil {
		return &IntersectionReport{
			AnalyzedAt: time.Now().UTC().Format(time.RFC3339),
		}
	}

	domainName := make(map[string]string)
	domainCount := make(map[string]int)
	domainSourceIDs := make(map[string]map[string]bool)
	sourceDomainCount := make(map[string]int)

	for _, d := range domains {
		name := strings.ToLower(strings.TrimSpace(d.Name))
		if name == "" {
			continue
		}
		domainName[name] = name
		domainCount[name]++
		if domainSourceIDs[name] == nil {
			domainSourceIDs[name] = make(map[string]bool)
		}
		if d.SourceID != nil {
			domainSourceIDs[name][*d.SourceID] = true
			sourceDomainCount[*d.SourceID]++
		} else {
			domainSourceIDs[name][""] = true
			sourceDomainCount[""]++
		}
	}

	var intersectingDomains []IntersectingDomain
	var uniqueDomains []UniqueDomain
	allDomains := make(map[string]bool)

	sourceNameCache := make(map[string]string)
	for sourceID := range sourceDomainCount {
		if sourceID == "" {
			sourceNameCache[""] = "Не указан"
			continue
		}
		if _, ok := sourceNameCache[sourceID]; ok {
			continue
		}
		if s.sourceRepo != nil {
			src, err := s.sourceRepo.Get(sourceID)
			if err == nil && src != nil {
				sourceNameCache[sourceID] = src.Name
			} else {
				sourceNameCache[sourceID] = sourceID
			}
		} else {
			sourceNameCache[sourceID] = sourceID
		}
	}

	for domain, count := range domainCount {
		allDomains[domain] = true
		if count >= 2 {
			sourceIDs := make([]string, 0, len(domainSourceIDs[domain]))
			for srcID := range domainSourceIDs[domain] {
				sourceIDs = append(sourceIDs, srcID)
			}
			sort.Strings(sourceIDs)
			sourceNames := make([]string, 0, len(sourceIDs))
			for _, srcID := range sourceIDs {
				sourceNames = append(sourceNames, sourceNameCache[srcID])
			}
			intersectingDomains = append(intersectingDomains, IntersectingDomain{
				Domain:      domain,
				SourceCount: count,
				Sources:     sourceNames,
			})
		} else {
			var srcName string
			if d, ok := domainSourceIDs[domain]; ok {
				for srcID := range d {
					srcName = sourceNameCache[srcID]
					break
				}
			}
			uniqueDomains = append(uniqueDomains, UniqueDomain{
				Domain: domain,
				Source: srcName,
			})
		}
	}

	sort.Slice(intersectingDomains, func(i, j int) bool {
		if intersectingDomains[i].SourceCount != intersectingDomains[j].SourceCount {
			return intersectingDomains[i].SourceCount > intersectingDomains[j].SourceCount
		}
		return intersectingDomains[i].Domain < intersectingDomains[j].Domain
	})

	sort.Slice(uniqueDomains, func(i, j int) bool {
		return uniqueDomains[i].Domain < uniqueDomains[j].Domain
	})

	var sourceDomainInfo []SourceDomainInfo
	for sourceID, count := range sourceDomainCount {
		sourceDomainInfo = append(sourceDomainInfo, SourceDomainInfo{
			SourceID:    sourceID,
			SourceName:  sourceNameCache[sourceID],
			DomainCount: count,
		})
	}
	sort.Slice(sourceDomainInfo, func(i, j int) bool {
		return sourceDomainInfo[i].DomainCount > sourceDomainInfo[j].DomainCount
	})

	return &IntersectionReport{
		IntersectingDomains: intersectingDomains,
		UniqueDomains:       uniqueDomains,
		Summary: ReportSummary{
			TotalSources:      len(sourceDomainCount),
			TotalDomains:      len(allDomains),
			IntersectingCount: len(intersectingDomains),
			UniqueCount:       len(uniqueDomains),
		},
		SourceDomains: sourceDomainInfo,
		AnalyzedAt:    time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *Service) fetchSourceDomains(src *source.Source) ([]string, error) {
	resp, err := s.fetcher.Fetch(src.URL, "", "")
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", src.Name, err)
	}
	if resp == nil || !resp.IsContentModified() {
		return nil, nil
	}

	autoParser := parser.NewAutoParser()
	entries, err := autoParser.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", src.Name, err)
	}

	var domains []string
	for _, e := range entries {
		domain := strings.ToLower(strings.TrimSpace(e.Domain))
		if domain != "" {
			domains = append(domains, domain)
		}
	}

	return domains, nil
}

func (s *Service) buildReport(domainsPerSource map[string]*DomainSet, sourceNameMap map[string]string) *IntersectionReport {
	domainSources := make(map[string][]string)
	allDomains := make(map[string]bool)

	for sourceID, ds := range domainsPerSource {
		for domain := range ds.Domains {
			allDomains[domain] = true
			domainSources[domain] = append(domainSources[domain], sourceID)
		}
	}

	var intersectingDomains []IntersectingDomain
	var uniqueDomains []UniqueDomain

	for domain, sources := range domainSources {
		if len(sources) >= 2 {
			sort.Strings(sources)
			intersectingDomains = append(intersectingDomains, IntersectingDomain{
				Domain:      domain,
				SourceCount: len(sources),
				Sources:     sources,
			})
		} else {
			uniqueDomains = append(uniqueDomains, UniqueDomain{
				Domain: domain,
				Source: sourceNameMap[sources[0]],
			})
		}
	}

	sort.Slice(intersectingDomains, func(i, j int) bool {
		if intersectingDomains[i].SourceCount != intersectingDomains[j].SourceCount {
			return intersectingDomains[i].SourceCount > intersectingDomains[j].SourceCount
		}
		return intersectingDomains[i].Domain < intersectingDomains[j].Domain
	})

	sort.Slice(uniqueDomains, func(i, j int) bool {
		return uniqueDomains[i].Domain < uniqueDomains[j].Domain
	})

	var sourceDomainInfo []SourceDomainInfo
	for sourceID, ds := range domainsPerSource {
		sourceDomainInfo = append(sourceDomainInfo, SourceDomainInfo{
			SourceID:    sourceID,
			SourceName:  sourceNameMap[sourceID],
			DomainCount: len(ds.Domains),
		})
	}

	sort.Slice(sourceDomainInfo, func(i, j int) bool {
		return sourceDomainInfo[i].DomainCount > sourceDomainInfo[j].DomainCount
	})

	return &IntersectionReport{
		IntersectingDomains: intersectingDomains,
		UniqueDomains:       uniqueDomains,
		Summary: ReportSummary{
			TotalSources:      len(domainsPerSource),
			TotalDomains:      len(allDomains),
			IntersectingCount: len(intersectingDomains),
			UniqueCount:       len(uniqueDomains),
		},
		SourceDomains: sourceDomainInfo,
		AnalyzedAt:    time.Now().UTC().Format(time.RFC3339),
	}
}
