package history

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"domain-list-manager/internal/dto"
)

type HistoryService struct {
	repo       *SnapshotRepository
	maxSnapshots int
}

func NewHistoryService(repo *SnapshotRepository, maxSnapshots int) *HistoryService {
	return &HistoryService{
		repo:         repo,
		maxSnapshots: maxSnapshots,
	}
}

type BuildInfo struct {
	BuildTime   time.Time
	TotalDomains int
	TotalSources int
	TotalFetched int
	TotalParsed  int
	Duplicates   int
	Errors       string
	BuildTimeMs  int64
	Domains      []string
}

func (s *HistoryService) SaveSnapshot(info *BuildInfo) error {
	snapshot := &Snapshot{
		ID:           fmt.Sprintf("snap_%d", time.Now().UnixNano()),
		BuildTime:    info.BuildTime,
		TotalDomains: info.TotalDomains,
		TotalSources: info.TotalSources,
		TotalFetched: info.TotalFetched,
		TotalParsed:  info.TotalParsed,
		Duplicates:   info.Duplicates,
		Errors:       info.Errors,
		BuildTimeMs:  info.BuildTimeMs,
		Domains:      info.Domains,
	}

	if err := s.repo.Save(snapshot); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}

	count, err := s.repo.Count()
	if err != nil {
		return fmt.Errorf("count snapshots: %w", err)
	}

	if count > s.maxSnapshots {
		if err := s.repo.DeleteOld(s.maxSnapshots); err != nil {
			return fmt.Errorf("cleanup old snapshots: %w", err)
		}
	}

	return nil
}

func (s *HistoryService) GetSnapshot(id string) (*dto.SnapshotDetail, error) {
	snap, err := s.repo.Get(id)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	domains := []string{}
	if snap.DomainsJSON != "" {
		if err := json.Unmarshal([]byte(snap.DomainsJSON), &domains); err != nil {
			domains = []string{}
		}
	}

	return &dto.SnapshotDetail{
		ID:           snap.ID,
		BuildTime:    snap.BuildTime.Format(time.RFC3339),
		TotalDomains: snap.TotalDomains,
		TotalSources: snap.TotalSources,
		TotalFetched: snap.TotalFetched,
		TotalParsed:  snap.TotalParsed,
		Duplicates:   snap.Duplicates,
		Errors:       parseErrors(snap.Errors),
		BuildTimeMs:  snap.BuildTimeMs,
		Domains:      domains,
		CreatedAt:    snap.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *HistoryService) ListSnapshots(limit int) ([]*dto.SnapshotSummary, error) {
	snapshots, err := s.repo.List(limit)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	result := make([]*dto.SnapshotSummary, 0, len(snapshots))
	for _, snap := range snapshots {
		result = append(result, &dto.SnapshotSummary{
			ID:           snap.ID,
			BuildTime:    snap.BuildTime.Format(time.RFC3339),
			TotalDomains: snap.TotalDomains,
			TotalSources: snap.TotalSources,
			TotalFetched: snap.TotalFetched,
			CreatedAt:    snap.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (s *HistoryService) DeleteSnapshot(id string) error {
	return s.repo.Delete(id)
}

func (s *HistoryService) DiffSnapshots(id1, id2 string) (*dto.DiffResponse, error) {
	snap1, err := s.repo.Get(id1)
	if err != nil {
		return nil, fmt.Errorf("get snapshot 1: %w", err)
	}

	snap2, err := s.repo.Get(id2)
	if err != nil {
		return nil, fmt.Errorf("get snapshot 2: %w", err)
	}

	domains1 := parseDomains(snap1.DomainsJSON)
	domains2 := parseDomains(snap2.DomainsJSON)

	set1 := make(map[string]bool, len(domains1))
	for _, d := range domains1 {
		set1[d] = true
	}

	set2 := make(map[string]bool, len(domains2))
	for _, d := range domains2 {
		set2[d] = true
	}

	var added []string
	var removed []string

	for _, d := range domains2 {
		if !set1[d] {
			added = append(added, d)
		}
	}

	for _, d := range domains1 {
		if !set2[d] {
			removed = append(removed, d)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)

	return &dto.DiffResponse{
		Snapshot1: id1,
		Snapshot2: id2,
		BuildTime1: snap1.BuildTime.Format(time.RFC3339),
		BuildTime2: snap2.BuildTime.Format(time.RFC3339),
		TotalDomains1: snap1.TotalDomains,
		TotalDomains2: snap2.TotalDomains,
		Added:      added,
		Removed:    removed,
		AddedCount:  len(added),
		RemovedCount: len(removed),
	}, nil
}

func parseErrors(errorsStr string) []string {
	if errorsStr == "" {
		return nil
	}
	var errors []string
	if err := json.Unmarshal([]byte(errorsStr), &errors); err != nil {
		return []string{errorsStr}
	}
	return errors
}

func parseDomains(domainsJSON string) []string {
	if domainsJSON == "" {
		return []string{}
	}
	var domains []string
	if err := json.Unmarshal([]byte(domainsJSON), &domains); err != nil {
		return []string{}
	}
	return domains
}
