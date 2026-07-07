package scheduler

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"

	"domain-list-manager/internal/domain"
	"domain-list-manager/internal/fetcher"
	"domain-list-manager/internal/source"
	"domain-list-manager/internal/update_metadata"
	"domain-list-manager/internal/parser"
)

type Scheduler struct {
	sourceRepo   source.Repository
	metadataRepo update_metadata.Repository
	fetcher      *fetcher.Fetcher
	domainRepo   domain.Repository
	ticker       *time.Ticker
	interval     time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.Mutex
	running      bool
	lastUpdate   time.Time
	nextUpdate   time.Time
	updateCount  int
	sourceCount  int
	errorCount   int
}

type UpdateResult struct {
	SourceID   string    `json:"source_id"`
	SourceName string    `json:"source_name"`
	Updated    bool      `json:"updated"`
	Skipped    bool      `json:"skipped"`
	Reason     string    `json:"reason,omitempty"`
	Error      string    `json:"error,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewScheduler(
	sourceRepo source.Repository,
	metadataRepo update_metadata.Repository,
	fetcher *fetcher.Fetcher,
	domainRepo domain.Repository,
	interval time.Duration,
) *Scheduler {
	return &Scheduler{
		sourceRepo:   sourceRepo,
		metadataRepo: metadataRepo,
		fetcher:      fetcher,
		domainRepo:   domainRepo,
		interval:     interval,
	}
}

func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.ticker = time.NewTicker(s.interval)
	s.running = true

	go s.run()
	return nil
}

func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	if s.cancel != nil {
		s.cancel()
	}

	if s.ticker != nil {
		s.ticker.Stop()
	}

	s.running = false
	return nil
}

func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Scheduler) TriggerUpdate() ([]*UpdateResult, error) {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	if !running {
		return nil, fmt.Errorf("scheduler is not running")
	}

	return s.updateAll(), nil
}

func (s *Scheduler) updateAll() []*UpdateResult {
	sources, err := s.sourceRepo.List()
	if err != nil {
		return []*UpdateResult{{
			SourceName: "all",
			Error:      fmt.Sprintf("list sources: %v", err),
		}}
	}

	s.mu.Lock()
	s.sourceCount = len(sources)
	s.mu.Unlock()

	results := make([]*UpdateResult, 0, len(sources))
	autoParser := parser.NewAutoParser()

	for _, src := range sources {
		if !src.Enabled {
			continue
		}

		result := s.updateSource(src, autoParser)
		results = append(results, result)
	}

	return results
}

func (s *Scheduler) updateSource(src *source.Source, autoParser *parser.AutoParser) *UpdateResult {
	now := time.Now().UTC()

	result := &UpdateResult{
		SourceID:   src.ID,
		SourceName: src.Name,
		UpdatedAt:  now,
	}

	if src.LastUpdate.IsZero() {
		result.Skipped = true
		result.Reason = "never updated"
		return result
	}

	elapsed := now.Sub(src.LastUpdate)
	if elapsed < time.Duration(src.UpdateInterval)*time.Second {
		result.Skipped = true
		result.Reason = fmt.Sprintf("interval not elapsed (%.0fs / %.0fs)", elapsed.Seconds(), float64(src.UpdateInterval))
		return result
	}

	resp, err := s.fetcher.Fetch(src.URL, "", "")
	if err != nil {
		result.Error = fmt.Sprintf("fetch: %v", err)
		s.mu.Lock()
		s.errorCount++
		s.mu.Unlock()
		return result
	}

	if !resp.IsContentModified() {
		result.Skipped = true
		result.Reason = "content not modified (304)"
		return result
	}

	_ = computeHash(resp.Body)

	metadata, err := s.metadataRepo.Get(src.ID)
	if err != nil {
		metadata = &update_metadata.UpdateMetadata{
			SourceID:  src.ID,
			LastFetched: now,
		}
		if err := s.metadataRepo.Create(metadata); err != nil {
			result.Error = fmt.Sprintf("create metadata: %v", err)
			return result
		}
	} else {
		metadata.LastFetched = now
		if err := s.metadataRepo.Update(src.ID, metadata); err != nil {
			result.Error = fmt.Sprintf("update metadata: %v", err)
			return result
		}
	}

	entries, err := autoParser.Parse(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("parse: %v", err)
		s.mu.Lock()
		s.errorCount++
		s.mu.Unlock()
		return result
	}

	fmt.Printf("[scheduler] source=%s detected=%s entries=%d body_size=%d\n", src.Name, autoParser.DetectType(resp.Body), len(entries), len(resp.Body))

	saved := 0
	missingDomain := 0
	for _, e := range entries {
		domainName := strings.ToLower(e.Domain)
		if domainName == "" {
			missingDomain++
			if missingDomain <= 3 {
				fmt.Printf("[scheduler] empty domain entry: %q\n", e.Domain)
			}
			continue
		}

		id := fmt.Sprintf("%s:%s", src.ID, domainName)
		d := &domain.Domain{
			ID:       id,
			Name:     domainName,
			SourceID: &src.ID,
		}

		if err := s.domainRepo.Upsert(d); err != nil {
			fmt.Printf("[scheduler] upsert error: %v for %s\n", err, domainName)
			continue
		}
		saved++
	}

	fmt.Printf("[scheduler] source=%s saved=%d missingDomain=%d\n", src.Name, saved, missingDomain)

	src.LastUpdate = now
	src.LastError = ""
	src.DomainCount = len(entries)
	if err := s.sourceRepo.Update(src.ID, src); err != nil {
		result.Error = fmt.Sprintf("update source: %v", err)
		return result
	}

	result.Updated = true
	result.Reason = fmt.Sprintf("parsed %d entries, saved %d domains", len(entries), saved)
	s.mu.Lock()
	s.updateCount++
	s.mu.Unlock()

	return result
}

func computeHash(content []byte) string {
	h := fnv.New32a()
	h.Write(content)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *Scheduler) run() {
	now := time.Now()
	s.mu.Lock()
	s.lastUpdate = now
	s.nextUpdate = now.Add(s.interval)
	s.mu.Unlock()

	for {
		select {
		case <-s.ticker.C:
			s.updateAll()
			s.mu.Lock()
			s.lastUpdate = time.Now()
			s.nextUpdate = s.lastUpdate.Add(s.interval)
			s.mu.Unlock()

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Scheduler) GetStatus() SchedulerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

		lastUpdate := s.lastUpdate
		nextUpdate := s.nextUpdate
		return SchedulerStatus{
			Running:     s.running,
			LastUpdate:  &lastUpdate,
			NextUpdate:  &nextUpdate,
			UpdateTime:  s.interval,
			UpdateCount: s.updateCount,
			SourceCount: s.sourceCount,
			ErrorCount:  s.errorCount,
		}
}

type SchedulerStatus struct {
	Running     bool       `json:"running"`
	LastUpdate  *time.Time `json:"last_update"`
	NextUpdate  *time.Time `json:"next_update"`
	UpdateTime  time.Duration `json:"update_interval_ms"`
	UpdateCount int        `json:"update_count"`
	SourceCount int        `json:"source_count"`
	ErrorCount  int        `json:"error_count"`
}
