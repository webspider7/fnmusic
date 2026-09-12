package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fn-lx-player/pkg/config"
)

type SourceItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	URL         string `json:"url"`
	IsCustom    bool   `json:"is_custom"`
	IsPreset    bool   `json:"is_preset"`
	Status      string `json:"status"`
	Size        int64  `json:"size"`
}

type Manager struct {
	cfgMgr *config.ConfigManager
}

func NewManager(cfgMgr *config.ConfigManager, _ interface{}, _ string) *Manager {
	return &Manager{
		cfgMgr: cfgMgr,
	}
}

func (m *Manager) HandleListSources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	allSources := make([]SourceItem, 0)

	// Read custom sources from config (All sources are user-imported, 0 built-in)
	cfg := m.cfgMgr.Get()
	for _, cs := range cfg.CustomSources {
		allSources = append(allSources, SourceItem{
			ID:          cs.ID,
			Name:        cs.Name,
			Description: cs.Description,
			Version:     cs.Version,
			Author:      cs.Author,
			IsCustom:    true,
			IsPreset:    false,
			Status:      "ready",
			Size:        int64(len(cs.Script)),
		})
	}

	// Ensure active source id is valid
	if cfg.ActiveSourceID != "" {
		exists := false
		for _, s := range allSources {
			if s.ID == cfg.ActiveSourceID {
				exists = true
				break
			}
		}
		if !exists {
			if len(allSources) > 0 {
				cfg.ActiveSourceID = allSources[0].ID
			} else {
				cfg.ActiveSourceID = ""
			}
			_ = m.cfgMgr.Update(cfg)
		}
	} else if len(allSources) > 0 {
		cfg.ActiveSourceID = allSources[0].ID
		_ = m.cfgMgr.Update(cfg)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":             200,
		"message":          "ok",
		"active_source_id": cfg.ActiveSourceID,
		"sources":          allSources,
		"data": map[string]interface{}{
			"sources":          allSources,
			"active_id":        cfg.ActiveSourceID,
			"active_source_id": cfg.ActiveSourceID,
		},
	})
}

func (m *Manager) HandleGetScript(w http.ResponseWriter, r *http.Request) {
	sourceID := strings.TrimSpace(r.URL.Query().Get("id"))
	if sourceID == "" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) >= 3 && parts[1] == "sources" {
			sourceID = parts[2]
			if sourceID == "script" && len(parts) >= 4 {
				sourceID = parts[3]
			}
		}
	}
	if sourceID == "" {
		http.Error(w, "id parameter required", http.StatusBadRequest)
		return
	}

	// Remove potential .js suffix
	sourceID = strings.TrimSuffix(sourceID, ".js")

	// 1. Check custom sources
	var scriptContent string
	cfg := m.cfgMgr.Get()
	for _, cs := range cfg.CustomSources {
		if cs.ID == sourceID {
			scriptContent = cs.Script
			break
		}
	}

	if scriptContent == "" {
		http.Error(w, "Source script not found", http.StatusNotFound)
		return
	}

	// If client expects JSON or format != raw
	format := r.URL.Query().Get("format")
	if format == "raw" || strings.Contains(r.Header.Get("Accept"), "text/javascript") {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write([]byte(scriptContent))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "ok",
		"data": map[string]string{
			"id":     sourceID,
			"script": scriptContent,
		},
		"script": scriptContent,
	})
}

func (m *Manager) HandleSetActive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SourceID string `json:"source_id"`
		ID       string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	targetID := req.SourceID
	if targetID == "" {
		targetID = req.ID
	}
	if targetID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "source_id required"})
		return
	}

	cfg := m.cfgMgr.Get()
	cfg.ActiveSourceID = targetID
	_ = m.cfgMgr.Update(cfg)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":             200,
		"status":           "ok",
		"active_source_id": cfg.ActiveSourceID,
		"data": map[string]string{
			"active_id": cfg.ActiveSourceID,
		},
	})
}

