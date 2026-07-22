package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/subme-app/subme/internal/cache"
	"github.com/subme-app/subme/internal/collector"
	"github.com/subme-app/subme/internal/config"
	"github.com/subme-app/subme/internal/db"
	"github.com/subme-app/subme/internal/notify"
	"gopkg.in/yaml.v3"
)

var webFS fs.FS

type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

type Server struct {
	db            *db.DB
	cache         *cache.Cache
	collectCfg    collector.Config
	notifier      *notify.Notifier
	settings      *config.SystemSettings
	logs          []LogEntry
	logMu         sync.Mutex
	logSubs       map[string]chan LogEntry
	logSubMu      sync.Mutex
	collectorsDir string
	minLogLevel   string
}

func New(database *db.DB, cacheDir string, settings *config.SystemSettings, collectorsDir string) (*Server, error) {
	c, err := cache.New(cacheDir)
	if err != nil {
		return nil, err
	}

	notifierCfg := &notify.Config{
		AppToken: settings.WxPusher.AppToken,
		UIDs:     settings.WxPusher.UIDs,
	}

	srv := &Server{
		db:            database,
		cache:         c,
		notifier:      notify.New(notifierCfg),
		settings:      settings,
		logs:          make([]LogEntry, 0, 1000),
		logSubs:       make(map[string]chan LogEntry),
		collectorsDir: collectorsDir,
		collectCfg: collector.Config{
			Proxy: settings.Proxy,
		},
		minLogLevel: LevelInfo,
	}

	settings.RefreshInterval = defaultRefreshInterval(srv.settings.RefreshInterval)
	return srv, nil
}

func (s *Server) Logf(level, format string, args ...interface{}) {
	s.addLog(level, fmt.Sprintf(format, args...))
}

func (s *Server) LogInfo(msg string) {
	s.addLog(LevelInfo, msg)
}

func (s *Server) LogWarn(msg string) {
	s.addLog(LevelWarn, msg)
}

func (s *Server) LogError(msg string) {
	s.addLog(LevelError, msg)
}

func (s *Server) SetLogLevel(level string) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	s.minLogLevel = level
}

func (s *Server) LogLevel() string {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	return s.minLogLevel
}

