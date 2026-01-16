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

	// For in-memory databases, use shared cache mode so all connections share the same database
	// This is important because :memory: creates a separate database per connection by default
	if dbPath == ":memory:" {
		dbPath = "file::memory:?cache=shared"
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings for in-memory databases
	// Use a single connection for in-memory to ensure consistency
	if dbPath == "file::memory:?cache=shared" || dbPath == ":memory:" {
		conn.SetMaxOpenConns(1)
		conn.SetMaxIdleConns(1)
	}

	// Test the connection with a simple query
	if _, err := conn.Exec("SELECT 1"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to test database connection: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Verify migration by checking if we can query the table
	var count int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM enrichments").Scan(&count); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to verify enrichments table after migration: %w", err)
	}

	log.Printf("Database initialized successfully with %d enrichments", count)

	return db, nil
}

// migrate creates the database schema
func (db *DB) migrate() error {
	// Enable foreign keys and other SQLite settings
	if _, err := db.conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to set foreign keys: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS enrichments (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		result TEXT,
		current_provider_id TEXT,
		phone_provider_id TEXT,
		email_provider_id TEXT,
		jobs TEXT,
		completed_jobs TEXT,
		is_static INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_enrichments_status ON enrichments(status);
	CREATE INDEX IF NOT EXISTS idx_enrichments_created_at ON enrichments(created_at);
	`

	if _, err := db.conn.Exec(schema); err != nil {
		return fmt.Errorf("failed to create enrichments table: %w", err)
	}

	// Verify table was created
	var tableName string
	err := db.conn.QueryRow(`
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name='enrichments'
	`).Scan(&tableName)
	if err != nil {
		return fmt.Errorf("failed to verify enrichments table exists: %w", err)
	}

	log.Printf("Database migration successful: enrichments table created")

	// Add columns if they don't exist (for existing databases)
	// SQLite doesn't support IF NOT EXISTS for ALTER TABLE ADD COLUMN,
	// so we ignore errors if columns already exist
	_, _ = db.conn.Exec(`ALTER TABLE enrichments ADD COLUMN current_provider_id TEXT`)
	_, _ = db.conn.Exec(`ALTER TABLE enrichments ADD COLUMN phone_provider_id TEXT`)
	_, _ = db.conn.Exec(`ALTER TABLE enrichments ADD COLUMN email_provider_id TEXT`)
	_, _ = db.conn.Exec(`ALTER TABLE enrichments ADD COLUMN jobs TEXT`)
	_, _ = db.conn.Exec(`ALTER TABLE enrichments ADD COLUMN completed_jobs TEXT`)

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// CreateEnrichment creates a new enrichment record
func (db *DB) CreateEnrichment(userID string, jobs []string) (*models.Enrichment, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	enrichment := &models.Enrichment{
		ID:        id,
		UserID:    userID,
		Status:    models.EnrichmentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Default to phone if no jobs specified
	if len(jobs) == 0 {
		jobs = []string{"phone"}
	}

	// Marshal jobs to JSON
	jobsJSON, err := json.Marshal(jobs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal jobs: %w", err)
	}

	_, err = db.conn.Exec(`
		INSERT INTO enrichments (id, user_id, status, created_at, updated_at, current_provider_id, phone_provider_id, email_provider_id, jobs, completed_jobs, is_static)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
	`, enrichment.ID, enrichment.UserID, enrichment.Status, enrichment.CreatedAt, enrichment.UpdatedAt, nil, nil, nil, string(jobsJSON), "[]")

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
	var phoneProviderID sql.NullString
	var emailProviderID sql.NullString
	var jobsJSON sql.NullString
	var completedJobsJSON sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, user_id, status, created_at, updated_at, result, current_provider_id, phone_provider_id, email_provider_id, jobs, completed_jobs
		FROM enrichments
		WHERE id = ?
	`, id).Scan(&enrichment.ID, &enrichment.UserID, &enrichment.Status, &enrichment.CreatedAt, &enrichment.UpdatedAt, &resultJSON, &currentProviderID, &phoneProviderID, &emailProviderID, &jobsJSON, &completedJobsJSON)

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

	// Store provider IDs for handler to populate JobStatus objects
	// The handler will populate the Phone and Email JobStatus objects
	if phoneProviderID.Valid && phoneProviderID.String != "" {
		// This will be used by the handler
	}
	if emailProviderID.Valid && emailProviderID.String != "" {
		// This will be used by the handler
	}

	return &enrichment, nil
}

// GetEnrichmentWithProviders retrieves an enrichment with provider IDs for phone and email
func (db *DB) GetEnrichmentWithProviders(id string) (*models.Enrichment, *string, *string, error) {
	enrichment, err := db.GetEnrichment(id)
	if err != nil {
		return nil, nil, nil, err
	}
	if enrichment == nil {
		return nil, nil, nil, nil
	}

	var phoneProviderID sql.NullString
	var emailProviderID sql.NullString

	err = db.conn.QueryRow(`
		SELECT phone_provider_id, email_provider_id
		FROM enrichments
		WHERE id = ?
	`, id).Scan(&phoneProviderID, &emailProviderID)

	if err != nil && err != sql.ErrNoRows {
		return nil, nil, nil, fmt.Errorf("failed to get provider IDs: %w", err)
	}

	var phoneProviderIDPtr *string
	var emailProviderIDPtr *string

	if phoneProviderID.Valid && phoneProviderID.String != "" {
		phoneProviderIDPtr = &phoneProviderID.String
	}
	if emailProviderID.Valid && emailProviderID.String != "" {
		emailProviderIDPtr = &emailProviderID.String
	}

	return enrichment, phoneProviderIDPtr, emailProviderIDPtr, nil
}

