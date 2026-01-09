package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/nexus/nexus-agent/internal/config"
	"github.com/nexus/nexus-agent/internal/queue"
	"github.com/nexus/nexus-agent/internal/sender"
)

// Handler handles HTTP requests
type Handler struct {
	config *config.Config
	sender *sender.Sender
	queue  *queue.Queue
}

// New creates a new Handler instance
func New(cfg *config.Config, s *sender.Sender, q *queue.Queue) *Handler {
	return &Handler{
		config: cfg,
		sender: s,
		queue:  q,
	}
}

// SendRequest represents the incoming request body
type SendRequest struct {
	AppKey string                 `json:"app_key"`
	Data   map[string]interface{} `json:"data"`
}

// SendResponse represents the response body
type SendResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      int64  `json:"id,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status         string `json:"status"`
	QueueSize      int    `json:"queue_size"`
	AppsConfigured int    `json:"apps_configured"`
}

// HandleSend handles POST /send requests
func (h *Handler) HandleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.AppKey == "" {
		h.jsonError(w, "app_key is required", http.StatusBadRequest)
		return
	}
	if req.Data == nil || len(req.Data) == 0 {
		h.jsonError(w, "data is required", http.StatusBadRequest)
		return
	}

	// Check if app_key is configured
	if h.config.GetAppByKey(req.AppKey) == nil {
		h.jsonError(w, "unknown app_key - not configured in agent", http.StatusBadRequest)
		return
	}

	// Try to send immediately
	result := h.sender.Send(req.AppKey, req.Data)

	if result.Success {
		h.jsonSuccess(w, "data sent successfully", 0)
		return
	}

	// If sending failed and buffering is enabled, queue the message
	if h.config.Buffer.Enabled && result.Retry && h.queue != nil {
		id, err := h.queue.Enqueue(req.AppKey, req.Data)
		if err != nil {
			log.Printf("Failed to queue message: %v", err)
			h.jsonError(w, "failed to send and queue message", http.StatusInternalServerError)
			return
		}
		h.jsonResponse(w, SendResponse{
			Success: true,
			Message: "data queued for delivery (server unavailable)",
			ID:      id,
		}, http.StatusAccepted)
		return
	}

	// Failed to send and can't queue
	h.jsonError(w, result.Message, http.StatusBadGateway)
}

// HandleHealth handles GET /health requests
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	queueSize := 0
	if h.queue != nil {
		size, _ := h.queue.Size()
		queueSize = size
	}

	resp := HealthResponse{
		Status:         "healthy",
		QueueSize:      queueSize,
		AppsConfigured: len(h.config.Apps),
	}

	h.jsonResponse(w, resp, http.StatusOK)
}

// Helper methods

func (h *Handler) jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) jsonSuccess(w http.ResponseWriter, message string, id int64) {
	h.jsonResponse(w, SendResponse{
		Success: true,
		Message: message,
		ID:      id,
	}, http.StatusOK)
}

func (h *Handler) jsonError(w http.ResponseWriter, message string, status int) {
	h.jsonResponse(w, SendResponse{
		Success: false,
		Message: message,
	}, status)
}
