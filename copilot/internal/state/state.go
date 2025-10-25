package state

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	"github.com/andrewsjg/simple-healthchecker/copilot/internal/checks"
	"github.com/andrewsjg/simple-healthchecker/copilot/internal/config"
)

type CheckStatus struct {
	Type      config.CheckType
	Enabled   bool
	OK        bool
	Message   string
	LatencyMS int64
	CheckedAt time.Time
	URL       string
	Expect    int
}

type HostStatus struct {
	Name    string
	Address string
	Checks  []CheckStatus
	HCURL   string
}

type State struct {
	mu         sync.RWMutex
	cfg        *config.Config
	hosts      map[string]*HostStatus // key: host name
	configPath string
}

func New(cfg *config.Config) *State {
	st := &State{cfg: cfg, hosts: make(map[string]*HostStatus)}
	for _, h := range cfg.Hosts {
		hs := &HostStatus{Name: h.Name, Address: h.Address, HCURL: h.HealthchecksPingURL}
		for _, c := range h.Checks {
			cs := CheckStatus{Type: c.Type, Enabled: c.Enabled}
			if c.Type == config.CheckHTTP {
				cs.URL = c.URL
				cs.Expect = c.Expect
			}
			hs.Checks = append(hs.Checks, cs)
		}
		st.hosts[h.Name] = hs
	}
	return st
}

func (s *State) Snapshot() []*HostStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// stable order: by cfg order
	order := make(map[string]int, len(s.cfg.Hosts))
	for i, h := range s.cfg.Hosts {
		order[h.Name] = i
	}
	out := make([]*HostStatus, 0, len(s.hosts))
	for _, v := range s.hosts {
		copy := *v
		copy.Checks = append([]CheckStatus(nil), v.Checks...)
		out = append(out, &copy)
	}
	sort.SliceStable(out, func(i, j int) bool { return order[out[i].Name] < order[out[j].Name] })
	return out
}

func (s *State) AddHost(name, address, hcurl string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.hosts[name]; exists {
		return fmt.Errorf("host exists")
	}
	hs := &HostStatus{Name: name, Address: address, HCURL: hcurl}
	hs.Checks = append(hs.Checks, CheckStatus{Type: config.CheckPing, Enabled: true})
	s.hosts[name] = hs
	// update cfg
	s.cfg.Hosts = append(s.cfg.Hosts, config.Host{
		Name: name, Address: address, HealthchecksPingURL: hcurl,
		Checks: []config.Check{{Type: config.CheckPing, Enabled: true}},
	})
	return s.saveConfigLocked()
}

func (s *State) GetHost(name string) (HostStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if hs, ok := s.hosts[name]; ok {
		copy := *hs
		copy.Checks = append([]CheckStatus(nil), hs.Checks...)
		return copy, true
	}
	return HostStatus{}, false
}

func (s *State) UpdateHost(oldName, newName, address, hcurl string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hs, ok := s.hosts[oldName]
	if !ok {
		return fmt.Errorf("host not found")
	}
	if newName != oldName {
		if _, exists := s.hosts[newName]; exists {
			return fmt.Errorf("host name already exists")
		}
		delete(s.hosts, oldName)
		hs.Name = newName
		s.hosts[newName] = hs
	} else {
		hs.Name = newName
	}
	hs.Address = address
	hs.HCURL = hcurl
	// update cfg
	for i := range s.cfg.Hosts {
		if s.cfg.Hosts[i].Name == oldName {
			s.cfg.Hosts[i].Name = newName
			s.cfg.Hosts[i].Address = address
			s.cfg.Hosts[i].HealthchecksPingURL = hcurl
			break
		}
	}
	return s.saveConfigLocked()
}

