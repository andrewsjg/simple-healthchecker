package server

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/andrewsjg/simple-healthchecker/copilot/internal/state"
)

//go:embed templates/*
var templatesFS embed.FS

type Server struct {
	st   *state.State
	http *http.Server
	tpl  *template.Template
}

func New(st *state.State) *Server {
	funcs := template.FuncMap{
		"slug": func(s string) string {
			b := make([]rune, 0, len(s))
			for _, r := range s {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
					b = append(b, r)
				} else {
					b = append(b, '-')
				}
			}
			return string(b)
		},
	}
	tpl := template.Must(template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/*.html", "templates/check_config_fragment.html"))
	return &Server{st: st, tpl: tpl}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/toggle", s.handleToggle)
	mux.HandleFunc("/hcurl", s.handleHCURL)
	mux.HandleFunc("/addhost", s.handleAddHost)
	mux.HandleFunc("/addhost-form", s.handleAddHostForm)
	mux.HandleFunc("/close-modal", s.handleCloseModal)
	mux.HandleFunc("/addhost-check-row", s.handleAddHostCheckRow)
	mux.HandleFunc("/hosts", s.handleHosts)
	mux.HandleFunc("/edithost-form", s.handleEditHostForm)
	mux.HandleFunc("/edithost", s.handleEditHost)
	mux.HandleFunc("/delhost", s.handleDeleteHost)
	mux.HandleFunc("/edithost-addcheck", s.handleEditAddCheck)
	mux.HandleFunc("/edithost-delcheck", s.handleEditDelCheck)
	mux.HandleFunc("/edithost-updatecheck", s.handleEditUpdateCheck)
	mux.HandleFunc("/check-config", s.handleCheckConfig)
	s.http = &http.Server{Addr: addr, Handler: logRequests(mux)}
	return s.http.ListenAndServe()
}

func (s *Server) Stop() error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(context.Background())
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := struct{ Hosts []*state.HostStatus }{Hosts: s.st.Snapshot()}
	_ = s.tpl.ExecuteTemplate(w, "index.html", data)
}

func (s *Server) handleToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	host := r.FormValue("host")
	idxStr := r.FormValue("idx")
	enabled := r.FormValue("enabled") == "true"
	idx, _ := strconv.Atoi(idxStr)
	s.st.Toggle(host, idx, enabled)
	fmt.Fprintf(w, toggleButton(host, idx, enabled))
}

func (s *Server) handleAddHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	name := r.FormValue("name")
	addr := r.FormValue("address")
	hcurl := r.FormValue("hcurl")
	if name == "" || addr == "" {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("name and address required"))
		return
	}
	if err := s.st.AddHost(name, addr, hcurl); err != nil {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	// Return refreshed hosts grid
	data := struct{ Hosts []*state.HostStatus }{Hosts: s.st.Snapshot()}
	_ = s.tpl.ExecuteTemplate(w, "add_host_result.html", data)
}

func (s *Server) handleAddHostForm(w http.ResponseWriter, r *http.Request) {
	_ = s.tpl.ExecuteTemplate(w, "addhost_modal.html", nil)
}

func (s *Server) handleCloseModal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(""))
}

func (s *Server) handleHosts(w http.ResponseWriter, r *http.Request) {
	data := struct{ Hosts []*state.HostStatus }{Hosts: s.st.Snapshot()}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.tpl.ExecuteTemplate(w, "hosts.html", data)
}

func (s *Server) handleAddHostCheckRow(w http.ResponseWriter, r *http.Request) {
	typ := r.FormValue("type")
	url := r.FormValue("url")
	expectStr := r.FormValue("expect")
	expect := 200
	if expectStr != "" {
		if v, err := strconv.Atoi(expectStr); err == nil {
			expect = v
		}
	}
	data := map[string]any{"Type": typ, "URL": url, "Expect": expect}
	_ = s.tpl.ExecuteTemplate(w, "addhost_check_row.html", data)
}

func (s *Server) handleEditHostForm(w http.ResponseWriter, r *http.Request) {
	host := r.FormValue("host")
	hs, ok := s.st.GetHost(host)
	if !ok {
		w.WriteHeader(404)
		return
	}
	_ = s.tpl.ExecuteTemplate(w, "edithost_modal.html", hs)
}

func (s *Server) handleEditHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	old := r.FormValue("old_name")
	name := r.FormValue("name")
	addr := r.FormValue("address")
	hcurl := r.FormValue("hcurl")
	if name == "" || addr == "" {
		w.WriteHeader(400)
		return
	}
	if err := s.st.UpdateHost(old, name, addr, hcurl); err != nil {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	data := struct{ Hosts []*state.HostStatus }{Hosts: s.st.Snapshot()}
	_ = s.tpl.ExecuteTemplate(w, "add_host_result.html", data)
}

func (s *Server) handleDeleteHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	name := r.FormValue("name")
	if err := s.st.DeleteHost(name); err != nil {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	data := struct{ Hosts []*state.HostStatus }{Hosts: s.st.Snapshot()}
	_ = s.tpl.ExecuteTemplate(w, "add_host_result.html", data)
}

func (s *Server) handleCheckConfig(w http.ResponseWriter, r *http.Request) {
	typ := r.FormValue("type")
	_ = s.tpl.ExecuteTemplate(w, "check_config_fragment.html", map[string]string{"Type": typ})
}

func (s *Server) handleHCURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	host := r.FormValue("host")
	url := r.FormValue("url")
	action := r.FormValue("action")
	if action == "clear" {
		url = ""
	}
	log.Printf("HCURL update request: host=%q url=%q", host, url)
	s.st.SetHCURL(host, url)
	fmt.Fprint(w, hcurlSection(host, url))
}

func hcurlSection(host, url string) string {
	return fmt.Sprintf(`
	<div class="field has-addons">
	  <div class="control is-expanded">
	    <input class="input" type="text" name="url" placeholder="Healthchecks.io ping URL" value="%s">
	  </div>
	  <div class="control">
	    <button class="button is-link" hx-post="/hcurl" hx-include="closest .field" hx-vals='{"host":"%s"}' hx-target="#hc-%s" hx-swap="outerHTML">Save</button>
	  </div>
	  <div class="control">
	    <button class="button is-light is-danger" hx-post="/hcurl" hx-vals='{"host":"%s","action":"clear"}' hx-target="#hc-%s" hx-swap="outerHTML">Clear</button>
	  </div>
	</div>`, template.HTMLEscapeString(url), host, host, host, host)
}

func (s *Server) handleAddHTTPForm(w http.ResponseWriter, r *http.Request) {
	host := r.FormValue("host")
	_ = s.tpl.ExecuteTemplate(w, "addhttp_modal.html", map[string]string{"Host": host})
}

func (s *Server) handleAddHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	host := r.FormValue("host")
	url := r.FormValue("url")
	expectStr := r.FormValue("expect")
	expect := 200
	if expectStr != "" {
		if v, err := strconv.Atoi(expectStr); err == nil {
			expect = v
		}
	}
	if err := s.st.AddHTTPCheck(host, url, expect); err != nil {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	data := struct{ Hosts []*state.HostStatus }{Hosts: s.st.Snapshot()}
	_ = s.tpl.ExecuteTemplate(w, "add_host_result.html", data)
}

func (s *Server) handleEditAddCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	host := r.FormValue("host")
	typ := r.FormValue("type")
	url := r.FormValue("url")
	expectStr := r.FormValue("expect")
	expect := 200
	if expectStr != "" {
		if v, err := strconv.Atoi(expectStr); err == nil {
			expect = v
		}
	}
	var err error
	switch typ {
	case "ping":
		err = s.st.AddPingCheck(host)
	case "http":
		err = s.st.AddHTTPCheck(host, url, expect)
	default:
		err = fmt.Errorf("unknown type")
	}
	if err != nil {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	hs, _ := s.st.GetHost(host)
	_ = s.tpl.ExecuteTemplate(w, "edithost_modal.html", hs)
}

func (s *Server) handleEditDelCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	host := r.FormValue("host")
	idxStr := r.FormValue("idx")
	idx, _ := strconv.Atoi(idxStr)
	if err := s.st.RemoveCheck(host, idx); err != nil {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	hs, _ := s.st.GetHost(host)
	_ = s.tpl.ExecuteTemplate(w, "edithost_modal.html", hs)
}

func (s *Server) handleEditUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	host := r.FormValue("host")
	idxStr := r.FormValue("idx")
	idx, _ := strconv.Atoi(idxStr)
	url := r.FormValue("url")
	expectStr := r.FormValue("expect")
	expect := 200
	if expectStr != "" {
		if v, err := strconv.Atoi(expectStr); err == nil {
			expect = v
		}
	}
	if err := s.st.UpdateHTTPCheck(host, idx, url, expect); err != nil {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	hs, _ := s.st.GetHost(host)
	_ = s.tpl.ExecuteTemplate(w, "edithost_modal.html", hs)
}

func toggleButton(host string, idx int, enabled bool) string {
	if enabled {
		return fmt.Sprintf(`<button class="button is-small is-warning is-light" hx-post="/toggle" hx-vals='{"host":"%s","idx":"%d","enabled":"false"}' hx-target="this" hx-swap="outerHTML">Disable</button>`, host, idx)
	}
	return fmt.Sprintf(`<button class="button is-small is-success is-light" hx-post="/toggle" hx-vals='{"host":"%s","idx":"%d","enabled":"true"}' hx-target="this" hx-swap="outerHTML">Enable</button>`, host, idx)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