func (s *Server) addLog(level, msg string) {
	if !levelEnabled(s.minLogLevel, level) {
		return
	}
	entry := LogEntry{Time: time.Now(), Level: level, Message: msg}
	s.logMu.Lock()
	s.logs = append(s.logs, entry)
	if len(s.logs) > 1000 {
		s.logs = s.logs[len(s.logs)-1000:]
	}
	s.logMu.Unlock()

	s.logSubMu.Lock()
	for _, ch := range s.logSubs {
		select {
		case ch <- entry:
		default:
		}
	}
	s.logSubMu.Unlock()
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /sub/{clashName}", s.handleGetSubscription)

	mux.HandleFunc("GET /api/sub/{clashName}/content", s.handleGetSubscriptionContent)

	mux.HandleFunc("GET /api/dashboard", s.handleDashboard)

	mux.HandleFunc("GET /api/providers", s.handleListProviders)
	mux.HandleFunc("POST /api/providers", s.handleCreateProvider)
	mux.HandleFunc("GET /api/providers/{id}", s.handleGetProvider)
	mux.HandleFunc("PUT /api/providers/{id}", s.handleUpdateProvider)
	mux.HandleFunc("DELETE /api/providers/{id}", s.handleDeleteProvider)
	mux.HandleFunc("POST /api/providers/{id}/test", s.handleTestProvider)
	mux.HandleFunc("POST /api/providers/{id}/refresh", s.handleRefreshProvider)
	mux.HandleFunc("POST /api/refresh", s.handleRefreshAll)

	mux.HandleFunc("GET /api/logs", s.handleLogsSSE)

	mux.HandleFunc("POST /api/register", s.handleRegister)
	mux.HandleFunc("POST /api/login", s.handleLogin)

	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", s.handleUpdateSettings)

	mux.HandleFunc("GET /api/log-level", s.handleGetLogLevel)
	mux.HandleFunc("PUT /api/log-level", s.handleSetLogLevel)

	mux.HandleFunc("GET /api/configs", s.handleListCollectors)
	mux.HandleFunc("POST /api/configs/upload", s.handleUploadCollector)

	mux.HandleFunc("GET /api/test-sub", func(w http.ResponseWriter, r *http.Request) {
		sampleYAML := `port: 7890
socks-port: 7891
mode: rule
log-level: info

proxies:
  - name: "HK-01"
    type: ss
    server: 1.2.3.4
    port: 443
    cipher: chacha20-ietf-poly1305
    password: "test-password-123"
    udp: true
  - name: "JP-01"
    type: ss
    server: 5.6.7.8
    port: 443
    cipher: aes-256-gcm
    password: "test-password-456"
    udp: true

proxy-groups:
  - name: "Proxy"
    type: select
    proxies:
      - HK-01
      - JP-01

rules:
  - MATCH,Proxy
`
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write([]byte(sampleYAML))
	})

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})

	apiMux := corsMiddleware(authMiddleware(s, s.logMiddleware(mux)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/sub/") {
			apiMux.ServeHTTP(w, r)
			return
		}
		// SPA fallback: serve index.html for all non-file routes
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(webFS, name); err != nil {
			name = "index.html"
		}
		data, err := fs.ReadFile(webFS, name)
		if err != nil {
			name = "index.html"
			data, err = fs.ReadFile(webFS, name)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
		}
		ct := "text/plain"
		if strings.HasSuffix(name, ".html") {
			ct = "text/html; charset=utf-8"
		} else if strings.HasSuffix(name, ".css") {
			ct = "text/css; charset=utf-8"
		} else if strings.HasSuffix(name, ".js") {
			ct = "application/javascript"
		} else if strings.HasSuffix(name, ".json") {
			ct = "application/json"
		} else if strings.HasSuffix(name, ".svg") {
			ct = "image/svg+xml"
		} else if strings.HasSuffix(name, ".png") {
			ct = "image/png"
		} else if strings.HasSuffix(name, ".ico") {
			ct = "image/x-icon"
		}
		w.Header().Set("Content-Type", ct)
		w.Write(data)
	})
}

func (s *Server) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	clashName := r.PathValue("clashName")
	s.addLog(LevelDebug, fmt.Sprintf("subscription request: clash_name=%s remote=%s", clashName, r.RemoteAddr))
	entry, err := s.cache.Get(clashName)
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("subscription not found: %s", clashName))
		http.Error(w, "subscription not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(entry.YAML)
}

func (s *Server) handleGetSubscriptionContent(w http.ResponseWriter, r *http.Request) {
	clashName := r.PathValue("clashName")
	s.addLog(LevelDebug, fmt.Sprintf("subscription content request: %s", clashName))
	entry, err := s.cache.Get(clashName)
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("subscription content not found: %s", clashName))
		writeJSON(w, map[string]string{"error": "subscription not found"})
		return
	}
	writeJSON(w, map[string]interface{}{
		"clash_name": clashName,
		"yaml":       string(entry.YAML),
		"last_fetch": entry.LastFetch.Format("2006-01-02 15:04:05"),
	})
}

