package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/auth"
	"domain-list-manager/internal/builder"
	"domain-list-manager/internal/config"
	"domain-list-manager/internal/database"
	"domain-list-manager/internal/diagnostics"
	"domain-list-manager/internal/domain"
	"domain-list-manager/internal/fetcher"
	"domain-list-manager/internal/history"
	"domain-list-manager/internal/intersection"
	"domain-list-manager/internal/repository"
	"domain-list-manager/internal/scheduler"
	"domain-list-manager/internal/source"
	"domain-list-manager/internal/setting"
	"domain-list-manager/internal/update_metadata"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds the HTTP server dependencies.
	type Server struct {
		Router      *chi.Mux
		cfg         config.Config
		domain      domain.Repository
		authService *auth.Service
		authHandler *auth.Handler
		settingsSvc *setting.Service
		settingsHdl *settingsHandler
		sourceSvc       *source.Service
		sourceHdl       *sourceHandler
		customDomainHdl *customDomainHandler
		builderHdl        *builderHandler
		historySvc      *history.HistoryService
		historyHdl      *historyHandler
		intersectionSvc *intersection.Service
		intersectionHdl *intersectionHandler
		dashboardHdl    *dashboardHandler
		snapshotDB      *sql.DB
		schedulerSvc    *scheduler.Scheduler
		schedulerHdl    *schedulerHandler
		diagnosticsSvc  *diagnostics.Service
		diagnosticsHdl  *diagnosticsHandler
		uiSvc           *UIService
		settingsRepo    setting.Repository
	}

// NewServer creates a new Server instance.
func NewServer(cfg config.Config) *Server {
	s := &Server{
		Router: chi.NewRouter(),
		cfg:    cfg,
	}

	db, err := database.Init(cfg.Database.Path)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize database: %v", err))
	}

	s.domain = repository.NewDomainRepository(db)

	sessionRepo := auth.NewSessionRepository(db)
	s.authService = auth.NewService(sessionRepo, cfg.Auth.Username, cfg.Auth.Password)
	s.authHandler = auth.NewHandler(s.authService)

	settingsRepo := setting.NewSettingRepository(db)
	s.settingsSvc = setting.NewService(settingsRepo)
	s.settingsHdl = newSettingsHandler(s.settingsSvc, settingsRepo)

	sourceRepo := source.NewSourceRepository(db)
	s.sourceSvc = source.NewService(sourceRepo)
	s.sourceHdl = newSourceHandler(s.sourceSvc, sourceRepo)

	builderCfg := builder.BuilderConfig{
		Timeout:    time.Duration(cfg.Fetcher.Timeout) * time.Second,
		MaxRetries: cfg.Fetcher.MaxRetries,
		OutputPath: cfg.Builder.OutputPath,
	}
	builderInst := builder.NewBuilder(sourceRepo, s.domain, builderCfg)

	s.customDomainHdl = newCustomDomainHandler(s.domain)
	s.customDomainHdl.builder = builderInst

	s.snapshotDB = db
	snapshotRepo := history.NewSnapshotRepository(db)
	s.historySvc = history.NewHistoryService(snapshotRepo, cfg.Builder.SnapshotCount)
	s.historyHdl = newHistoryHandler(s.historySvc, db)

	s.builderHdl = newBuilderHandler(builderInst, sourceRepo, s.historySvc)

	intFetcher := fetcher.New(
		fetcher.WithTimeout(time.Duration(cfg.Fetcher.Timeout) * time.Second),
		fetcher.WithMaxRetries(cfg.Fetcher.MaxRetries),
	)
	s.intersectionSvc = intersection.NewService(sourceRepo, intFetcher, s.domain)
	s.intersectionHdl = newIntersectionHandler(s.intersectionSvc)

		metadataRepo := update_metadata.NewRepository(db)
		s.schedulerSvc = scheduler.NewScheduler(sourceRepo, metadataRepo, intFetcher, s.domain, 15*time.Minute)
		s.schedulerHdl = newSchedulerHandler(s.schedulerSvc)

		s.dashboardHdl = newDashboardHandler(db)

		s.diagnosticsSvc = diagnostics.NewService(sourceRepo, db, intFetcher)
		s.diagnosticsHdl = newDiagnosticsHandler(s.diagnosticsSvc)

		s.settingsRepo = settingsRepo

		s.uiSvc = NewUIService(
			db,
			s.dashboardHdl.svc,
			s.diagnosticsSvc,
			s.historySvc,
			s.sourceSvc,
			s.intersectionSvc,
			s.schedulerSvc,
			s.builderHdl.builder,
			sourceRepo,
			s.settingsSvc,
			s.authService,
			s.authHandler,
			s.settingsRepo,
		)

	if err := s.settingsSvc.EnsureDefaults(); err != nil {
		panic(fmt.Sprintf("failed to ensure default settings: %v", err))
	}

	if err := s.settingsSvc.ApplyToConfig(&s.cfg); err != nil {
		panic(fmt.Sprintf("failed to apply settings: %v", err))
	}

	return s
}

