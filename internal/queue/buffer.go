package queue

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Message represents a queued message
type Message struct {
	ID        int64
	AppKey    string
	Data      map[string]interface{}
	CreatedAt time.Time
	Attempts  int
}

// Queue handles offline buffering of messages
type Queue struct {
	db      *sql.DB
	maxSize int
	mu      sync.Mutex
}

// New creates a new queue instance
func New(dbPath string, maxSize int) (*Queue, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			app_key TEXT NOT NULL,
			data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			attempts INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &Queue{
		db:      db,
		maxSize: maxSize,
	}, nil
}

// Enqueue adds a message to the queue
func (q *Queue) Enqueue(appKey string, data map[string]interface{}) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check queue size
	var count int
	err := q.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to check queue size: %w", err)
	}

	if count >= q.maxSize {
		return 0, fmt.Errorf("queue is full (max: %d)", q.maxSize)
	}

	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Insert message
	result, err := q.db.Exec(
		"INSERT INTO messages (app_key, data) VALUES (?, ?)",
		appKey, string(dataJSON),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert message: %w", err)
	}

	id, _ := result.LastInsertId()
	return id, nil
}

// Dequeue retrieves the oldest message from the queue
func (q *Queue) Dequeue() (*Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var msg Message
	var dataJSON string

	err := q.db.QueryRow(`
		SELECT id, app_key, data, created_at, attempts 
		FROM messages 
		ORDER BY id ASC 
		LIMIT 1
	`).Scan(&msg.ID, &msg.AppKey, &dataJSON, &msg.CreatedAt, &msg.Attempts)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue: %w", err)
	}

	// Parse data JSON
	if err := json.Unmarshal([]byte(dataJSON), &msg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &msg, nil
}

// Remove deletes a message from the queue
func (q *Queue) Remove(id int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	_, err := q.db.Exec("DELETE FROM messages WHERE id = ?", id)
	return err
}

// IncrementAttempts increases the attempt counter for a message
func (q *Queue) IncrementAttempts(id int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	_, err := q.db.Exec("UPDATE messages SET attempts = attempts + 1 WHERE id = ?", id)
	return err
}

// Size returns the number of messages in the queue
func (q *Queue) Size() (int, error) {
	var count int
	err := q.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	return count, err
}

// Close closes the database connection
func (q *Queue) Close() error {
	return q.db.Close()
}