func countProxies(yamlData []byte) int {
	var doc struct {
		Proxies []interface{} `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(yamlData, &doc); err != nil {
		return 0
	}
	return len(doc.Proxies)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.addLog(LevelDebug, "dashboard request")
	providers, err := s.db.ListProviders()
	if err != nil {
		s.addLog(LevelError, fmt.Sprintf("dashboard list providers: %v", err))
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	cachedNames, _ := s.cache.List()
	cacheMap := make(map[string]bool)
	for _, n := range cachedNames {
		cacheMap[n] = true
	}

	cards := make([]map[string]interface{}, 0)
	totalProxies := 0
	for _, p := range providers {
		card := map[string]interface{}{
			"id":           p.ID,
			"clash_name":   p.ClashName,
			"panel_url":    p.PanelURL,
			"collector":    p.CollectorName,
			"has_cache":    cacheMap[p.ClashName],
			"last_fetch":   nil,
			"proxy_count":  0,
		}
		if cacheMap[p.ClashName] {
			if entry, err := s.cache.Get(p.ClashName); err == nil {
				card["last_fetch"] = entry.LastFetch.Format("2006-01-02 15:04:05")
				card["proxy_count"] = countProxies(entry.YAML)
				totalProxies += card["proxy_count"].(int)
			}
		}
		cards = append(cards, card)
	}

	s.addLog(LevelDebug, fmt.Sprintf("dashboard result: %d providers, %d cached, %d proxies", len(providers), len(cacheMap), totalProxies))
	writeJSON(w, map[string]interface{}{
		"providers":     cards,
		"total":         len(providers),
		"cached":        len(cacheMap),
		"total_proxies": totalProxies,
	})
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	s.addLog(LevelDebug, "list providers")
	providers, err := s.db.ListProviders()
	if err != nil {
		s.addLog(LevelError, fmt.Sprintf("list providers: %v", err))
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if providers == nil {
		providers = []db.Provider{}
	}
	s.addLog(LevelDebug, fmt.Sprintf("found %d providers", len(providers)))
	writeJSON(w, providers)
}

func (s *Server) handleCreateProvider(w http.ResponseWriter, r *http.Request) {
	var p db.Provider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("create provider invalid body: %v", err))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if p.ClashName == "" {
		s.addLog(LevelWarn, "create provider missing clash_name")
		http.Error(w, "clash_name is required", http.StatusBadRequest)
		return
	}
	if p.CollectorName == "" {
		s.addLog(LevelWarn, fmt.Sprintf("create provider %s missing collector_name", p.ClashName))
		http.Error(w, "collector_name is required", http.StatusBadRequest)
		return
	}
	scriptPath := filepath.Join(s.collectorsDir, p.CollectorName, "collector.js")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		s.addLog(LevelWarn, fmt.Sprintf("create provider %s collector.js not found: %s", p.ClashName, p.CollectorName))
		http.Error(w, fmt.Sprintf("collector.js not found for %s", p.CollectorName), http.StatusBadRequest)
		return
	}
	if p.Interval <= 0 {
		p.Interval = 3600
	}

	cfgPath := s.buildConfigPath(p.ClashName)
	p.ConfigPath = cfgPath

	id, err := s.db.CreateProvider(&p)
	if err != nil {
		s.addLog(LevelError, fmt.Sprintf("create provider %s db error: %v", p.ClashName, err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.writeCollectorConfig(p); err != nil {
		s.addLog(LevelError, fmt.Sprintf("failed to write collector config for %s: %v", p.ClashName, err))
	}

	p.ID = id
	s.addLog(LevelInfo, fmt.Sprintf("provider created: %s (collector=%s interval=%d)", p.ClashName, p.CollectorName, p.Interval))
	writeJSON(w, p)
}

func (s *Server) handleGetProvider(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("get provider invalid id: %s", r.PathValue("id")))
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	p, err := s.db.GetProvider(id)
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("get provider %d not found", id))
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}
	s.addLog(LevelDebug, fmt.Sprintf("get provider %d: %s", id, p.ClashName))
	writeJSON(w, p)
}

func (s *Server) handleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("update provider invalid id: %s", r.PathValue("id")))
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var p db.Provider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("update provider %d invalid body: %v", id, err))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	p.ID = id
	old, _ := s.db.GetProvider(id)
	if err := s.db.UpdateProvider(&p); err != nil {
		s.addLog(LevelError, fmt.Sprintf("update provider %d db error: %v", id, err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.writeCollectorConfig(p); err != nil {
		s.addLog(LevelError, fmt.Sprintf("failed to update collector config for %s: %v", p.ClashName, err))
	}

	if old != nil {
		s.addLog(LevelInfo, fmt.Sprintf("provider updated: %s (panel_url: %q -> %q, interval: %d -> %d)",
			p.ClashName, old.PanelURL, p.PanelURL, old.Interval, p.Interval))
	}
	writeJSON(w, p)
}

func (s *Server) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("delete provider invalid id: %s", r.PathValue("id")))
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	p, err := s.db.GetProvider(id)
	if err == nil {
		s.addLog(LevelDebug, fmt.Sprintf("delete provider cleaning up: %s", p.ClashName))
		s.cache.Delete(p.ClashName)
		os.RemoveAll(s.providerDir(p.ClashName))
	}
	if err := s.db.DeleteProvider(id); err != nil {
		s.addLog(LevelError, fmt.Sprintf("delete provider %d db error: %v", id, err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if p != nil {
		s.addLog(LevelInfo, fmt.Sprintf("provider deleted: id=%d name=%s", id, p.ClashName))
	} else {
		s.addLog(LevelInfo, fmt.Sprintf("provider deleted: id=%d (unknown)", id))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTestProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CollectorName string `json:"collector_name"`
		PanelURL    string `json:"panel_url"`
		LandingPage string `json:"landing_page"`
		Username    string `json:"username"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.CollectorName == "" {
		http.Error(w, "collector_name is required", http.StatusBadRequest)
		return
	}

	collectorDir := filepath.Join(s.collectorsDir, body.CollectorName)
	absCollectorDir, _ := filepath.Abs(collectorDir)
	scriptPath := filepath.Join(absCollectorDir, "collector.js")

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("collector.js not found for %s", body.CollectorName),
		})
		return
	}

	s.addLog(LevelInfo, fmt.Sprintf("testing connection: collector=%s username=%s", body.CollectorName, body.Username))
	s.addLog(LevelDebug, fmt.Sprintf("test config created at %s", filepath.Join(absCollectorDir, "_test_config.yaml")))

	tmpConfigPath := filepath.Join(absCollectorDir, "_test_config.yaml")
	pc := &config.ProviderConfig{
		ClashName:   "_test",
		Interval:    3600,
		PanelURL:    body.PanelURL,
		LandingPage: body.LandingPage,
		Username:    body.Username,
		Password:    body.Password,
	}
	if err := config.SaveProviderConfig(tmpConfigPath, pc); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	defer os.Remove(tmpConfigPath)

	cfg := collector.Config{
		ScriptDir:   absCollectorDir,
		ConfigPath: tmpConfigPath,
		Proxy:      s.collectCfg.Proxy,
	}

	result, err := collector.Run(r.Context(), cfg)
	if err != nil {
		s.addLog(LevelError, fmt.Sprintf("test connection error: %v", err))
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	if !result.Success {
		s.addLog(LevelError, fmt.Sprintf("test connection failed: %s | stderr: %s", result.Error, result.Stderr))
	} else {
		s.addLog(LevelInfo, fmt.Sprintf("test connection success: panel=%s", result.PanelURL))
	}

	writeJSON(w, result)
}

