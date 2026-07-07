package web

import (
	"bytes"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"domain-list-manager/internal/auth"
	"domain-list-manager/internal/builder"
	"domain-list-manager/internal/dashboard"
	"domain-list-manager/internal/diagnostics"
	"domain-list-manager/internal/history"
	"domain-list-manager/internal/intersection"
	"domain-list-manager/internal/scheduler"
	"domain-list-manager/internal/setting"
	"domain-list-manager/internal/source"

	"github.com/go-chi/chi/v5"
)

//go:embed templates/ui/*.html
var templateFS embed.FS

const apiBase = "/api/v1"

// UIService manages the Admin Dashboard UI pages and API client.
type UIService struct {
	dashboardSvc     *dashboard.Service
	diagnosticsSvc   *diagnostics.Service
	historySvc       *history.HistoryService
	sourceSvc        *source.Service
	intersectionSvc  *intersection.Service
	schedulerSvc     *scheduler.Scheduler
	builder          *builder.Builder
	sourceRepo       source.Repository
	settingsSvc      *setting.Service
	authService      *auth.Service
	authHandler      *auth.Handler
	settingsRepo     setting.Repository
	snapshotDB       *sql.DB
	cookieName       string
	templates        *template.Template
	apiCookies       map[string]string
	mu               sync.RWMutex
}

func loadTemplates() *template.Template {
	subFS, err := fs.Sub(templateFS, "templates/ui")
	if err != nil {
		fmt.Printf("Failed to sub template FS: %v\n", err)
		return template.New("root")
	}
	t, err := template.New("root").ParseFS(subFS, "*.html")
	if err != nil {
		fmt.Printf("Failed to parse templates from embed FS: %v\n", err)
		return template.New("root")
	}
	var names []string
	for _, tmpl := range t.Templates() {
		names = append(names, tmpl.Name())
	}
	fmt.Printf("SUCCESS: loaded %d templates: %v\n", len(names), names)
	return t
}

var globalTemplates *template.Template

func init() {
	globalTemplates = loadTemplates()
}

// NewUIService creates a new UI service.
func NewUIService(
	db *sql.DB,
	dashboardSvc *dashboard.Service,
	diagnosticsSvc *diagnostics.Service,
	historySvc *history.HistoryService,
	sourceSvc *source.Service,
	intersectionSvc *intersection.Service,
	schedulerSvc *scheduler.Scheduler,
	builder *builder.Builder,
	sourceRepo source.Repository,
	settingsSvc *setting.Service,
	authService *auth.Service,
	authHandler *auth.Handler,
	settingsRepo setting.Repository,
) *UIService {
	ui := &UIService{
		dashboardSvc:     dashboardSvc,
		diagnosticsSvc:   diagnosticsSvc,
		historySvc:       historySvc,
		sourceSvc:        sourceSvc,
		intersectionSvc:  intersectionSvc,
		schedulerSvc:     schedulerSvc,
		builder:          builder,
		sourceRepo:       sourceRepo,
		settingsSvc:      settingsSvc,
		authService:      authService,
		authHandler:      authHandler,
		snapshotDB:       db,
		settingsRepo:     settingsRepo,
		cookieName:       "session_token",
		apiCookies:       make(map[string]string),
		templates:        globalTemplates,
	}
	return ui
}

// SetAuthCookie stores auth cookie for API calls.
func (s *UIService) SetAuthCookie(name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiCookies[name] = value
}

// GetAPIResponse fetches data from the API and decodes into target.
func (s *UIService) GetAPIResponse(path string, target interface{}) error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, apiBase+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	s.mu.RLock()
	for name, value := range s.apiCookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	s.mu.RUnlock()

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// APIResponseWrapper wraps standard API responses.
type APIResponseWrapper struct {
	Success bool              `json:"success"`
	Data    json.RawMessage   `json:"data"`
	Message string            `json:"message"`
}

// DashboardPageData holds data for the dashboard page.
type DashboardPageData struct {
	Title       string
	Current     string
	Dashboard   map[string]interface{}
	SourceStats []map[string]interface{}
	BuildStats  map[string]interface{}
	Errors      []string
}

// DomainsPageData holds data for the domains page.
type DomainsPageData struct {
	Title   string
	Current string
	Domains []map[string]interface{}
	Search  string
	Sort    string
	Dir     string
}

// SourcesPageData holds data for the sources page.
type SourcesPageData struct {
	Title   string
	Current string
	Sources []map[string]interface{}
	Enabled int
	Disabled int
	Search  string
}

// HistoryPageData holds data for the history page.
type HistoryPageData struct {
	Title       string
	Current     string
	Snapshots   []map[string]interface{}
	TotalCount  int
	Page        int
	PageSize    int
}

// DiffPageData holds data for the diff page.
type DiffPageData struct {
	Title        string
	Current      string
	Diff         map[string]interface{}
	Snapshot1ID  string
	Snapshot2ID  string
}

// DiagnosticsPageData holds data for the diagnostics page.
type DiagnosticsPageData struct {
	Title        string
	Current      string
	Diagnostics  map[string]interface{}
}

// IntersectionsPageData holds data for the intersections page.
type IntersectionsPageData struct {
	Title           string
	Current         string
	Intersections   map[string]interface{}
}

// SchedulerPageData holds data for the scheduler page.
type SchedulerPageData struct {
	Title     string
	Current   string
	Status    map[string]interface{}
	Running   bool
}

// BuildPageData holds data for the build page.
type BuildPageData struct {
	Title       string
	Current     string
	BuildStatus map[string]interface{}
}

