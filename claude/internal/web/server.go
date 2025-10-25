package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andrewsjg/simple-healthchecker/claude/internal/config"
	"github.com/andrewsjg/simple-healthchecker/claude/pkg/models"
)

//go:embed templates/*
var templatesFS embed.FS

// HostStatus represents the status of a host with check results
type HostStatus struct {
	models.Host
	Checks []CheckStatus
}

// CheckStatus represents a check with its last result
type CheckStatus struct {
	models.Check
	LastResult *models.CheckResult
}

// Server represents the web server
type Server struct {
	config     *models.Config
	configPath string
	port       int
	results    map[string]map[models.CheckType]*models.CheckResult
	resultsMux sync.RWMutex
	configMux  sync.RWMutex
	templates  *template.Template
}

// NewServer creates a new web server
func NewServer(config *models.Config, configPath string, port int) (*Server, error) {
	// Create template with custom functions
	tmpl := template.New("").Funcs(template.FuncMap{
		"json": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	})

	tmpl, err := tmpl.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Server{
		config:     config,
		configPath: configPath,
		port:       port,
		results:    make(map[string]map[models.CheckType]*models.CheckResult),
		templates:  tmpl,
	}, nil
}

// UpdateResult updates the result for a host/check
func (s *Server) UpdateResult(result models.CheckResult) {
	s.resultsMux.Lock()
	defer s.resultsMux.Unlock()

	if s.results[result.Host] == nil {
		s.results[result.Host] = make(map[models.CheckType]*models.CheckResult)
	}
	s.results[result.Host][result.CheckType] = &result
}

// Start starts the web server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/hosts", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact match, not paths starting with /api/hosts/
		if r.URL.Path == "/api/hosts" {
			s.handleGetHosts(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/api/hosts/", s.handleAPIRoutes)
	mux.HandleFunc("/api/host/add", s.handleAddHost)
	mux.HandleFunc("/api/host/edit", s.handleEditHost)
	mux.HandleFunc("/api/host/delete", s.handleDeleteHost)
	mux.HandleFunc("/api/host/add-form", s.handleGetAddForm)
	mux.HandleFunc("/api/host/edit-form", s.handleGetEditForm)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down web server: %v", err)
		}
	}()

	log.Printf("Web server starting on port %d", s.port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("web server error: %w", err)
	}

	return nil
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := s.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleGetHosts(w http.ResponseWriter, r *http.Request) {
	s.resultsMux.RLock()
	defer s.resultsMux.RUnlock()

	var hostStatuses []HostStatus
	for _, host := range s.config.Hosts {
		status := HostStatus{
			Host:   host,
			Checks: make([]CheckStatus, 0, len(host.Checks)),
		}

		for _, check := range host.Checks {
			checkStatus := CheckStatus{
				Check: check,
			}

			if hostResults, ok := s.results[host.Name]; ok {
				if result, ok := hostResults[check.Type]; ok {
					checkStatus.LastResult = result
				}
			}

			status.Checks = append(status.Checks, checkStatus)
		}

		hostStatuses = append(hostStatuses, status)
	}

	data := struct {
		Hosts []HostStatus
	}{
		Hosts: hostStatuses,
	}

	// Set Content-Type before writing
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := s.templates.ExecuteTemplate(w, "hosts.html", data); err != nil {
		// Log error but don't try to write response again as headers are already sent
		log.Printf("Error rendering template: %v", err)
		return
	}
}

func (s *Server) handleAPIRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse URL: /api/hosts/{hostName}/checks/{checkType}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	parts := strings.Split(path, "/")

	if len(parts) == 4 && parts[1] == "checks" {
		s.handleCheckToggle(w, r)
		return
	}

	http.NotFound(w, r)
}

