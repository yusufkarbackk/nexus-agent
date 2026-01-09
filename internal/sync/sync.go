package sync

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/nexus/nexus-agent/internal/config"
)

// SyncResponse is the response from the server's sync endpoint
type SyncResponse struct {
	Success  bool      `json:"success"`
	Message  string    `json:"message"`
	Apps     []AppData `json:"apps"`
	SyncedAt string    `json:"synced_at"`
}

// AppData is the app data from the sync response
type AppData struct {
	ID                uint64 `json:"id"`
	Name              string `json:"name"`
	AppKey            string `json:"app_key"`
	MasterSecret      string `json:"master_secret"`
	EncryptionEnabled bool   `json:"encryption_enabled"`
}

// Syncer handles auto-sync with the Nexus server
type Syncer struct {
	config     *config.Config
	httpClient *http.Client
	stopCh     chan struct{}
	running    bool
}

// NewSyncer creates a new syncer instance
func NewSyncer(cfg *config.Config) *Syncer {
	return &Syncer{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Nexus.Timeout,
		},
		stopCh: make(chan struct{}),
	}
}

// Start begins the periodic sync loop
func (s *Syncer) Start() {
	if !s.config.HasAutoSync() {
		log.Println("Auto-sync disabled (no agent_token configured)")
		return
	}

	s.running = true
	log.Printf("Starting auto-sync (interval: %v)", s.config.Nexus.SyncInterval)

	// Initial sync
	if err := s.Sync(); err != nil {
		log.Printf("WARN: Initial sync failed: %v (will retry)", err)
	}

	// Periodic sync
	go func() {
		ticker := time.NewTicker(s.config.Nexus.SyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.Sync(); err != nil {
					log.Printf("WARN: Sync failed: %v", err)
				}
			case <-s.stopCh:
				log.Println("Auto-sync stopped")
				return
			}
		}
	}()
}

// Stop stops the sync loop
func (s *Syncer) Stop() {
	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// Sync performs a single sync with the server
func (s *Syncer) Sync() error {
	url := fmt.Sprintf("%s/agent/sync", s.config.Nexus.ServerURL)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Agent-Token", s.config.Nexus.AgentToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var syncResp SyncResponse
	if err := json.Unmarshal(body, &syncResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !syncResp.Success {
		return fmt.Errorf("sync failed: %s", syncResp.Message)
	}

	// Update config with synced apps
	apps := make([]config.AppConfig, len(syncResp.Apps))
	for i, app := range syncResp.Apps {
		apps[i] = config.AppConfig{
			Name:         app.Name,
			AppKey:       app.AppKey,
			MasterSecret: app.MasterSecret,
		}
	}

	s.config.UpdateSyncedApps(apps)
	log.Printf("Synced %d apps from server", len(apps))

	return nil
}