func (s *Server) handleRefreshProvider(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("refresh provider invalid id: %s", r.PathValue("id")))
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	p, err := s.db.GetProvider(id)
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("refresh provider %d not found", id))
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}

	s.addLog(LevelInfo, fmt.Sprintf("manual refresh triggered: %s (id=%d)", p.ClashName, id))
	go s.refreshProvider(p)
	writeJSON(w, map[string]string{"status": "started"})
}

func (s *Server) handleRefreshAll(w http.ResponseWriter, r *http.Request) {
	s.addLog(LevelInfo, "manual refresh all triggered")
	go s.refreshAll()
	writeJSON(w, map[string]string{"status": "started"})
}

func (s *Server) handleLogsSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush() // send headers immediately so EventSource.onopen fires

	// Send buffered log history first
	s.logMu.Lock()
	for _, entry := range s.logs {
		data, _ := json.Marshal(entry)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}
	s.logMu.Unlock()
	flusher.Flush()

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	ch := make(chan LogEntry, 64)
	s.logSubMu.Lock()
	s.logSubs[id] = ch
	s.logSubMu.Unlock()

	defer func() {
		s.logSubMu.Lock()
		delete(s.logSubs, id)
		s.logSubMu.Unlock()
	}()

	ctx := r.Context()
	for {
		select {
		case entry := <-ch:
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) handleGetLogLevel(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"level": s.LogLevel()})
}

