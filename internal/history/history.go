package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Snapshot struct {
	ID           string    `json:"id"`
	BuildTime    time.Time `json:"build_time"`
	TotalDomains int       `json:"total_domains"`
	TotalSources int       `json:"total_sources"`
	TotalFetched int       `json:"total_fetched"`
	TotalParsed  int       `json:"total_parsed"`
	Duplicates   int       `json:"duplicates"`
	Errors       string    `json:"errors"`
	BuildTimeMs  int64     `json:"build_time_ms"`
	DomainsJSON  string    `json:"domains_json"`
	CreatedAt    time.Time `json:"created_at"`
	Domains      []string  `json:"-"`
}

type SnapshotRepository struct {
	db *sql.DB
}

func NewSnapshotRepository(db *sql.DB) *SnapshotRepository {
	return &SnapshotRepository{db: db}
}

func (r *SnapshotRepository) Save(snapshot *Snapshot) error {
	domainsJSON, err := json.Marshal(snapshot.Domains)
	if err != nil {
		return err
	}
	snapshot.DomainsJSON = string(domainsJSON)
	_, err = r.db.Exec(
		`INSERT INTO snapshots (id, build_time, total_domains, total_sources, total_fetched, total_parsed, duplicates, errors, build_time_ms, domains_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.ID,
		snapshot.BuildTime.UTC(),
		snapshot.TotalDomains,
		snapshot.TotalSources,
		snapshot.TotalFetched,
		snapshot.TotalParsed,
		snapshot.Duplicates,
		snapshot.Errors,
		snapshot.BuildTimeMs,
		snapshot.DomainsJSON,
		time.Now().UTC(),
	)
	return err
}

func (r *SnapshotRepository) Get(id string) (*Snapshot, error) {
	snapshot := &Snapshot{}
	err := r.db.QueryRow(
		`SELECT id, build_time, total_domains, total_sources, total_fetched, total_parsed, duplicates, errors, build_time_ms, domains_json, created_at
		 FROM snapshots WHERE id = ?`,
		id,
	).Scan(
		&snapshot.ID,
		&snapshot.BuildTime,
		&snapshot.TotalDomains,
		&snapshot.TotalSources,
		&snapshot.TotalFetched,
		&snapshot.TotalParsed,
		&snapshot.Duplicates,
		&snapshot.Errors,
		&snapshot.BuildTimeMs,
		&snapshot.DomainsJSON,
		&snapshot.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if snapshot.DomainsJSON != "" {
		if err := json.Unmarshal([]byte(snapshot.DomainsJSON), &snapshot.Domains); err != nil {
			return nil, fmt.Errorf("unmarshal domains: %w", err)
		}
	}
	return snapshot, nil
}

func (r *SnapshotRepository) List(limit int) ([]*Snapshot, error) {
	rows, err := r.db.Query(
		`SELECT id, build_time, total_domains, total_sources, total_fetched, total_parsed, duplicates, errors, build_time_ms, domains_json, created_at
		 FROM snapshots ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []*Snapshot
	for rows.Next() {
		snapshot := &Snapshot{}
		err := rows.Scan(
			&snapshot.ID,
			&snapshot.BuildTime,
			&snapshot.TotalDomains,
			&snapshot.TotalSources,
			&snapshot.TotalFetched,
			&snapshot.TotalParsed,
			&snapshot.Duplicates,
			&snapshot.Errors,
			&snapshot.BuildTimeMs,
			&snapshot.DomainsJSON,
			&snapshot.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (r *SnapshotRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM snapshots WHERE id = ?`, id)
	return err
}

func (r *SnapshotRepository) DeleteOld(maxCount int) error {
	_, err := r.db.Exec(
		`DELETE FROM snapshots WHERE id IN (
			SELECT id FROM snapshots ORDER BY created_at DESC LIMIT -1 OFFSET ?
		)`,
		maxCount,
	)
	return err
}

func (r *SnapshotRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM snapshots`).Scan(&count)
	return count, err
}
