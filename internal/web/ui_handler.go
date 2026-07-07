package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"domain-list-manager/internal/source"
)

// Domains handler.
func (s *UIService) Domains(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	sourceFilter := r.URL.Query().Get("source")
	sort := r.URL.Query().Get("sort")
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "asc"
	}
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	pageSize := 50
	offset := (page - 1) * pageSize

	log.Printf("Domains page: page=%d pageSize=%d sort=%q dir=%q search=%q", page, pageSize, sort, dir, search)

	pageData := map[string]interface{}{
		"Title":           "Домены",
		"Current":         "domains",
		"Domains":         []map[string]interface{}{},
		"Search":          search,
		"Sort":            sort,
		"Dir":             dir,
		"SelectedSource":  sourceFilter,
		"Page":            page,
		"PageSize":        pageSize,
	}

	if s.snapshotDB != nil {
		// Test database connection
		var dbCount int
		if err := s.snapshotDB.QueryRow("SELECT COUNT(*) FROM domains").Scan(&dbCount); err == nil {
			log.Printf("Domains DB total count: %d", dbCount)
		}
		var paginationLinks []map[string]interface{}
		countQuery := "SELECT COUNT(*) FROM domains"
		countParams := []interface{}{}
		countWhereClauses := []string{}
		if search != "" {
			countWhereClauses = append(countWhereClauses, "name LIKE ?")
			countParams = append(countParams, "%"+search+"%")
		}
		if sourceFilter != "" {
			countWhereClauses = append(countWhereClauses, "source_id = ?")
			countParams = append(countParams, sourceFilter)
		}
		if len(countWhereClauses) > 0 {
			countQuery += " WHERE " + strings.Join(countWhereClauses, " AND ")
		}
		var totalCount int
		if err := s.snapshotDB.QueryRow(countQuery, countParams...).Scan(&totalCount); err == nil {
			pageData["TotalCount"] = totalCount
			if totalCount > 0 {
				totalPages := (totalCount + pageSize - 1) / pageSize
				pageData["TotalPages"] = totalPages
				baseURL := "?search=" + search + "&source=" + sourceFilter + "&sort=" + sort + "&dir=" + dir
				pageNums := []int{}
				start := page - 2
				if start < 1 {
					start = 1
				}
				end := page + 2
				if end > totalPages {
					end = totalPages
				}
				if start > 1 {
					pageNums = append(pageNums, 1)
					if start > 2 {
						pageNums = append(pageNums, -1)
					}
				}
				for i := start; i <= end; i++ {
					if i > 0 && i <= totalPages {
						pageNums = append(pageNums, i)
					}
				}
				if end < totalPages {
					if end < totalPages-1 {
						pageNums = append(pageNums, -2)
					}
					pageNums = append(pageNums, totalPages)
				}
				for _, p := range pageNums {
					if p == -1 {
						paginationLinks = append(paginationLinks, map[string]interface{}{"Type": "ellipsis"})
					} else if p == -2 {
						paginationLinks = append(paginationLinks, map[string]interface{}{"Type": "ellipsis"})
					} else {
						active := false
						if p == page {
							active = true
						}
						paginationLinks = append(paginationLinks, map[string]interface{}{"Type": "link", "Href": baseURL + "&page=" + strconv.Itoa(p), "Label": strconv.Itoa(p), "Active": active})
					}
				}
				if page > 1 {
					paginationLinks = append([]map[string]interface{}{{"Type": "link", "Href": baseURL + "&page=" + strconv.Itoa(page-1), "Label": "\u25C0"}}, paginationLinks...)
				}
				if page < totalPages {
					paginationLinks = append(paginationLinks, map[string]interface{}{"Type": "link", "Href": baseURL + "&page=" + strconv.Itoa(page+1), "Label": "\u25B6"})
				}
			}
		}

		type SrcOpt struct {
			ID   string
			Name string
		}
		var srcOpts []SrcOpt
		srcRows, err := s.snapshotDB.Query("SELECT DISTINCT source_id FROM domains WHERE source_id IS NOT NULL AND source_id != ''")
		if err == nil {
			for srcRows.Next() {
				var srcID string
				if err := srcRows.Scan(&srcID); err == nil {
					if srcID == "manual" {
						srcOpts = append(srcOpts, SrcOpt{ID: srcID, Name: "Ручное добавление"})
					} else {
						srcOpts = append(srcOpts, SrcOpt{ID: srcID, Name: srcID})
					}
				}
			}
			srcRows.Close()
		}
		// Add manual source if not already present
		hasManual := false
		for _, opt := range srcOpts {
			if opt.ID == "manual" {
				hasManual = true
				break
			}
		}
		if !hasManual {
			srcOpts = append([]SrcOpt{{ID: "manual", Name: "Ручное добавление"}}, srcOpts...)
		}
		if len(srcOpts) > 0 {
			srcNames := make([]string, len(srcOpts))
			for i, o := range srcOpts {
				srcNames[i] = o.ID
			}
			if srcRows2, err := s.snapshotDB.Query("SELECT id, name FROM sources WHERE id IN ("+placeholders(len(srcNames))+")", toStringAny(srcNames)...); err == nil {
				for srcRows2.Next() {
					var id, name string
					if err := srcRows2.Scan(&id, &name); err == nil {
						for i := range srcOpts {
							if srcOpts[i].ID == id {
								srcOpts[i].Name = name
							}
						}
					}
				}
				srcRows2.Close()
			}
		pageData["SourceOptions"] = srcOpts
	}

	query := "SELECT d.id, COALESCE(s.name, d.source_id), d.name, d.created_at FROM domains d LEFT JOIN sources s ON d.source_id = s.id"
		params := []interface{}{}
		whereClauses := []string{}
		if search != "" {
			whereClauses = append(whereClauses, "d.name LIKE ?")
			params = append(params, "%"+search+"%")
		}
		if sourceFilter != "" {
			whereClauses = append(whereClauses, "d.source_id = ?")
			params = append(params, sourceFilter)
		}
		if len(whereClauses) > 0 {
			query += " WHERE " + strings.Join(whereClauses, " AND ")
		}
		query += " ORDER BY d.name " + dir + " LIMIT ? OFFSET ?"
		params = append(params, pageSize, offset)
		log.Printf("Domains SQL: %s params: %v", query, params)
		rows, err := s.snapshotDB.Query(query, params...)
		if err == nil {
			defer rows.Close()
			domainCount := 0
			for rows.Next() {
				var id string
				var sourceName interface{}
				var name string
				var createdAt string
				if err := rows.Scan(&id, &sourceName, &name, &createdAt); err != nil {
					log.Printf("Domains row scan error: %v", err)
					continue
				}
				if sourceName == nil {
					sourceName = ""
				}
				pageData["Domains"] = append(pageData["Domains"].([]map[string]interface{}), map[string]interface{}{
					"ID":        id,
					"Name":      name,
					"Source":    sourceName.(string),
					"CreatedAt": createdAt,
				})
				domainCount++
			}
			if err := rows.Err(); err != nil {
				log.Printf("Domains rows.Err() error: %v", err)
			}
			log.Printf("Domains query returned %d domains", domainCount)
		} else {
			log.Printf("Domains query error: %v", err)
		}
		if totalCount > 0 {
			pageData["PaginationLinks"] = paginationLinks
		}
	}

	s.render(w, r, "Домены", "domains.html", pageData)
}