// SetupRoutes registers all application routes.
func (s *Server) SetupRoutes() {
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.RequestID)
	s.Router.Use(middleware.RealIP)

	// Public routes
	s.Router.HandleFunc("/", s.healthCheck)
	s.Router.Get("/login", s.authHandler.LoginPage)
	s.Router.Post("/login", s.authHandler.Login)
	s.Router.HandleFunc("/logout", s.authHandler.Logout)

	// API v1 routes
	apiRouter := chi.NewRouter()
	apiRouter.Use(s.authService.Middleware)

	// Domains
	apiRouter.Get("/domains", s.listDomains)
	apiRouter.Post("/domains", s.createDomain)
	apiRouter.Get("/domains/{id}", s.getDomain)
	apiRouter.Put("/domains/{id}", s.updateDomain)
	apiRouter.Delete("/domains/{id}", s.deleteDomain)
	apiRouter.Delete("/domains", s.deleteDomainByName)
	apiRouter.Post("/domains/bulk", s.customDomainHdl.BulkAdd)
	apiRouter.Post("/domains/import-txt", s.customDomainHdl.ImportTXT)
	apiRouter.Post("/domains/import-from-url", s.customDomainHdl.ImportFromURL)
	apiRouter.Get("/domains/export-txt", s.customDomainHdl.ExportTXT)

	// Settings
	apiRouter.Get("/settings", s.settingsHdl.List)
	apiRouter.Post("/settings", s.settingsHdl.Create)
	apiRouter.Get("/settings/{key}", s.settingsHdl.GetSetting)
	apiRouter.Put("/settings/{key}", s.settingsHdl.UpdateSetting)
	apiRouter.Delete("/settings/{key}", s.settingsHdl.Delete)

	// Sources
	apiRouter.Get("/sources", s.sourceHdl.List)
	apiRouter.Get("/sources/{id}", s.sourceHdl.GetSource)
	apiRouter.Post("/sources", s.sourceHdl.Create)
	apiRouter.Put("/sources/{id}", s.sourceHdl.UpdateSource)
	apiRouter.Delete("/sources/{id}", s.sourceHdl.Delete)
	apiRouter.Post("/sources/{id}/enable", s.sourceHdl.Enable)
	apiRouter.Post("/sources/{id}/disable", s.sourceHdl.Disable)

	// Build
	apiRouter.Post("/build", s.builderHdl.Build)
	apiRouter.Get("/build/status", s.builderHdl.GetStatus)
	apiRouter.Post("/build/write", s.builderHdl.WriteOutput)

	// History
	apiRouter.Get("/history", s.historyHdl.List)
	apiRouter.Get("/history/{id}", s.historyHdl.Get)
	apiRouter.Delete("/history/{id}", s.historyHdl.Delete)
	apiRouter.Get("/history/diff", s.historyHdl.Diff)
	apiRouter.Get("/history/{id}/domains", s.historyHdl.GetDomains)

	// Intersection
	apiRouter.Get("/intersections", s.intersectionHdl.Analyze)
	apiRouter.Get("/intersections/sources/{domain}", s.intersectionHdl.GetSourcesForDomain)

	// Scheduler
	apiRouter.Get("/scheduler/status", s.schedulerHdl.GetStatus)
	apiRouter.Post("/scheduler/start", s.schedulerHdl.Start)
	apiRouter.Post("/scheduler/stop", s.schedulerHdl.Stop)
	apiRouter.Post("/scheduler/trigger", s.schedulerHdl.TriggerUpdate)

	// Diagnostics
	apiRouter.Get("/diagnostics", s.diagnosticsHdl.RunDiagnostics)
	apiRouter.Get("/diagnostics/source/{id}", s.diagnosticsHdl.GetSourceDiagnostics)

	// Dashboard
	apiRouter.Get("/dashboard", s.dashboardHdl.GetDashboard)

	// API Documentation (protected, requires auth)
	apiRouter.Get("/docs", api.GetAPIDocumentation)

	s.Router.Mount("/api/v1", apiRouter)

	// API Documentation (public, no auth required)
	s.Router.Get("/docs", api.GetAPIDocumentation)

	// UI routes (mounted at /ui/*)
	s.uiSvc.ServeHTTP(s.Router, "/ui")
}