// GetEnrichmentJobs retrieves the jobs and completed jobs for an enrichment
func (db *DB) GetEnrichmentJobs(id string) ([]string, []string, error) {
	var jobsJSON sql.NullString
	var completedJobsJSON sql.NullString

	err := db.conn.QueryRow(`
		SELECT jobs, completed_jobs
		FROM enrichments
		WHERE id = ?
	`, id).Scan(&jobsJSON, &completedJobsJSON)

	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("enrichment not found")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get enrichment jobs: %w", err)
	}

	var jobs []string
	if jobsJSON.Valid && jobsJSON.String != "" {
		if err := json.Unmarshal([]byte(jobsJSON.String), &jobs); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal jobs: %w", err)
		}
	} else {
		// Default to phone for backward compatibility
		jobs = []string{"phone"}
	}

	var completedJobs []string
	if completedJobsJSON.Valid && completedJobsJSON.String != "" {
		if err := json.Unmarshal([]byte(completedJobsJSON.String), &completedJobs); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal completed jobs: %w", err)
		}
	}

	return jobs, completedJobs, nil
}

// AddCompletedJob adds a job to the completed jobs list
func (db *DB) AddCompletedJob(enrichmentID, job string) error {
	jobs, completedJobs, err := db.GetEnrichmentJobs(enrichmentID)
	if err != nil {
		return err
	}

	// Check if job is already completed
	for _, completed := range completedJobs {
		if completed == job {
			return nil // Already completed
		}
	}

	// Add to completed jobs
	completedJobs = append(completedJobs, job)

	// Check if all jobs are completed
	allCompleted := true
	for _, job := range jobs {
		found := false
		for _, completed := range completedJobs {
			if completed == job {
				found = true
				break
			}
		}
		if !found {
			allCompleted = false
			break
		}
	}

	// Marshal completed jobs
	completedJobsJSON, err := json.Marshal(completedJobs)
	if err != nil {
		return fmt.Errorf("failed to marshal completed jobs: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Update the enrichment
	if allCompleted {
		// Get current result to preserve it
		enrichment, err := db.GetEnrichment(enrichmentID)
		if err != nil {
			return err
		}

		var resultJSON *string
		if enrichment != nil && enrichment.Result != nil {
			data, err := json.Marshal(enrichment.Result)
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}
			s := string(data)
			resultJSON = &s
		}

		// When all jobs are completed, clear all provider IDs
		_, err = db.conn.Exec(`
			UPDATE enrichments
			SET status = ?, updated_at = ?, completed_jobs = ?, current_provider_id = ?, phone_provider_id = ?, email_provider_id = ?, result = ?
			WHERE id = ?
		`, models.EnrichmentStatusCompleted, now, string(completedJobsJSON), nil, nil, nil, resultJSON, enrichmentID)
		if err != nil {
			return fmt.Errorf("failed to update enrichment: %w", err)
		}
	} else {
		_, err = db.conn.Exec(`
			UPDATE enrichments
			SET updated_at = ?, completed_jobs = ?
			WHERE id = ?
		`, now, string(completedJobsJSON), enrichmentID)
		if err != nil {
			return fmt.Errorf("failed to update enrichment: %w", err)
		}
	}

	return nil
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
	return db.UpdateEnrichmentStatusWithJobProvider(id, status, result, providerID, "")
}

// UpdateEnrichmentStatusWithJobProvider updates the status, result, and provider for a specific job type
func (db *DB) UpdateEnrichmentStatusWithJobProvider(id string, status models.EnrichmentStatus, result *models.EnrichmentResult, providerID *string, jobType string) error {
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

	// Update based on job type
	if jobType == "phone" {
		_, err := db.conn.Exec(`
			UPDATE enrichments
			SET status = ?, updated_at = ?, result = ?, phone_provider_id = ?
			WHERE id = ?
		`, status, now, resultJSON, providerID, id)
		if err != nil {
			return fmt.Errorf("failed to update enrichment: %w", err)
		}
	} else if jobType == "email" {
		_, err := db.conn.Exec(`
			UPDATE enrichments
			SET status = ?, updated_at = ?, result = ?, email_provider_id = ?
			WHERE id = ?
		`, status, now, resultJSON, providerID, id)
		if err != nil {
			return fmt.Errorf("failed to update enrichment: %w", err)
		}
	} else {
		// Legacy support or general update - update current_provider_id and clear job-specific providers if status is completed
		if status == models.EnrichmentStatusCompleted {
			_, err := db.conn.Exec(`
				UPDATE enrichments
				SET status = ?, updated_at = ?, result = ?, current_provider_id = ?, phone_provider_id = ?, email_provider_id = ?
				WHERE id = ?
			`, status, now, resultJSON, nil, nil, nil, id)
			if err != nil {
				return fmt.Errorf("failed to update enrichment: %w", err)
			}
		} else {
			_, err := db.conn.Exec(`
				UPDATE enrichments
				SET status = ?, updated_at = ?, result = ?, current_provider_id = ?
				WHERE id = ?
			`, status, now, resultJSON, providerID, id)
			if err != nil {
				return fmt.Errorf("failed to update enrichment: %w", err)
			}
		}
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

		// Default to phone job for static enrichments
		jobsJSON := `["phone"]`

		_, err = db.conn.Exec(`
			INSERT INTO enrichments (id, user_id, status, created_at, updated_at, result, current_provider_id, phone_provider_id, email_provider_id, jobs, completed_jobs, is_static)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
		`, e.ID, e.UserID, e.Status, now, now, resultJSON, nil, nil, nil, jobsJSON, "[]")

		if err != nil {
			return fmt.Errorf("failed to seed enrichment %s: %w", e.ID, err)
		}

		log.Printf("Seeded static enrichment: %s (%s)", e.ID, e.Status)
	}

	return nil
}