// Sources handler.
func (s *UIService) Sources(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	statusFilter := r.URL.Query().Get("status")

	pageData := map[string]interface{}{
		"Title":    "Источники",
		"Current":  "sources",
		"Sources":  []source.Source{},
		"Search":   search,
		"Status":   statusFilter,
		"Enabled":  0,
		"Disabled": 0,
	}

	client := &http.Client{}
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}
	req, err := http.NewRequest(http.MethodGet, "http://"+host+"/api/v1/sources", nil)
	if err != nil {
		s.render(w, r, "Источники", "sources.html", pageData)
		return
	}
	for _, c := range r.Cookies() {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		s.render(w, r, "Источники", "sources.html", pageData)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.render(w, r, "Источники", "sources.html", pageData)
		return
	}

	if resp.StatusCode != http.StatusOK {
		s.render(w, r, "Источники", "sources.html", pageData)
		return
	}

	var wrapper struct {
		Success bool              `json:"success"`
		Data    json.RawMessage   `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil && len(wrapper.Data) > 0 {
		var srcs []source.Source
		if err := json.Unmarshal(wrapper.Data, &srcs); err == nil {
			pageData["Sources"] = srcs

			enabled := 0
			disabled := 0
			for _, src := range srcs {
				if src.Enabled {
					enabled++
				} else {
					disabled++
				}
			}
			pageData["Enabled"] = enabled
			pageData["Disabled"] = disabled
		}
	} else {
		var srcs []source.Source
		if err := json.Unmarshal(body, &srcs); err == nil {
			pageData["Sources"] = srcs

			enabled := 0
			disabled := 0
			for _, src := range srcs {
				if src.Enabled {
					enabled++
				} else {
					disabled++
				}
			}
			pageData["Enabled"] = enabled
			pageData["Disabled"] = disabled
		}
	}

	s.render(w, r, "Источники", "sources.html", pageData)
}

// History handler.
func (s *UIService) History(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "desc"
	}

	pageData := map[string]interface{}{
		"Title":     "История",
		"Current":   "history",
		"Snapshots": []map[string]interface{}{},
		"Sort":      sort,
		"Dir":       dir,
		"PageSize":  50,
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
		if limit <= 0 {
			limit = 100
		}
	}

	if s.snapshotDB != nil {
		rows, err := s.snapshotDB.Query(
			`SELECT id, build_time, total_domains, total_sources, total_fetched, total_parsed, duplicates, errors, build_time_ms, domains_json, created_at
			 FROM snapshots ORDER BY created_at DESC LIMIT ?`,
			limit,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var snap = make(map[string]interface{})
				var id string
				var buildTime, createdAt string
				var totalDomains64, totalSources64, totalFetched64, totalParsed64, duplicates64 int64
				var errorsStr string
				var buildTimeMs int64
				var domainsJSON string
				err := rows.Scan(&id, &buildTime, &totalDomains64, &totalSources64, &totalFetched64, &totalParsed64, &duplicates64, &errorsStr, &buildTimeMs, &domainsJSON, &createdAt)
				if err == nil {
					snap["ID"] = id
					snap["BuildTime"] = buildTime
					snap["TotalDomains"] = int(totalDomains64)
					snap["TotalSources"] = int(totalSources64)
					snap["TotalFetched"] = int(totalFetched64)
					snap["TotalParsed"] = int(totalParsed64)
					snap["Duplicates"] = int(duplicates64)
					snap["Errors"] = errorsStr
					snap["BuildTimeMs"] = buildTimeMs
					snap["CreatedAt"] = createdAt
					pageData["Snapshots"] = append(pageData["Snapshots"].([]map[string]interface{}), snap)
				}
			}
		}
	}

	s.render(w, r, "История", "history.html", pageData)
}

// HistoryDetail handles /ui/history/{id} for snapshot detail or diff.
func (s *UIService) HistoryDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/ui/history/"):]
	if id == "" || id == "/" {
		s.History(w, r)
		return
	}

	if r.URL.Query().Get("diff") != "" || (r.URL.Query().Get("s1") != "" && r.URL.Query().Get("s2") != "") {
		s.HistoryDiff(w, r)
		return
	}

	pageData := map[string]interface{}{
		"Title":    "Детали снимка",
		"Current":  "history",
		"Snapshot": map[string]interface{}{"BuildTimeMs": int64(0), "ID": "", "BuildTime": "", "TotalDomains": 0, "TotalSources": 0, "TotalFetched": 0, "TotalParsed": 0, "Duplicates": 0, "Errors": []string{}, "Domains": []string{}, "CreatedAt": ""},
	}

	client := &http.Client{}
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	req, err := http.NewRequest(http.MethodGet, scheme+"://"+host+"/api/v1/history/"+id, nil)
	if err != nil {
		s.render(w, r, "Детали снимка", "history_snapshot.html", pageData)
		return
	}
	for _, c := range r.Cookies() {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		pageData["Error"] = "Не удалось получить данные: " + err.Error()
		s.render(w, r, pageData["Title"].(string), "history_snapshot.html", pageData)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		pageData["Error"] = "API error (status " + fmt.Sprintf("%d", resp.StatusCode) + "): " + string(body)
		s.render(w, r, pageData["Title"].(string), "history_snapshot.html", pageData)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		pageData["Error"] = "Не удалось прочитать ответ: " + err.Error()
		s.render(w, r, pageData["Title"].(string), "history_snapshot.html", pageData)
		return
	}

	var wrapper struct {
		Success bool              `json:"success"`
		Data    json.RawMessage   `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil && wrapper.Data != nil && len(wrapper.Data) > 0 {
		var snap map[string]interface{}
		if err := json.Unmarshal(wrapper.Data, &snap); err == nil {
			pascal := map[string]interface{}{}
			for k, v := range snap {
				switch k {
				case "id":
					pascal["ID"] = v
				case "build_time":
					pascal["BuildTime"] = v
				case "build_time_ms":
					pascal["BuildTimeMs"] = v
				case "total_domains":
					pascal["TotalDomains"] = v
				case "total_sources":
					pascal["TotalSources"] = v
				case "total_fetched":
					pascal["TotalFetched"] = v
				case "total_parsed":
					pascal["TotalParsed"] = v
				case "duplicates":
					pascal["Duplicates"] = v
				case "errors":
					pascal["Errors"] = v
				case "domains":
					pascal["Domains"] = v
				case "created_at":
					pascal["CreatedAt"] = v
				}
			}
			if p, ok := pascal["BuildTimeMs"]; !ok || p == nil {
				pascal["BuildTimeMs"] = int64(0)
			} else if f, ok := p.(float64); ok {
				pascal["BuildTimeMs"] = int64(f)
			} else if i, ok := p.(int); ok {
				pascal["BuildTimeMs"] = int64(i)
			} else if i, ok := p.(int64); ok {
				pascal["BuildTimeMs"] = i
			}
			if _, ok := pascal["TotalDomains"]; !ok {
				pascal["TotalDomains"] = 0
			}
			if _, ok := pascal["TotalSources"]; !ok {
				pascal["TotalSources"] = 0
			}
			if _, ok := pascal["TotalFetched"]; !ok {
				pascal["TotalFetched"] = 0
			}
			if _, ok := pascal["TotalParsed"]; !ok {
				pascal["TotalParsed"] = 0
			}
			if _, ok := pascal["Duplicates"]; !ok {
				pascal["Duplicates"] = 0
			}
			if _, ok := pascal["Errors"]; !ok {
				pascal["Errors"] = []string{}
			}
			if _, ok := pascal["Domains"]; !ok {
				pascal["Domains"] = []string{}
			}
			if _, ok := pascal["ID"]; !ok {
				pascal["ID"] = ""
			}
			if _, ok := pascal["BuildTime"]; !ok {
				pascal["BuildTime"] = ""
			}
			if _, ok := pascal["CreatedAt"]; !ok {
				pascal["CreatedAt"] = ""
			}
			pageData["Snapshot"] = pascal
		}
	}
	if pageData["Snapshot"] == nil || len(pageData["Snapshot"].(map[string]interface{})) == 0 {
		pageData["Error"] = "Снимок не найден или пустой"
	}

	pageData["Title"] = "Детали снимка: " + id

	s.render(w, r, pageData["Title"].(string), "history_snapshot.html", pageData)
}

