package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the agent configuration
type Config struct {
	Agent  AgentConfig  `yaml:"agent"`
	Nexus  NexusConfig  `yaml:"nexus"`
	Apps   []AppConfig  `yaml:"apps"` // Static apps (fallback if auto-sync fails)
	Buffer BufferConfig `yaml:"buffer"`

	// Runtime state (not from config file)
	syncedApps map[string]*AppConfig
	mu         sync.RWMutex
}

// AgentConfig contains local HTTP server settings
type AgentConfig struct {
	Port int    `yaml:"port"`
	Bind string `yaml:"bind"`
}

// NexusConfig contains settings for connecting to the Nexus server
type NexusConfig struct {
	ServerURL     string        `yaml:"server_url"`
	AgentToken    string        `yaml:"agent_token"`   // NEW: Token for auto-sync
	SyncInterval  time.Duration `yaml:"sync_interval"` // How often to sync (default: 60s)
	Timeout       time.Duration `yaml:"timeout"`
	RetryAttempts int           `yaml:"retry_attempts"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
}

// AppConfig contains credentials for a sender app
type AppConfig struct {
	Name         string `yaml:"name" json:"name"`
	AppKey       string `yaml:"app_key" json:"app_key"`
	MasterSecret string `yaml:"master_secret" json:"master_secret"`
}

// BufferConfig contains settings for offline buffering
type BufferConfig struct {
	Enabled bool   `yaml:"enabled"`
	MaxSize int    `yaml:"max_size"`
	DBPath  string `yaml:"db_path"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Agent.Port == 0 {
		config.Agent.Port = 9000
	}
	if config.Agent.Bind == "" {
		config.Agent.Bind = "127.0.0.1"
	}
	if config.Nexus.Timeout == 0 {
		config.Nexus.Timeout = 30 * time.Second
	}
	if config.Nexus.RetryAttempts == 0 {
		config.Nexus.RetryAttempts = 3
	}
	if config.Nexus.RetryDelay == 0 {
		config.Nexus.RetryDelay = 5 * time.Second
	}
	if config.Nexus.SyncInterval == 0 {
		config.Nexus.SyncInterval = 60 * time.Second
	}
	if config.Buffer.MaxSize == 0 {
		config.Buffer.MaxSize = 10000
	}
	if config.Buffer.DBPath == "" {
		config.Buffer.DBPath = "./queue.db"
	}

	// Validate
	if config.Nexus.ServerURL == "" {
		return nil, fmt.Errorf("nexus.server_url is required")
	}

	// Either agent_token OR static apps must be configured
	if config.Nexus.AgentToken == "" && len(config.Apps) == 0 {
		return nil, fmt.Errorf("either nexus.agent_token or apps must be configured")
	}

	// Initialize synced apps map
	config.syncedApps = make(map[string]*AppConfig)

	return &config, nil
}

// GetAppByKey finds an app configuration by its app_key
// Checks synced apps first, then static config
func (c *Config) GetAppByKey(appKey string) *AppConfig {
	c.mu.RLock()
	if app, ok := c.syncedApps[appKey]; ok {
		c.mu.RUnlock()
		return app
	}
	c.mu.RUnlock()

	// Fallback to static config
	for i := range c.Apps {
		if c.Apps[i].AppKey == appKey {
			return &c.Apps[i]
		}
	}
	return nil
}

// UpdateSyncedApps updates the synced apps from the server
func (c *Config) UpdateSyncedApps(apps []AppConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.syncedApps = make(map[string]*AppConfig)
	for i := range apps {
		c.syncedApps[apps[i].AppKey] = &apps[i]
	}
}

// GetAllApps returns all configured apps (synced + static)
func (c *Config) GetAllApps() []AppConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	apps := make([]AppConfig, 0)

	// Add synced apps
	for _, app := range c.syncedApps {
		apps = append(apps, *app)
	}

	// Add static apps that aren't in synced
	for _, app := range c.Apps {
		if _, exists := c.syncedApps[app.AppKey]; !exists {
			apps = append(apps, app)
		}
	}

	return apps
}

// HasAutoSync returns true if agent token is configured
func (c *Config) HasAutoSync() bool {
	return c.Nexus.AgentToken != ""
}

// AppCount returns the total number of configured apps
func (c *Config) AppCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := len(c.syncedApps)
	for _, app := range c.Apps {
		if _, exists := c.syncedApps[app.AppKey]; !exists {
			count++
		}
	}
	return count
}