func (s *Server) handleCheckToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL: /api/hosts/{hostName}/checks/{checkType}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	parts := strings.Split(path, "/")
	if len(parts) != 4 || parts[1] != "checks" {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	hostName := parts[0]
	checkType := models.CheckType(parts[2])
	action := parts[3]

	if action != "enable" && action != "disable" {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// Find the host and check
	var found bool
	for i, host := range s.config.Hosts {
		if host.Name == hostName {
			for j, check := range host.Checks {
				if check.Type == checkType {
					s.config.Hosts[i].Checks[j].Enabled = (action == "enable")
					found = true
					break
				}
			}
			break
		}
	}

	if !found {
		http.Error(w, "Host or check not found", http.StatusNotFound)
		return
	}

	// Return updated hosts list
	s.handleGetHosts(w, r)
}

// GetConfig returns the current configuration (thread-safe)
func (s *Server) GetConfig() *models.Config {
	s.configMux.RLock()
	defer s.configMux.RUnlock()
	return s.config
}

// saveConfig saves the current configuration to disk
func (s *Server) saveConfig() error {
	s.configMux.RLock()
	defer s.configMux.RUnlock()

	if s.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	return config.SaveConfig(s.configPath, s.config)
}

// Handler for JSON API (optional, for debugging)
func (s *Server) handleGetHostsJSON(w http.ResponseWriter, r *http.Request) {
	s.resultsMux.RLock()
	defer s.resultsMux.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.config)
}