// HistoryDiff handles diff viewing.
func (s *UIService) HistoryDiff(w http.ResponseWriter, r *http.Request) {
	s1 := r.URL.Query().Get("s1")
	s2 := r.URL.Query().Get("s2")

	pageData := map[string]interface{}{
		"Title":     "Сравнение снимков",
		"Current":   "history",
		"Diff":      map[string]interface{}{},
		"Snapshot1": s1,
		"Snapshot2": s2,
	}

	if s1 == "" || s2 == "" {
		if s.snapshotDB != nil {
			rows, err := s.snapshotDB.Query(
				`SELECT id, build_time, total_domains, total_sources, total_fetched, total_parsed, duplicates, errors, build_time_ms, domains_json, created_at
				 FROM snapshots ORDER BY created_at DESC LIMIT 200`,
			)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var snap = make(map[string]interface{})
					var id string
					var buildTime, createdAt string
					var totalDomains64, totalSources64, totalFetched64, totalParsed64, duplicates64 int64
					var errorsStr string
					var buildTimeMs int64
					var domainsJSON string
					err := rows.Scan(&id, &buildTime, &totalDomains64, &totalSources64, &totalFetched64, &totalParsed64, &duplicates64, &errorsStr, &buildTimeMs, &domainsJSON, &createdAt)
					if err == nil {
						snap["ID"] = id
						snap["BuildTime"] = buildTime
						snap["TotalDomains"] = int(totalDomains64)
						snap["TotalSources"] = int(totalSources64)
						snap["TotalFetched"] = int(totalFetched64)
						snap["TotalParsed"] = int(totalParsed64)
						snap["Duplicates"] = int(duplicates64)
						snap["Errors"] = errorsStr
						snap["BuildTimeMs"] = buildTimeMs
						snap["CreatedAt"] = createdAt
						pageData["Snapshots"] = append(pageData["Snapshots"].([]map[string]interface{}), snap)
					}
				}
			}
		}
		s.render(w, r, pageData["Title"].(string), "history_diff.html", pageData)
		return
	}

	if s.historySvc != nil {
		diff, err := s.historySvc.DiffSnapshots(s1, s2)
		if err == nil && diff != nil {
			pageData["Diff"] = map[string]interface{}{
				"AddedCount":    diff.AddedCount,
				"RemovedCount":  diff.RemovedCount,
				"TotalDomains1": diff.TotalDomains1,
				"TotalDomains2": diff.TotalDomains2,
				"Added":         diff.Added,
				"Removed":       diff.Removed,
			}
		} else {
			pageData["Diff"] = map[string]interface{}{"error": err.Error()}
		}
	} else {
		pageData["Diff"] = map[string]interface{}{"error": "history service not available"}
	}

	s.render(w, r, "Сравнение снимков", "history_diff.html", pageData)
}

