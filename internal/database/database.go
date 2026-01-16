package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/surfe/mock-api/internal/models"
)

// DB wraps the SQL database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes the schema
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// migrate creates the database schema
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS enrichments (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		result TEXT,
		current_provider_id TEXT,
		is_static INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_enrichments_status ON enrichments(status);
	CREATE INDEX IF NOT EXISTS idx_enrichments_created_at ON enrichments(created_at);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	// Add current_provider_id column if it doesn't exist (for existing databases)
	_, err = db.conn.Exec(`ALTER TABLE enrichments ADD COLUMN current_provider_id TEXT`)
	// Ignore error if column already exists
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// CreateEnrichment creates a new enrichment record
func (db *DB) CreateEnrichment(userID string) (*models.Enrichment, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	enrichment := &models.Enrichment{
		ID:        id,
		UserID:    userID,
		Status:    models.EnrichmentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := db.conn.Exec(`
		INSERT INTO enrichments (id, user_id, status, created_at, updated_at, current_provider_id, is_static)
		VALUES (?, ?, ?, ?, ?, ?, 0)
	`, enrichment.ID, enrichment.UserID, enrichment.Status, enrichment.CreatedAt, enrichment.UpdatedAt, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to create enrichment: %w", err)
	}

	return enrichment, nil
}

// GetEnrichment retrieves an enrichment by ID
func (db *DB) GetEnrichment(id string) (*models.Enrichment, error) {
	var enrichment models.Enrichment
	var resultJSON sql.NullString
	var currentProviderID sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, user_id, status, created_at, updated_at, result, current_provider_id
		FROM enrichments
		WHERE id = ?
	`, id).Scan(&enrichment.ID, &enrichment.UserID, &enrichment.Status, &enrichment.CreatedAt, &enrichment.UpdatedAt, &resultJSON, &currentProviderID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get enrichment: %w", err)
	}

	if resultJSON.Valid && resultJSON.String != "" {
		var result models.EnrichmentResult
		if err := json.Unmarshal([]byte(resultJSON.String), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
		enrichment.Result = &result
	}

	if currentProviderID.Valid && currentProviderID.String != "" {
		enrichment.CurrentProvider = &models.Provider{
			ID: currentProviderID.String,
		}
	}

	return &enrichment, nil
}

// GetPendingEnrichments returns enrichments that are pending and older than the given duration
func (db *DB) GetPendingEnrichments(olderThan time.Duration) ([]*models.Enrichment, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)

	rows, err := db.conn.Query(`
		SELECT id, user_id, status, created_at, updated_at
		FROM enrichments
		WHERE status = ? AND is_static = 0 AND created_at < ?
	`, models.EnrichmentStatusPending, cutoff)

	if err != nil {
		return nil, fmt.Errorf("failed to query pending enrichments: %w", err)
	}
	defer rows.Close()

	var enrichments []*models.Enrichment
	for rows.Next() {
		var e models.Enrichment
		if err := rows.Scan(&e.ID, &e.UserID, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan enrichment: %w", err)
		}
		enrichments = append(enrichments, &e)
	}

	return enrichments, nil
}

// GetInProgressEnrichments returns enrichments that are in_progress and older than the given duration
func (db *DB) GetInProgressEnrichments(olderThan time.Duration) ([]*models.Enrichment, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)

	rows, err := db.conn.Query(`
		SELECT id, user_id, status, created_at, updated_at
		FROM enrichments
		WHERE status = ? AND is_static = 0 AND updated_at < ?
	`, models.EnrichmentStatusInProgress, cutoff)

	if err != nil {
		return nil, fmt.Errorf("failed to query in_progress enrichments: %w", err)
	}
	defer rows.Close()

	var enrichments []*models.Enrichment
	for rows.Next() {
		var e models.Enrichment
		if err := rows.Scan(&e.ID, &e.UserID, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan enrichment: %w", err)
		}
		enrichments = append(enrichments, &e)
	}

	return enrichments, nil
}

// UpdateEnrichmentStatus updates the status of an enrichment
func (db *DB) UpdateEnrichmentStatus(id string, status models.EnrichmentStatus, result *models.EnrichmentResult) error {
	return db.UpdateEnrichmentStatusWithProvider(id, status, result, nil)
}

// UpdateEnrichmentStatusWithProvider updates the status and current provider of an enrichment
func (db *DB) UpdateEnrichmentStatusWithProvider(id string, status models.EnrichmentStatus, result *models.EnrichmentResult, providerID *string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	var resultJSON *string
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		s := string(data)
		resultJSON = &s
	}

	_, err := db.conn.Exec(`
		UPDATE enrichments
		SET status = ?, updated_at = ?, result = ?, current_provider_id = ?
		WHERE id = ?
	`, status, now, resultJSON, providerID, id)

	if err != nil {
		return fmt.Errorf("failed to update enrichment: %w", err)
	}

	return nil
}

// SeedStaticEnrichments creates the static test enrichments if they don't exist
func (db *DB) SeedStaticEnrichments() error {
	staticEnrichments := []struct {
		ID     string
		UserID string
		Status models.EnrichmentStatus
		Result *models.EnrichmentResult
	}{
		{
			ID:     "e5f6a7b8-c9d0-1234-ef12-345678901234",
			UserID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890", // John Doe
			Status: models.EnrichmentStatusPending,
		},
		{
			ID:     "f6a7b8c9-d0e1-2345-f123-456789012345",
			UserID: "b2c3d4e5-f6a7-8901-bcde-f12345678901", // Jane Smith
			Status: models.EnrichmentStatusInProgress,
		},
		{
			ID:     "a7b8c9d0-e1f2-3456-0123-567890123456",
			UserID: "c3d4e5f6-a7b8-9012-cdef-123456789012", // Bob Johnson
			Status: models.EnrichmentStatusCompleted,
			Result: &models.EnrichmentResult{
				LinkedInURL: "https://linkedin.com/in/bobjohnson",
				TwitterURL:  "https://twitter.com/bob_johnson",
				Skills:      []string{"Leadership", "Architecture", "Go", "Kubernetes"},
				Experience:  12,
			},
		},
		{
			ID:     "b8c9d0e1-f2a3-4567-1234-678901234567",
			UserID: "d4e5f6a7-b8c9-0123-def1-234567890123", // Alice Williams
			Status: models.EnrichmentStatusFailed,
		},
	}

	now := time.Now().UTC().Format(time.RFC3339)

	for _, e := range staticEnrichments {
		// Check if already exists
		var count int
		err := db.conn.QueryRow("SELECT COUNT(*) FROM enrichments WHERE id = ?", e.ID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check existing enrichment: %w", err)
		}

		if count > 0 {
			continue // Already seeded
		}

		var resultJSON *string
		if e.Result != nil {
			data, err := json.Marshal(e.Result)
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}
			s := string(data)
			resultJSON = &s
		}

		_, err = db.conn.Exec(`
			INSERT INTO enrichments (id, user_id, status, created_at, updated_at, result, current_provider_id, is_static)
			VALUES (?, ?, ?, ?, ?, ?, ?, 1)
		`, e.ID, e.UserID, e.Status, now, now, resultJSON, nil)

		if err != nil {
			return fmt.Errorf("failed to seed enrichment %s: %w", e.ID, err)
		}

		log.Printf("Seeded static enrichment: %s (%s)", e.ID, e.Status)
	}

	return nil
}
