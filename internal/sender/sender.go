package sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nexus/nexus-agent/internal/config"
	"github.com/nexus/nexus-agent/internal/crypto"
)

// Sender handles sending encrypted data to the Nexus server
type Sender struct {
	config *config.Config
	client *http.Client
}

// New creates a new Sender instance
func New(cfg *config.Config) *Sender {
	return &Sender{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Nexus.Timeout,
		},
	}
}

// SendResult contains the result of a send operation
type SendResult struct {
	Success bool
	Message string
	Retry   bool
}

// Send encrypts and sends data to the Nexus server
func (s *Sender) Send(appKey string, data map[string]interface{}) SendResult {
	// Find the app configuration
	appConfig := s.config.GetAppByKey(appKey)
	if appConfig == nil {
		return SendResult{
			Success: false,
			Message: fmt.Sprintf("unknown app_key: %s", appKey),
			Retry:   false, // Don't retry - configuration issue
		}
	}

	// Encrypt the data using the Nexus Enigma format
	encryptedPayload, err := crypto.EncryptPayload(data, appConfig.MasterSecret, appKey)
	if err != nil {
		return SendResult{
			Success: false,
			Message: fmt.Sprintf("encryption failed: %v", err),
			Retry:   false,
		}
	}

	// Marshal the encrypted payload directly (it's already in the correct format)
	bodyJSON, err := json.Marshal(encryptedPayload)
	if err != nil {
		return SendResult{
			Success: false,
			Message: fmt.Sprintf("failed to marshal body: %v", err),
			Retry:   false,
		}
	}

	// Send to Nexus with retry
	var lastErr error
	for attempt := 1; attempt <= s.config.Nexus.RetryAttempts; attempt++ {
		result := s.doSend(appKey, appConfig.MasterSecret, bodyJSON)
		if result.Success {
			return result
		}

		lastErr = fmt.Errorf(result.Message)

		// If not retryable, return immediately
		if !result.Retry {
			return result
		}

		// Wait before retry
		if attempt < s.config.Nexus.RetryAttempts {
			time.Sleep(s.config.Nexus.RetryDelay)
		}
	}

	return SendResult{
		Success: false,
		Message: fmt.Sprintf("all retries failed: %v", lastErr),
		Retry:   true, // Can still retry later (queue)
	}
}

// doSend performs the actual HTTP request
func (s *Sender) doSend(appKey, masterSecret string, body []byte) SendResult {
	url := fmt.Sprintf("%s/ingress", s.config.Nexus.ServerURL)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return SendResult{
			Success: false,
			Message: fmt.Sprintf("failed to create request: %v", err),
			Retry:   false,
		}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", appKey) // Nexus API expects X-API-Key

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return SendResult{
			Success: false,
			Message: fmt.Sprintf("request failed: %v", err),
			Retry:   true, // Network error - can retry
		}
	}
	defer resp.Body.Close()

	// Read response
	respBody, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return SendResult{
			Success: true,
			Message: "data sent successfully",
			Retry:   false,
		}
	}

	// Server error - may retry
	if resp.StatusCode >= 500 {
		return SendResult{
			Success: false,
			Message: fmt.Sprintf("server error %d: %s", resp.StatusCode, string(respBody)),
			Retry:   true,
		}
	}

	// Client error - don't retry
	return SendResult{
		Success: false,
		Message: fmt.Sprintf("client error %d: %s", resp.StatusCode, string(respBody)),
		Retry:   false,
	}
}