// Diagnostics handler.
func (s *UIService) Diagnostics(w http.ResponseWriter, r *http.Request) {
	pageData := map[string]interface{}{
		"Title":       "Диагностика",
		"Current":     "diagnostics",
		"Diagnostics": map[string]interface{}{},
	}

	result, err := s.diagnosticsSvc.RunDiagnostics()
	if err != nil {
		fmt.Printf("Diagnostics error: %v\n", err)
	}

	if result != nil {
		data := map[string]interface{}{
			"OverallSummary": result.OverallSummary,
			"Intersections":  result.Intersections,
			"ParsingErrors":  result.ParsingErrors,
			"InvalidDomains": result.InvalidDomains,
			"AnalyzedAt":     result.AnalyzedAt,
		}
		pageData["Diagnostics"] = data
	}

	s.render(w, r, "Диагностика", "diagnostics.html", pageData)
}

// Intersections handler.
func (s *UIService) Intersections(w http.ResponseWriter, r *http.Request) {
	pageData := map[string]interface{}{
		"Title":         "Пересечения",
		"Current":       "intersections",
		"Intersections": map[string]interface{}{},
	}

	report, err := s.intersectionSvc.Analyze()
	if err != nil {
		fmt.Printf("Intersection analysis error: %v\n", err)
	}

	if report != nil && report.Report != nil {
		data := map[string]interface{}{
			"IntersectingDomains": report.Report.IntersectingDomains,
			"UniqueDomains":       report.Report.UniqueDomains,
			"Summary":             report.Report.Summary,
			"SourceDomains":       report.Report.SourceDomains,
			"AnalyzedAt":          report.Report.AnalyzedAt,
		}
		pageData["Intersections"] = data
	}

	s.render(w, r, "Пересечения", "intersections.html", pageData)
}