// SettingsPageData holds data for the settings page.
type SettingsPageData struct {
	Title    string
	Current  string
	Settings []map[string]interface{}
}

// AuthPageData holds data for the auth page.
type AuthPageData struct {
	Title string
}

// LoginRequiredData is used when authentication is required.
type LoginRequiredData struct {
	Title   string
	Message string
}

// render renders a page with full base template.
func (s *UIService) render(w http.ResponseWriter, r *http.Request, title, tmpl string, data interface{}) {
	tmplName := tmpl
	if !strings.HasSuffix(tmplName, ".html") {
		tmplName = tmplName + ".html"
	}

	// Find the page template
	baseTmpl := s.templates.Lookup("base.html")
	if baseTmpl == nil {
		http.Error(w, "Base template not found", http.StatusInternalServerError)
		return
	}
	pageTmpl := s.templates.Lookup(tmplName)
	if pageTmpl == nil {
		http.Error(w, "Template not found: "+tmplName, http.StatusNotFound)
		return
	}

	// Execute the page template
	var contentBuf bytes.Buffer
	if err := pageTmpl.Execute(&contentBuf, data); err != nil {
		fmt.Printf("Template exec error [%s]: %v\n", tmplName, err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}

	pageData := map[string]interface{}{
		"Title":   title,
		"Current": strings.TrimSuffix(tmplName, ".html"),
		"Content": template.HTML(contentBuf.String()),
	}
	if data != nil {
		if m, ok := data.(map[string]interface{}); ok {
			for k, v := range m {
				pageData[k] = v
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := baseTmpl.Execute(w, pageData); err != nil {
		fmt.Printf("Failed to execute base template: %v\n", err)
	}
}

// getCookieFromRequest extracts session cookie from request.
func (s *UIService) getCookieFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(s.cookieName)
	if err != nil {
		return "", false
	}
	if cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

// ServeHTTP mounts UI routes.
func (s *UIService) ServeHTTP(router *chi.Mux, root string) {
	if !strings.HasSuffix(root, "/") {
		root = root + "/"
	}

	mux := chi.NewRouter()

	// Public UI routes (no auth required)
	mux.Get("/login", s.LoginPage)
	mux.Post("/login", s.LoginPage)
	mux.Get("/logout", s.Logout)

	// Default route
	mux.Get("/", s.Dashboard)

	// Protected UI routes (with auth middleware)
	protect := chi.NewRouter()
	protect.Use(s.requireAuthMiddleware)
	protect.Get("/dashboard", s.Dashboard)
	protect.Get("/domains", s.Domains)
	protect.Get("/sources", s.Sources)
	protect.Get("/history", s.History)
	protect.Get("/history/diff", s.HistoryDetail)
	protect.Get("/history/{id}", s.HistoryDetail)
	protect.Get("/diagnostics", s.Diagnostics)
	protect.Get("/intersections", s.Intersections)
	protect.Get("/scheduler", s.Scheduler)
	protect.Get("/build", s.Build)
	protect.Get("/settings", s.Settings)
	protect.Get("/auth", s.AuthInfo)

	mux.Mount("/", protect)
	router.Mount(root, mux)
}

// requireAuthMiddleware checks if the user is authenticated.
func (s *UIService) requireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, hasCookie := s.getCookieFromRequest(r)
		if !hasCookie {
			redirectURL := "/ui/login?redirect=" + r.URL.String()
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}

		if !s.authService.ValidateSession(cookie) {
			s.clearAuthCookie(w, r)
			redirectURL := "/ui/login?redirect=" + r.URL.String()
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}

		s.SetAuthCookie(s.cookieName, cookie)
		next.ServeHTTP(w, r)
	})
}

// clearAuthCookie removes the session cookie.
func (s *UIService) clearAuthCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// LoginPage handles the login page and login form submission.
func (s *UIService) LoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleLogin(w, r)
		return
	}

	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/ui/"
	}

	s.render(w, r, "Вход", "login.html", map[string]interface{}{
		"Title":   "Вход",
		"Current": "login",
		"Redirect": redirect,
	})
}

// handleLogin processes the login form submission.
func (s *UIService) handleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = r.URL.Query().Get("redirect")
	}
	if redirect == "" {
		redirect = "/ui/"
	}

	session, err := s.authService.Login(username, password)
	if err != nil {
		s.render(w, r, "Ошибка входа", "auth.html", map[string]interface{}{
			"Title":       "Ошибка входа",
			"Current":     "login",
			"Error":       "Неверное имя пользователя или пароль",
			"Redirect":    redirect,
			"IsLoginPage": true,
		})
		return
	}

	s.clearAuthCookie(w, r)
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    session.Token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	s.SetAuthCookie(s.cookieName, session.Token)
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// Logout handles user logout.
func (s *UIService) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, hasCookie := s.getCookieFromRequest(r)
	if hasCookie {
		s.authService.Logout(cookie)
	}
	s.clearAuthCookie(w, r)
	http.Redirect(w, r, "/ui/", http.StatusSeeOther)
}

func placeholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func toStringAny(s []string) []any {
	r := make([]any, len(s))
	for i, v := range s {
		r[i] = v
	}
	return r
}

// CountDomains returns the total number of domains from the database.
func (s *UIService) CountDomains() (int, error) {
	if s.snapshotDB == nil {
		return 0, fmt.Errorf("database not available")
	}
	var count int
	err := s.snapshotDB.QueryRow("SELECT COUNT(*) FROM domains").Scan(&count)
	return count, err
}