func (s *Server) handleSetLogLevel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Level string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if _, ok := levelOrder[body.Level]; !ok {
		http.Error(w, fmt.Sprintf("invalid level: %s (valid: debug, info, warn, error)", body.Level), http.StatusBadRequest)
		return
	}
	prev := s.LogLevel()
	s.SetLogLevel(body.Level)
	s.addLog(LevelInfo, fmt.Sprintf("log level changed: %s -> %s", prev, body.Level))
	writeJSON(w, map[string]string{"level": body.Level})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	hasUsers, err := s.db.HasUsers()
	if err != nil {
		s.addLog(LevelError, fmt.Sprintf("register check users: %v", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if hasUsers {
		s.addLog(LevelWarn, "register blocked: admin already exists")
		http.Error(w, "admin already exists", http.StatusForbidden)
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.addLog(LevelWarn, "register invalid body")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if body.Username == "" || body.Password == "" {
		s.addLog(LevelWarn, "register missing username or password")
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}

	hash := hashPassword(body.Password)
	if err := s.db.RegisterUser(body.Username, hash); err != nil {
		s.addLog(LevelError, fmt.Sprintf("register failed: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.addLog(LevelInfo, fmt.Sprintf("admin registered: %s", body.Username))
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.addLog(LevelWarn, "login invalid body")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	s.addLog(LevelDebug, fmt.Sprintf("login attempt: %s", body.Username))
	user, err := s.db.GetUser(body.Username)
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("login failed: user not found: %s", body.Username))
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if user.PasswordHash != hashPassword(body.Password) {
		s.addLog(LevelWarn, fmt.Sprintf("login failed: wrong password for %s", body.Username))
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token := hashPassword(body.Username + ":" + time.Now().String())
	s.addLog(LevelInfo, fmt.Sprintf("login success: %s", body.Username))
	writeJSON(w, map[string]string{"token": token})
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	s.addLog(LevelDebug, "get settings")
	writeJSON(w, s.settings)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var newSettings config.SystemSettings
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("update settings invalid body: %v", err))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	old := s.settings
	s.db.SetSetting("refresh_interval", fmt.Sprintf("%d", newSettings.RefreshInterval))
	s.db.SetSetting("proxy", newSettings.Proxy)
	s.db.SetSetting("wxpusher_app_token", newSettings.WxPusher.AppToken)
	s.db.SetSetting("wxpusher_uids", fmt.Sprintf("%v", newSettings.WxPusher.UIDs))

	s.settings = &newSettings
	s.collectCfg.Proxy = newSettings.Proxy
	s.notifier = notify.New(&notify.Config{
		AppToken: newSettings.WxPusher.AppToken,
		UIDs:     newSettings.WxPusher.UIDs,
	})

	s.addLog(LevelInfo, fmt.Sprintf("settings updated: proxy=%q refresh_interval=%d notify=%v",
		newSettings.Proxy, newSettings.RefreshInterval, newSettings.NotifyOn))
	if old.Proxy != newSettings.Proxy {
		s.addLog(LevelDebug, fmt.Sprintf("proxy changed: %q -> %q", old.Proxy, newSettings.Proxy))
	}
	if old.RefreshInterval != newSettings.RefreshInterval {
		s.addLog(LevelDebug, fmt.Sprintf("refresh interval changed: %d -> %d", old.RefreshInterval, newSettings.RefreshInterval))
	}

	writeJSON(w, s.settings)
}

func (s *Server) handleListCollectors(w http.ResponseWriter, r *http.Request) {
	entries, err := listCollectorDirs(s.collectorsDir)
	if err != nil {
		writeJSON(w, []string{})
		return
	}
	writeJSON(w, entries)
}

func (s *Server) handleUploadCollector(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("upload collector parse form: %v", err))
		writeJSON(w, map[string]string{"error": "invalid form data"})
		return
	}

	name := r.FormValue("name")
	if name == "" {
		s.addLog(LevelWarn, "upload collector missing name")
		writeJSON(w, map[string]string{"error": "name is required"})
		return
	}
	validName := regexp.MustCompile(`^[\p{L}0-9_.-]+$`)
	if !validName.MatchString(name) {
		s.addLog(LevelWarn, fmt.Sprintf("upload collector invalid name: %s", name))
		writeJSON(w, map[string]string{"error": "name must be alphanumeric, hyphens, or underscores only"})
		return
	}

	collectorFile, _, err := r.FormFile("collector")
	if err != nil {
		s.addLog(LevelWarn, fmt.Sprintf("upload collector %s missing file", name))
		writeJSON(w, map[string]string{"error": "collector.js file is required"})
		return
	}
	defer collectorFile.Close()

	collectorData, err := io.ReadAll(collectorFile)
	if err != nil {
		s.addLog(LevelError, fmt.Sprintf("upload collector %s read file: %v", name, err))
		writeJSON(w, map[string]string{"error": "failed to read collector.js"})
		return
	}

	dir := filepath.Join(s.collectorsDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.addLog(LevelError, fmt.Sprintf("upload collector %s mkdir: %v", name, err))
		writeJSON(w, map[string]string{"error": "failed to create directory"})
		return
	}

	if err := os.WriteFile(filepath.Join(dir, "collector.js"), collectorData, 0644); err != nil {
		s.addLog(LevelError, fmt.Sprintf("upload collector %s write file: %v", name, err))
		writeJSON(w, map[string]string{"error": "failed to write collector.js"})
		return
	}

	configFile, _, err := r.FormFile("config")
	if err == nil {
		defer configFile.Close()
		configData, _ := io.ReadAll(configFile)
		if len(configData) > 0 {
			os.WriteFile(filepath.Join(dir, "config.yaml"), configData, 0644)
			s.addLog(LevelDebug, fmt.Sprintf("upload collector %s: config file included (%d bytes)", name, len(configData)))
		}
	} else {
		defaultConfig := fmt.Sprintf(`clash_name: %s
interval: 3600
panel_url: ""
landing_page: ""
username: ""
password: ""
`, name)
		os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(defaultConfig), 0644)
		s.addLog(LevelDebug, fmt.Sprintf("upload collector %s: default config created", name))
	}

	s.addLog(LevelInfo, fmt.Sprintf("collector uploaded: %s (%d bytes)", name, len(collectorData)))
	writeJSON(w, map[string]string{"status": "ok", "name": name})
}

func (s *Server) refreshProvider(p *db.Provider) {
	start := time.Now()
	name := p.ClashName
	s.addLog(LevelInfo, fmt.Sprintf("[开始] 刷新: %s (collector=%s)", name, p.CollectorName))

	if p.ConfigPath == "" {
		s.addLog(LevelError, fmt.Sprintf("[结束] 刷新失败: %s - no config path", name))
		return
	}

	collectorDir, _ := filepath.Abs(filepath.Join(s.collectorsDir, p.CollectorName))
	configPath, _ := filepath.Abs(p.ConfigPath)
	scriptPath := filepath.Join(collectorDir, "collector.js")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		s.addLog(LevelError, fmt.Sprintf("[结束] 刷新失败: %s - collector.js not found", name))
		return
	}

	s.addLog(LevelDebug, fmt.Sprintf("collector config: dir=%s config=%s proxy=%s", collectorDir, configPath, s.collectCfg.Proxy))

	cfg := collector.Config{
		ScriptDir:   collectorDir,
		ConfigPath: configPath,
		Proxy:      s.collectCfg.Proxy,
	}

	collectorStart := time.Now()
	result, err := collector.Run(context.Background(), cfg)
	collectorDuration := time.Since(collectorStart)
	if err != nil {
		s.addLog(LevelError, fmt.Sprintf("collector error for %s (%v): %v", name, collectorDuration, err))
		s.addLog(LevelInfo, fmt.Sprintf("[结束] 刷新失败: %s (%v)", name, time.Since(start)))
		s.notifyIfNeeded(name, false, fmt.Sprintf("collector error: %v", err))
		return
	}

	if !result.Success {
		s.addLog(LevelError, fmt.Sprintf("collector failed for %s (%v): %s", name, collectorDuration, result.Error))
		s.addLog(LevelInfo, fmt.Sprintf("[结束] 刷新失败: %s (%v)", name, time.Since(start)))
		s.notifyIfNeeded(name, false, result.Error)
		return
	}

	s.addLog(LevelDebug, fmt.Sprintf("collector succeeded for %s (%v) via_proxy=%v", name, collectorDuration, result.ViaProxy))

	if result.PanelURL != "" && result.PanelURL != p.PanelURL {
		s.addLog(LevelInfo, fmt.Sprintf("updating panel_url for %s: %s", name, result.PanelURL))
		p.PanelURL = result.PanelURL
		if err := s.db.UpdateProvider(p); err != nil {
			s.addLog(LevelError, fmt.Sprintf("update db panel_url for %s: %v", name, err))
		}
		if result.UpdateConfig != nil {
			if err := config.UpdateProviderConfig(p.ConfigPath, result.UpdateConfig); err != nil {
				s.addLog(LevelError, fmt.Sprintf("update config failed for %s: %v", name, err))
			}
		}
	} else if result.UpdateConfig != nil {
		s.addLog(LevelDebug, fmt.Sprintf("updating config for %s", name))
		if err := config.UpdateProviderConfig(p.ConfigPath, result.UpdateConfig); err != nil {
			s.addLog(LevelError, fmt.Sprintf("update config failed for %s: %v", name, err))
		}
	}

	s.addLog(LevelDebug, fmt.Sprintf("fetching subscription: %s", maskURL(result.SubscriptionURL)))
	fetchStart := time.Now()
	subscriptionYAML, err := fetchSubscription(result.SubscriptionURL, s.collectCfg.Proxy)
	if err != nil {
		s.addLog(LevelError, fmt.Sprintf("fetch subscription failed for %s (%v): %v", name, time.Since(fetchStart), err))
		s.addLog(LevelInfo, fmt.Sprintf("[结束] 刷新失败: %s (%v)", name, time.Since(start)))
		s.notifyIfNeeded(name, false, fmt.Sprintf("fetch subscription failed: %v", err))
		return
	}
	s.addLog(LevelDebug, fmt.Sprintf("subscription fetched (%d bytes) in %v", len(subscriptionYAML), time.Since(fetchStart)))

	if err := s.cache.Set(name, subscriptionYAML); err != nil {
		s.addLog(LevelError, fmt.Sprintf("cache write failed for %s: %v", name, err))
		s.addLog(LevelInfo, fmt.Sprintf("[结束] 刷新失败: %s (%v)", name, time.Since(start)))
		return
	}

	proxyCount := countProxies(subscriptionYAML)
	s.addLog(LevelInfo, fmt.Sprintf("[结束] 刷新完成: %s (%d bytes, %d proxies, %v)", name, len(subscriptionYAML), proxyCount, time.Since(start)))
}

func (s *Server) refreshAll() {
	start := time.Now()
	s.addLog(LevelInfo, "[开始] 刷新全部 Provider")
	providers, err := s.db.ListProviders()
	if err != nil {
		s.addLog(LevelError, fmt.Sprintf("list providers: %v", err))
		s.addLog(LevelInfo, fmt.Sprintf("[结束] 刷新全部失败 (%v)", time.Since(start)))
		return
	}
	s.addLog(LevelDebug, fmt.Sprintf("refresh all: %d providers to refresh", len(providers)))

	var wg sync.WaitGroup
	for _, p := range providers {
		wg.Add(1)
		go func(prov db.Provider) {
			defer wg.Done()
			s.refreshProvider(&prov)
		}(p)
	}
	wg.Wait()
	s.addLog(LevelInfo, fmt.Sprintf("[结束] 刷新全部完成 (%v)", time.Since(start)))
}

func (s *Server) RefreshAllSync() {
	s.refreshAll()
}

func (s *Server) notifyIfNeeded(provider string, success bool, message string) {
	if success {
		return
	}
	if s.settings.NotifyOn.CollectFailure || s.settings.NotifyOn.RefreshFailure {
		event := notify.Event{
			Provider: provider,
			Success:  success,
			Message:  message,
		}
		s.addLog(LevelDebug, fmt.Sprintf("sending notification for %s", provider))
		if err := s.notifier.Send(event); err != nil {
			s.addLog(LevelWarn, fmt.Sprintf("notification send error: %v", err))
		}
	}
}

func (s *Server) buildConfigPath(clashName string) string {
	return s.providerDir(clashName) + string(os.PathSeparator) + "config.yaml"
}

func (s *Server) providerDir(clashName string) string {
	return s.collectorsDir + string(os.PathSeparator) + clashName
}

func (s *Server) writeCollectorConfig(p db.Provider) error {
	dir := s.providerDir(p.ClashName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	pc := &config.ProviderConfig{
		ClashName:   p.ClashName,
		Interval:    p.Interval,
		PanelURL:    p.PanelURL,
		LandingPage: p.LandingPage,
		Username:    p.Username,
		Password:    p.Password,
	}
	return config.SaveProviderConfig(s.buildConfigPath(p.ClashName), pc)
}

func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		if duration > time.Second {
			s.addLog(LevelDebug, fmt.Sprintf("[%s] %s %s (%v)", r.Method, r.URL.Path, r.RemoteAddr, duration))
		} else {
			s.addLog(LevelDebug, fmt.Sprintf("[%s] %s %s (%v)", r.Method, r.URL.Path, r.RemoteAddr, duration))
		}
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func hashPassword(pw string) string {
	h := sha256.Sum256([]byte(pw))
	return hex.EncodeToString(h[:])
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(s *Server, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		publicPaths := []string{"/api/register", "/api/login", "/api/health"}
		for _, p := range publicPaths {
			if path == p {
				next.ServeHTTP(w, r)
				return
			}
		}

		if strings.HasPrefix(path, "/sub/") {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(path, "/api/") {
			hasUsers, err := s.db.HasUsers()
			if err != nil || !hasUsers {
				s.addLog(LevelWarn, fmt.Sprintf("auth rejected: %s (no admin registered)", path))
				http.Error(w, "no admin registered", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func listCollectorDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		scriptPath := filepath.Join(dir, e.Name(), "collector.js")
		if _, err := os.Stat(scriptPath); err == nil {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs, nil
}

func maskURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	if len(u.Host) > 8 {
		return u.Scheme + "://" + u.Host[:4] + "..." + u.Host[len(u.Host)-4:] + u.Path
	}
	return u.Scheme + "://" + u.Host + u.Path
}

func fetchSubscription(rawURL, proxy string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "SubMe/1.0")

	if proxy != "" {
		proxyURL, parseErr := url.Parse(proxy)
		if parseErr == nil {
			transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
			client.Transport = transport
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return body, nil
}

func defaultRefreshInterval(interval int) int {
	if interval <= 0 {
		return 3600
	}
	return interval
}