// Scheduler handler.
func (s *UIService) Scheduler(w http.ResponseWriter, r *http.Request) {
	pageData := map[string]interface{}{
		"Title":   "Планировщик",
		"Current": "scheduler",
	}

	status := s.schedulerSvc.GetStatus()
	data := map[string]interface{}{
		"Status": map[string]interface{}{
			"Running":     status.Running,
			"LastUpdate":  status.LastUpdate,
			"NextUpdate":  status.NextUpdate,
			"UpdateTime":  status.UpdateTime.Milliseconds(),
			"UpdateCount": status.UpdateCount,
			"SourceCount": status.SourceCount,
			"ErrorCount":  status.ErrorCount,
		},
	}
	pageData["Status"] = data["Status"]

	s.render(w, r, "Планировщик", "scheduler.html", pageData)
}

// Build handler.
func (s *UIService) Build(w http.ResponseWriter, r *http.Request) {
	pageData := map[string]interface{}{
		"Title":       "Сборка",
		"Current":     "build",
		"BuildStatus": map[string]interface{}{"initialized": true},
	}

	if s.builder != nil {
		sources, err := s.sourceRepo.List()
		outputPath := ""
		hasOutput := false
		totalDomains := 0
		totalSources := len(sources)
		if err == nil {
			totalDomains = len(sources)
			outputPath = s.builder.OutputPath()
			if _, statErr := os.Stat(outputPath); statErr == nil {
				hasOutput = true
			}
		}
		if s.snapshotDB != nil {
			var dbCount int
			dbCount, err = s.CountDomains()
			if err == nil && dbCount > 0 {
				totalDomains = dbCount
			}
		}
		host := r.Host
		downloadURL := ""
		if host != "" {
			downloadURL = "http://" + host + "/api/v1/domains/export-txt"
		}
		data := map[string]interface{}{
			"TotalDomains": totalDomains,
			"TotalSources": totalSources,
			"TotalFetched": 0,
			"BuildTimeMs":  0,
			"HasOutput":    hasOutput,
			"OutputPath":   outputPath,
			"HasDomains":   totalDomains > 0,
			"DownloadURL":  downloadURL,
			"LastBuildTime": "—",
		}
		pageData["BuildStatus"] = data
	}

	s.render(w, r, "Сборка", "build.html", pageData)
}