func (m *Manager) HandleAddCustom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Author      string `json:"author"`
		Script      string `json:"script"`
		Filename    string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Script) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "message": "script content required"})
		return
	}

	csID := fmt.Sprintf("custom_%d", time.Now().Unix())
	csName := req.Name
	if csName == "" {
		if req.Filename != "" {
			csName = strings.TrimSuffix(req.Filename, ".js")
		} else {
			csName = "自定义音源"
		}
	}

	cs := config.CustomSource{
		ID:          csID,
		Name:        csName,
		Description: req.Description,
		Version:     req.Version,
		Author:      req.Author,
		Script:      req.Script,
		CreatedAt:   time.Now().Unix(),
	}

	cfg := m.cfgMgr.Get()
	cfg.CustomSources = append(cfg.CustomSources, cs)
	_ = m.cfgMgr.Update(cfg)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"status":  "ok",
		"message": "Custom source added successfully",
		"data":    cs,
	})
}

func (m *Manager) HandleImportURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "message": "URL required"})
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(req.URL)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 502, "message": "Failed to fetch source script: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "message": "Empty response from URL"})
		return
	}

	csID := fmt.Sprintf("custom_%d", time.Now().Unix())
	csName := req.Name
	if csName == "" {
		csName = "导入音源"
	}

	cs := config.CustomSource{
		ID:        csID,
		Name:      csName,
		Script:    string(body),
		CreatedAt: time.Now().Unix(),
	}

	cfg := m.cfgMgr.Get()
	cfg.CustomSources = append(cfg.CustomSources, cs)
	_ = m.cfgMgr.Update(cfg)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"status":  "ok",
		"message": "Source imported successfully",
		"data":    cs,
	})
}

func (m *Manager) HandleDeleteCustom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var targetID string
	if r.Method == http.MethodDelete || r.Method == http.MethodPost {
		var req struct {
			ID       string `json:"id"`
			SourceID string `json:"source_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		targetID = req.ID
		if targetID == "" {
			targetID = req.SourceID
		}
	}
	if targetID == "" {
		targetID = r.URL.Query().Get("id")
	}

	if targetID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "message": "id required"})
		return
	}

	cfg := m.cfgMgr.Get()
	filtered := make([]config.CustomSource, 0)
	found := false
	for _, cs := range cfg.CustomSources {
		if cs.ID == targetID {
			found = true
			continue
		}
		filtered = append(filtered, cs)
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 404, "message": "Custom source not found"})
		return
	}

	cfg.CustomSources = filtered
	if cfg.ActiveSourceID == targetID {
		if len(filtered) > 0 {
			cfg.ActiveSourceID = filtered[0].ID
		} else {
			cfg.ActiveSourceID = ""
		}
	}
	_ = m.cfgMgr.Update(cfg)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"status":  "ok",
		"message": "Custom source deleted",
	})
}

func (m *Manager) HandleSourcesRest(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) >= 3 && parts[1] == "sources" {
		sourceID := parts[2]
		if r.Method == http.MethodDelete {
			// Delete custom source
			cfg := m.cfgMgr.Get()
			filtered := make([]config.CustomSource, 0)
			for _, cs := range cfg.CustomSources {
				if cs.ID == sourceID {
					continue
				}
				filtered = append(filtered, cs)
			}
			cfg.CustomSources = filtered
			if cfg.ActiveSourceID == sourceID {
				if len(filtered) > 0 {
					cfg.ActiveSourceID = filtered[0].ID
				} else {
					cfg.ActiveSourceID = ""
				}
			}
			_ = m.cfgMgr.Update(cfg)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "status": "ok"})
			return
		} else if strings.HasSuffix(r.URL.Path, "/script") || (len(parts) == 4 && parts[3] == "script") {
			m.HandleGetScript(w, r)
			return
		}
	}
	m.HandleGetScript(w, r)
}