// handleAddHost adds a new host with checks
func (s *Server) handleAddHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	hostName := r.FormValue("name")
	hostAddress := r.FormValue("address")

	if hostName == "" || hostAddress == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	s.configMux.Lock()
	defer s.configMux.Unlock()

	// Check if host already exists
	for _, host := range s.config.Hosts {
		if host.Name == hostName {
			http.Error(w, "Host with this name already exists", http.StatusConflict)
			return
		}
	}

	// Parse checks from form
	checks := parseChecksFromForm(r)

	// Create new host
	newHost := models.Host{
		Name:    hostName,
		Address: hostAddress,
		Checks:  checks,
	}

	// Add host to config
	s.config.Hosts = append(s.config.Hosts, newHost)

	// Save configuration
	if err := config.SaveConfig(s.configPath, s.config); err != nil {
		log.Printf("Failed to save configuration: %v", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	// Return updated hosts list
	s.configMux.Unlock()
	s.handleGetHosts(w, r)
	s.configMux.Lock()
}

// handleEditHost edits an existing host
func (s *Server) handleEditHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	originalName, _ := url.QueryUnescape(r.FormValue("original_name"))
	hostName := r.FormValue("name")
	hostAddress := r.FormValue("address")

	if originalName == "" || hostName == "" || hostAddress == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	s.configMux.Lock()
	defer s.configMux.Unlock()

	// Find the host
	hostIndex := -1
	for i, host := range s.config.Hosts {
		if host.Name == originalName {
			hostIndex = i
			break
		}
	}

	if hostIndex == -1 {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	// Check if new name conflicts with existing host (unless it's the same host)
	if originalName != hostName {
		for _, host := range s.config.Hosts {
			if host.Name == hostName {
				http.Error(w, "Host with this name already exists", http.StatusConflict)
				return
			}
		}
	}

	// Parse checks from form
	checks := parseChecksFromForm(r)

	// Update host
	s.config.Hosts[hostIndex].Name = hostName
	s.config.Hosts[hostIndex].Address = hostAddress
	s.config.Hosts[hostIndex].Checks = checks

	// Save configuration
	if err := config.SaveConfig(s.configPath, s.config); err != nil {
		log.Printf("Failed to save configuration: %v", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	// Return updated hosts list
	s.configMux.Unlock()
	s.handleGetHosts(w, r)
	s.configMux.Lock()
}

// handleDeleteHost deletes a host
func (s *Server) handleDeleteHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	hostName, _ := url.QueryUnescape(r.FormValue("name"))

	if hostName == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	s.configMux.Lock()
	defer s.configMux.Unlock()

	// Find and remove the host
	hostIndex := -1
	for i, host := range s.config.Hosts {
		if host.Name == hostName {
			hostIndex = i
			break
		}
	}

	if hostIndex == -1 {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	// Remove host from slice
	s.config.Hosts = append(s.config.Hosts[:hostIndex], s.config.Hosts[hostIndex+1:]...)

	// Save configuration
	if err := config.SaveConfig(s.configPath, s.config); err != nil {
		log.Printf("Failed to save configuration: %v", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	// Return updated hosts list
	s.configMux.Unlock()
	s.handleGetHosts(w, r)
	s.configMux.Lock()
}

// parseChecksFromForm parses check data from form submission
func parseChecksFromForm(r *http.Request) []models.Check {
	var checks []models.Check

	// Get array of check data
	checkTypes := r.Form["check_type[]"]
	checkEnabled := r.Form["check_enabled[]"]
	checkTimeouts := r.Form["check_timeout[]"]
	checkHealthcheckURLs := r.Form["check_healthcheck_url[]"]
	checkHTTPURLs := r.Form["check_http_url[]"]
	checkHTTPStatuses := r.Form["check_http_status[]"]

	for i := 0; i < len(checkTypes); i++ {
		if checkTypes[i] == "" {
			continue
		}

		enabled := false
		if i < len(checkEnabled) && checkEnabled[i] == "on" {
			enabled = true
		}

		timeout := 5 // default
		if i < len(checkTimeouts) {
			if t, err := strconv.Atoi(checkTimeouts[i]); err == nil {
				timeout = t
			}
		}

		healthcheckURL := ""
		if i < len(checkHealthcheckURLs) {
			healthcheckURL = checkHealthcheckURLs[i]
		}

		// Parse HTTP-specific options
		options := make(map[string]string)
		if checkTypes[i] == "http" {
			if i < len(checkHTTPURLs) && checkHTTPURLs[i] != "" {
				options["url"] = checkHTTPURLs[i]
			}
			if i < len(checkHTTPStatuses) && checkHTTPStatuses[i] != "" {
				options["expected_status"] = checkHTTPStatuses[i]
			} else {
				options["expected_status"] = "200"
			}
		}

		check := models.Check{
			Type:             models.CheckType(checkTypes[i]),
			Enabled:          enabled,
			Timeout:          models.Duration(time.Duration(timeout) * time.Second),
			HealthcheckIOURL: healthcheckURL,
			Options:          options,
		}

		checks = append(checks, check)
	}

	// If no checks provided, add a default ping check
	if len(checks) == 0 {
		checks = append(checks, models.Check{
			Type:    models.CheckTypePing,
			Enabled: true,
			Timeout: models.Duration(5 * time.Second),
		})
	}

	return checks
}

// handleGetAddForm returns the add host form
func (s *Server) handleGetAddForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render add form with one default check
	data := struct {
		Title      string
		Action     string
		Host       *models.Host
		ShowDelete bool
	}{
		Title:  "Add New Host",
		Action: "/api/host/add",
		Host: &models.Host{
			Name:    "",
			Address: "",
			Checks: []models.Check{
				{
					Type:    models.CheckTypePing,
					Enabled: true,
					Timeout: models.Duration(5 * time.Second),
				},
			},
		},
		ShowDelete: false,
	}

	if err := s.templates.ExecuteTemplate(w, "host-form.html", data); err != nil {
		log.Printf("Error rendering add form: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleGetEditForm returns the edit host form
func (s *Server) handleGetEditForm(w http.ResponseWriter, r *http.Request) {
	hostName, _ := url.QueryUnescape(r.URL.Query().Get("name"))

	if hostName == "" {
		http.Error(w, "Missing host name", http.StatusBadRequest)
		return
	}

	s.configMux.RLock()
	defer s.configMux.RUnlock()

	// Find the host
	var foundHost *models.Host
	for _, host := range s.config.Hosts {
		if host.Name == hostName {
			foundHost = &host
			break
		}
	}

	if foundHost == nil {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := struct {
		Title      string
		Action     string
		Host       *models.Host
		ShowDelete bool
	}{
		Title:      "Edit Host",
		Action:     "/api/host/edit",
		Host:       foundHost,
		ShowDelete: true,
	}

	if err := s.templates.ExecuteTemplate(w, "host-form.html", data); err != nil {
		log.Printf("Error rendering edit form: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
