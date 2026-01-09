package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nexus/nexus-agent/internal/config"
	"github.com/nexus/nexus-agent/internal/handler"
	"github.com/nexus/nexus-agent/internal/queue"
	"github.com/nexus/nexus-agent/internal/sender"
	"github.com/nexus/nexus-agent/internal/sync"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Nexus Agent starting...")

	// Start auto-sync if configured
	var syncer *sync.Syncer
	if cfg.HasAutoSync() {
		syncer = sync.NewSyncer(cfg)
		syncer.Start()
		defer syncer.Stop()
		log.Printf("Auto-sync enabled (token configured)")
	} else {
		log.Printf("Using static config for %d app(s)", len(cfg.Apps))
	}

	// Initialize sender
	s := sender.New(cfg)

	// Initialize queue if buffering is enabled
	var q *queue.Queue
	if cfg.Buffer.Enabled {
		q, err = queue.New(cfg.Buffer.DBPath, cfg.Buffer.MaxSize)
		if err != nil {
			log.Fatalf("Failed to initialize queue: %v", err)
		}
		defer q.Close()
		log.Printf("Offline buffering enabled (max: %d messages)", cfg.Buffer.MaxSize)

		// Start queue processor
		go processQueue(cfg, s, q)
	}

	// Initialize handler
	h := handler.New(cfg, s, q)

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/send", h.HandleSend)
	mux.HandleFunc("/health", h.HandleHealth)

	// Create server
	addr := fmt.Sprintf("%s:%d", cfg.Agent.Bind, cfg.Agent.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Agent listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down agent...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Agent stopped")
}

// processQueue continuously processes queued messages
func processQueue(cfg *config.Config, s *sender.Sender, q *queue.Queue) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for {
			// Get next message from queue
			msg, err := q.Dequeue()
			if err != nil {
				log.Printf("Queue dequeue error: %v", err)
				break
			}
			if msg == nil {
				// Queue is empty
				break
			}

			// Try to send
			result := s.Send(msg.AppKey, msg.Data)
			if result.Success {
				// Remove from queue on success
				q.Remove(msg.ID)
				log.Printf("Queued message %d sent successfully", msg.ID)
			} else if !result.Retry || msg.Attempts >= cfg.Nexus.RetryAttempts*3 {
				// Remove if not retryable or too many attempts
				q.Remove(msg.ID)
				log.Printf("Queued message %d failed permanently: %s", msg.ID, result.Message)
			} else {
				// Increment attempts and keep in queue
				q.IncrementAttempts(msg.ID)
				log.Printf("Queued message %d failed, will retry later: %s", msg.ID, result.Message)
				break // Wait for next tick before trying more
			}
		}
	}
}

// loggingMiddleware logs all HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
