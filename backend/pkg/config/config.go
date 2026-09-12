package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type CustomSource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Script      string `json:"script"`
	CreatedAt   int64  `json:"created_at"`
}

type AppConfig struct {
	Port           int            `json:"port"`
	DefaultNasDir  string         `json:"default_nas_dir"`
	DownloadDir    string         `json:"download_dir"`
	ActiveSourceID string         `json:"active_source_id"`
	Theme          string         `json:"theme"`
	VisualizerMode string         `json:"visualizer_mode"`
	PreferQuality  string         `json:"prefer_quality"`
	CustomSources  []CustomSource `json:"custom_sources"`
}

type ConfigManager struct {
	mu       sync.RWMutex
	filePath string
	config   AppConfig
}

func NewConfigManager(dataDir string, defaultPort int) (*ConfigManager, error) {
	if dataDir == "" {
		dataDir = "./data"
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		dataDir = "/tmp/fn-lx-player-data"
		_ = os.MkdirAll(dataDir, 0755)
	}

	defaultDir := "/vol1/Music"
	if _, err := os.Stat("/vol1"); err != nil {
		defaultDir = filepath.Join(dataDir, "music")
		_ = os.MkdirAll(defaultDir, 0755)
	}

	cm := &ConfigManager{
		filePath: filepath.Join(dataDir, "config.json"),
		config: AppConfig{
			Port:           defaultPort,
			DefaultNasDir:  defaultDir,
			DownloadDir:    defaultDir,
			ActiveSourceID: "",
			Theme:          "fresh-mint",
			VisualizerMode: "bars",
			PreferQuality:  "320k",
			CustomSources:  make([]CustomSource, 0),
		},
	}

	cm.load()
	return cm, nil
}

func (cm *ConfigManager) load() {
	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		return
	}
	var c AppConfig
	if err := json.Unmarshal(data, &c); err == nil {
		if c.DefaultNasDir != "" {
			cm.config.DefaultNasDir = c.DefaultNasDir
		}
		if c.DownloadDir != "" {
			cm.config.DownloadDir = c.DownloadDir
		} else if cm.config.DefaultNasDir != "" {
			cm.config.DownloadDir = cm.config.DefaultNasDir
		}
		if c.ActiveSourceID != "" {
			cm.config.ActiveSourceID = c.ActiveSourceID
		}
		if c.Theme != "" {
			cm.config.Theme = c.Theme
		}
		if c.VisualizerMode != "" {
			cm.config.VisualizerMode = c.VisualizerMode
		}
		if c.PreferQuality != "" {
			cm.config.PreferQuality = c.PreferQuality
		}
		if c.CustomSources != nil {
			cm.config.CustomSources = c.CustomSources
		}
	}
}

func (cm *ConfigManager) Get() AppConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func (cm *ConfigManager) Update(c AppConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if c.DefaultNasDir != "" {
		cm.config.DefaultNasDir = c.DefaultNasDir
	}
	if c.DownloadDir != "" {
		cm.config.DownloadDir = c.DownloadDir
	}
	if c.ActiveSourceID != "" {
		cm.config.ActiveSourceID = c.ActiveSourceID
	}
	if c.Theme != "" {
		cm.config.Theme = c.Theme
	}
	if c.VisualizerMode != "" {
		cm.config.VisualizerMode = c.VisualizerMode
	}
	if c.PreferQuality != "" {
		cm.config.PreferQuality = c.PreferQuality
	}
	if c.CustomSources != nil {
		cm.config.CustomSources = c.CustomSources
	}

	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cm.filePath, data, 0644)
}