// healthCheck returns a simple health check endpoint.
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// listDomains returns all domains in the repository.
func (s *Server) listDomains(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	sourceID := r.URL.Query().Get("source")

	var domains []*domain.Domain
	var err error

	if sourceID != "" {
		domains, err = s.domain.ListBySource(sourceID)
	} else if search != "" {
		domains, err = s.domain.Search(search)
	} else {
		domains, err = s.domain.List()
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to list domains: " + err.Error(),
		})
		return
	}

	if domains == nil {
		domains = []*domain.Domain{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    domains,
	})
}

// createDomain creates a new domain.
func (s *Server) createDomain(w http.ResponseWriter, r *http.Request) {
	var d domain.Domain
	
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		log.Printf("createDomain: decode error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid JSON: " + err.Error(),
		})
		return
	}
	
	log.Printf("createDomain: name=%q id=%q source_id=%v", d.Name, d.ID, d.SourceID)
	
	// Strip comments and BOM
	d.Name = strings.TrimSpace(d.Name)
	if len(d.Name) >= 3 && d.Name[:3] == "\xef\xbb\xbf" {
		d.Name = d.Name[3:]
	}
	d.Name = strings.TrimSpace(d.Name)
	for len(d.Name) > 0 && d.Name[0] == 0xC2 && len(d.Name) > 1 && d.Name[1] == 0xA0 {
		d.Name = d.Name[2:]
	}
	d.Name = strings.TrimSpace(d.Name)
	if idx := strings.Index(d.Name, "#"); idx != -1 {
		d.Name = strings.TrimSpace(d.Name[:idx])
	}
	
	if d.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Domain name cannot be empty or only comments",
		})
		return
	}
	
	// Set source_id for manually added domains
	if d.SourceID == nil || *d.SourceID == "" {
		sourceID := "manual"
		d.SourceID = &sourceID
	}
	
	// Set a default ID if not provided
	now := time.Now().UTC()
	if d.ID == "" {
		d.ID = fmt.Sprintf("%d", now.UnixNano())
	}
	d.CreatedAt = now
	d.UpdatedAt = now
	
	log.Printf("createDomain: after id set: id=%q name=%q source_id=%q", d.ID, d.Name, *d.SourceID)
	
	if err := s.domain.Create(&d); err != nil {
		log.Printf("createDomain: repo.Create error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to create domain: " + err.Error(),
		})
		return
	}
	
	log.Printf("createDomain: domain created successfully: id=%q name=%q source_id=%q", d.ID, d.Name, *d.SourceID)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    d,
	})
}

// getDomain retrieves a specific domain by ID.
func (s *Server) getDomain(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	domain, err := s.domain.Get(id)
	if err != nil {
		if err.Error() == "domain not found" {
			http.Error(w, "Domain not found", http.StatusNotFound)
			return
		}
		
		http.Error(w, fmt.Sprintf("Failed to get domain: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain)
}

// updateDomain updates an existing domain.
func (s *Server) updateDomain(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	var d domain.Domain
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	d.ID = id // Ensure ID is set correctly
	
	// Strip comments and BOM
	d.Name = strings.TrimSpace(d.Name)
	if len(d.Name) >= 3 && d.Name[:3] == "\xef\xbb\xbf" {
		d.Name = d.Name[3:]
	}
	d.Name = strings.TrimSpace(d.Name)
	for len(d.Name) > 0 && d.Name[0] == 0xC2 && len(d.Name) > 1 && d.Name[1] == 0xA0 {
		d.Name = d.Name[2:]
	}
	d.Name = strings.TrimSpace(d.Name)
	if idx := strings.Index(d.Name, "#"); idx != -1 {
		d.Name = strings.TrimSpace(d.Name[:idx])
	}
	
	if d.Name == "" {
		http.Error(w, "Domain name cannot be empty", http.StatusBadRequest)
		return
	}
	
	if err := s.domain.Update(id, &d); err != nil {
		if err.Error() == "domain not found" {
			http.Error(w, "Domain not found", http.StatusNotFound)
			return
		}
		
		http.Error(w, fmt.Sprintf("Failed to update domain: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

// deleteDomain removes a domain by ID.
func (s *Server) deleteDomain(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	if err := s.domain.Delete(id); err != nil {
		if err.Error() == "domain not found" {
			http.Error(w, "Domain not found", http.StatusNotFound)
			return
		}
		
		http.Error(w, fmt.Sprintf("Failed to delete domain: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Domain deleted successfully"})
}

func (s *Server) deleteDomainByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.WriteMethodNotAllowed(w, "DELETE")
		return
	}

	name := r.URL.Query().Get("name")
	sourceID := r.URL.Query().Get("source_id")

	if sourceID == "" {
		api.WriteBadRequest(w, "Source ID is required for deletion")
		return
	}

	if err := s.domain.DeleteByNameAndSource(name, sourceID); err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to delete domain: %v", err))
		return
	}

	api.WriteSuccess(w, map[string]string{"message": "Domain deleted successfully"}, "Domain deleted successfully")
}