// Settings handler.
func (s *UIService) Settings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.settingsSvc.GetMap()

	pageData := map[string]interface{}{
		"Title":   "Настройки",
		"Current": "settings",
	}

	if err == nil {
		setList := make([]map[string]interface{}, 0, len(settings))
		for k, v := range settings {
			setList = append(setList, map[string]interface{}{
				"Key":    k,
				"Value":  v,
				"String": fmt.Sprintf("%v", v),
			})
		}
		pageData["Settings"] = setList
	}

	s.render(w, r, "Настройки", "settings.html", pageData)
}

// AuthInfo handler.
func (s *UIService) AuthInfo(w http.ResponseWriter, r *http.Request) {
	cookie, hasCookie := s.getCookieFromRequest(r)

	pageData := map[string]interface{}{
		"Title":     "Аутентификация",
		"Current":   "auth",
		"HasCookie": hasCookie,
	}
	if hasCookie {
		pageData["Cookie"] = cookie
	}

	s.render(w, r, "Аутентификация", "auth.html", pageData)
}

// Dashboard handler.
func (s *UIService) Dashboard(w http.ResponseWriter, r *http.Request) {
	dashData, err := s.dashboardSvc.GetDashboardData()
	if err != nil {
		fmt.Printf("Dashboard data error: %v\n", err)
	}

	s.render(w, r, "Дашборд", "dashboard.html", map[string]interface{}{
		"Dashboard": dashData,
	})
}