func (s *State) AddHTTPCheck(hostName, url string, expect int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hs, ok := s.hosts[hostName]
	if !ok {
		return fmt.Errorf("host not found")
	}
	// append to runtime
	hs.Checks = append(hs.Checks, CheckStatus{Type: config.CheckHTTP, Enabled: true, URL: url, Expect: expect})
	// append to cfg
	for i := range s.cfg.Hosts {
		if s.cfg.Hosts[i].Name == hostName {
			s.cfg.Hosts[i].Checks = append(s.cfg.Hosts[i].Checks, config.Check{Type: config.CheckHTTP, Enabled: true, URL: url, Expect: expect})
			break
		}
	}
	return s.saveConfigLocked()
}

func (s *State) DeleteHost(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.hosts[name]; !ok {
		return fmt.Errorf("host not found")
	}
	delete(s.hosts, name)
	// remove from cfg
	for i := range s.cfg.Hosts {
		if s.cfg.Hosts[i].Name == name {
			s.cfg.Hosts = append(s.cfg.Hosts[:i], s.cfg.Hosts[i+1:]...)
			break
		}
	}
	return s.saveConfigLocked()
}

func (s *State) AddPingCheck(hostName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hs, ok := s.hosts[hostName]
	if !ok {
		return fmt.Errorf("host not found")
	}
	hs.Checks = append(hs.Checks, CheckStatus{Type: config.CheckPing, Enabled: true})
	for i := range s.cfg.Hosts {
		if s.cfg.Hosts[i].Name == hostName {
			s.cfg.Hosts[i].Checks = append(s.cfg.Hosts[i].Checks, config.Check{Type: config.CheckPing, Enabled: true})
			break
		}
	}
	return s.saveConfigLocked()
}

func (s *State) RemoveCheck(hostName string, idx int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hs, ok := s.hosts[hostName]
	if !ok {
		return fmt.Errorf("host not found")
	}
	if idx < 0 || idx >= len(hs.Checks) {
		return fmt.Errorf("bad index")
	}
	hs.Checks = append(hs.Checks[:idx], hs.Checks[idx+1:]...)
	for i := range s.cfg.Hosts {
		if s.cfg.Hosts[i].Name == hostName {
			if idx < 0 || idx >= len(s.cfg.Hosts[i].Checks) {
				break
			}
			s.cfg.Hosts[i].Checks = append(s.cfg.Hosts[i].Checks[:idx], s.cfg.Hosts[i].Checks[idx+1:]...)
			break
		}
	}
	return s.saveConfigLocked()
}

func (s *State) UpdateHTTPCheck(hostName string, idx int, url string, expect int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hs, ok := s.hosts[hostName]
	if !ok {
		return fmt.Errorf("host not found")
	}
	if idx < 0 || idx >= len(hs.Checks) {
		return fmt.Errorf("bad index")
	}
	if hs.Checks[idx].Type != config.CheckHTTP {
		return fmt.Errorf("not http check")
	}
	hs.Checks[idx].URL = url
	hs.Checks[idx].Expect = expect
	for i := range s.cfg.Hosts {
		if s.cfg.Hosts[i].Name == hostName {
			if idx < 0 || idx >= len(s.cfg.Hosts[i].Checks) {
				break
			}
			s.cfg.Hosts[i].Checks[idx].URL = url
			s.cfg.Hosts[i].Checks[idx].Expect = expect
			break
		}
	}
	return s.saveConfigLocked()
}

func (s *State) Toggle(hostName string, idx int, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hs, ok := s.hosts[hostName]; ok {
		if idx >= 0 && idx < len(hs.Checks) {
			hs.Checks[idx].Enabled = enabled
		}
	}
}

func (s *State) SetConfigPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if abs, err := filepath.Abs(path); err == nil {
		s.configPath = abs
	} else {
		s.configPath = path
	}
}

func (s *State) SetHCURL(hostName, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hs, ok := s.hosts[hostName]; ok {
		hs.HCURL = url
		// also persist into cfg
		found := false
		for i := range s.cfg.Hosts {
			if s.cfg.Hosts[i].Name == hostName {
				s.cfg.Hosts[i].HealthchecksPingURL = url
				found = true
				break
			}
		}
		if !found {
			log.Printf("warning: host %q not found in cfg when saving HCURL", hostName)
		}
		if err := s.saveConfigLocked(); err != nil {
			log.Printf("persist config failed: %v", err)
		} else {
			log.Printf("persist config ok: %s", s.configPath)
		}
	} else {
		log.Printf("warning: host %q not found in state when setting HCURL", hostName)
	}
}

func (s *State) StartScheduler(interval time.Duration, stop <-chan struct{}) {
	go func() {
		// run immediately, then on each tick
		s.runOnce()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				fmt.Println("scheduler tick")
				s.runOnce()
			case <-stop:
				return
			}
		}
	}()
}

func (s *State) runOnce() {
	fmt.Println("running checks")
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, hs := range s.hosts {
		for i := range hs.Checks {
			c := &hs.Checks[i]
			if !c.Enabled {
				continue
			}
			switch c.Type {
			case config.CheckPing:
				res := checks.PingOnce(hs.Address, 2*time.Second)
				c.OK = res.OK
				c.CheckedAt = time.Now()
				if res.OK {
					c.Message = "pong"
					c.LatencyMS = res.Latency.Milliseconds()
					if hs.HCURL != "" {
						_ = notifyHealthchecksOK(hs.HCURL)
					}
				} else {
					if res.Err != nil {
						c.Message = res.Err.Error()
					} else {
						c.Message = "no reply"
					}
					c.LatencyMS = 0
					if hs.HCURL != "" {
						_ = notifyHealthchecksFail(hs.HCURL)
					}
				}
			case config.CheckHTTP:
				url := c.URL
				if url == "" {
					// fallback to http://address if URL not set
					url = "http://" + hs.Address
				}
				res := checks.HTTPGet(url, 5*time.Second)
				c.CheckedAt = time.Now()
				if res.Err != nil {
					c.OK = false
					c.Message = res.Err.Error()
					c.LatencyMS = 0
				} else {
					expect := c.Expect
					if expect == 0 {
						expect = 200
					}
					c.OK = (res.Code == expect)
					c.Message = fmt.Sprintf("status %d (expect %d)", res.Code, expect)
					c.LatencyMS = res.Latency.Milliseconds()
				}
			}
		}
	}
}

func (s *State) saveConfigLocked() error {
	if s.configPath == "" {
		return nil
	}
	// sync HC URLs from runtime state to cfg before writing
	for i := range s.cfg.Hosts {
		name := s.cfg.Hosts[i].Name
		if hs, ok := s.hosts[name]; ok {
			s.cfg.Hosts[i].HealthchecksPingURL = hs.HCURL
		}
	}
	ext := filepath.Ext(s.configPath)
	switch ext {
	case ".yaml", ".yml":
		b, err := yaml.Marshal(s.cfg)
		if err != nil {
			return err
		}
		tmp := s.configPath + ".tmp"
		if err := os.WriteFile(tmp, b, 0644); err != nil {
			return err
		}
		if err := os.Rename(tmp, s.configPath); err != nil {
			// fallback: write directly
			if werr := os.WriteFile(s.configPath, b, 0644); werr != nil {
				return werr
			}
		}
		log.Printf("saved config to %s", s.configPath)
		return nil
	case ".toml":
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(s.cfg); err != nil {
			return err
		}
		tmp := s.configPath + ".tmp"
		if err := os.WriteFile(tmp, buf.Bytes(), 0644); err != nil {
			return err
		}
		if err := os.Rename(tmp, s.configPath); err != nil {
			if werr := os.WriteFile(s.configPath, buf.Bytes(), 0644); werr != nil {
				return werr
			}
		}
		log.Printf("saved config to %s", s.configPath)
		return nil
	default:
		return nil
	}
}

func notifyHealthchecksFail(base string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	url := base
	if url != "" && url[len(url)-1] != '/' {
		url += "/fail"
	} else {
		url += "fail"
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func notifyHealthchecksOK(base string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, base, nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}
