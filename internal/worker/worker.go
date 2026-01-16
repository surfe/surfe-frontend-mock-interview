package worker

import (
	"log"
	"math/rand"
	"time"

	"github.com/surfe/mock-api/internal/data"
	"github.com/surfe/mock-api/internal/database"
	"github.com/surfe/mock-api/internal/models"
)

// Config holds worker configuration
type Config struct {
	// PollInterval is how often the worker checks for enrichments to process
	PollInterval time.Duration

	// PendingToInProgressDelay is how long an enrichment stays pending before moving to in_progress
	PendingToInProgressDelay time.Duration
}

// DefaultConfig returns the default worker configuration
func DefaultConfig() Config {
	return Config{
		PollInterval:             10 * time.Second,
		PendingToInProgressDelay: 10 * time.Second, // Move to in_progress after 10s
	}
}

// Worker processes enrichments in the background
type Worker struct {
	db       *database.DB
	mockData *data.MockData
	config   Config
	stopCh   chan struct{}
}

// New creates a new background worker
func New(db *database.DB, mockData *data.MockData, config Config) *Worker {
	return &Worker{
		db:       db,
		mockData: mockData,
		config:   config,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the background processing loop
func (w *Worker) Start() {
	log.Printf("Starting enrichment worker (poll: %v, pending→in_progress: %v)",
		w.config.PollInterval,
		w.config.PendingToInProgressDelay,
	)

	go w.run()
}

// Stop stops the background worker
func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) run() {
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processEnrichments()

	for {
		select {
		case <-ticker.C:
			w.processEnrichments()
		case <-w.stopCh:
			log.Println("Stopping enrichment worker")
			return
		}
	}
}

func (w *Worker) processEnrichments() {
	// Process pending → in_progress
	w.processPendingEnrichments()
}

func (w *Worker) processPendingEnrichments() {
	enrichments, err := w.db.GetPendingEnrichments(w.config.PendingToInProgressDelay)
	if err != nil {
		log.Printf("Error fetching pending enrichments: %v", err)
		return
	}

	for _, e := range enrichments {
		log.Printf("Moving enrichment %s from pending to in_progress", e.ID)

		err := w.db.UpdateEnrichmentStatus(e.ID, models.EnrichmentStatusInProgress, nil)
		if err != nil {
			log.Printf("Error updating enrichment %s to in_progress: %v", e.ID, err)
			continue
		}

		// Start processing this enrichment through providers in a goroutine
		go w.processEnrichmentThroughProviders(e.ID, e.UserID)
	}
}

// processEnrichmentThroughProviders processes an enrichment by checking each provider
// for the contact's phone number and/or email based on the requested jobs.
// Stops when all requested jobs are found or all providers are checked.
func (w *Worker) processEnrichmentThroughProviders(enrichmentID, userID string) {
	// Get the contact
	contact, exists := w.mockData.GetContact(userID)
	if !exists {
		log.Printf("Contact not found for enrichment %s (userID: %s), marking as failed", enrichmentID, userID)
		w.db.UpdateEnrichmentStatus(enrichmentID, models.EnrichmentStatusFailed, nil)
		return
	}

	// Get the jobs for this enrichment
	jobs, completedJobs, err := w.db.GetEnrichmentJobs(enrichmentID)
	if err != nil {
		log.Printf("Error getting jobs for enrichment %s: %v", enrichmentID, err)
		w.db.UpdateEnrichmentStatus(enrichmentID, models.EnrichmentStatusFailed, nil)
		return
	}

	// Determine which jobs are needed
	needsPhone := false
	needsEmail := false
	for _, job := range jobs {
		if job == "phone" {
			needsPhone = true
		}
		if job == "email" {
			needsEmail = true
		}
	}

	// Get all providers
	providers := w.mockData.GetAllProviders()
	if len(providers) == 0 {
		log.Printf("No providers available, marking enrichment %s as failed", enrichmentID)
		w.db.UpdateEnrichmentStatus(enrichmentID, models.EnrichmentStatusFailed, nil)
		return
	}

	log.Printf("Processing enrichment %s through %d providers for jobs: %v", enrichmentID, len(providers), jobs)

	// Get current result to preserve existing data
	enrichment, err := w.db.GetEnrichment(enrichmentID)
	if err != nil {
		log.Printf("Error getting enrichment %s: %v", enrichmentID, err)
		return
	}
	result := &models.EnrichmentResult{}
	if enrichment != nil && enrichment.Result != nil {
		result = enrichment.Result
	}

	// Process through each provider
	for _, provider := range providers {
		// Check if enrichment was already completed or failed (in case of concurrent updates)
		enrichment, err := w.db.GetEnrichment(enrichmentID)
		if err != nil {
			log.Printf("Error checking enrichment %s status: %v", enrichmentID, err)
			return
		}
		if enrichment == nil || enrichment.Status != models.EnrichmentStatusInProgress {
			log.Printf("Enrichment %s is no longer in_progress, stopping provider processing", enrichmentID)
			return
		}

		// Check which jobs are still needed
		_, completedJobs, err := w.db.GetEnrichmentJobs(enrichmentID)
		if err != nil {
			log.Printf("Error getting completed jobs for enrichment %s: %v", enrichmentID, err)
			return
		}

		stillNeedsPhone := needsPhone
		stillNeedsEmail := needsEmail
		for _, completed := range completedJobs {
			if completed == "phone" {
				stillNeedsPhone = false
			}
			if completed == "email" {
				stillNeedsEmail = false
			}
		}

		// If all jobs are completed, we're done
		if !stillNeedsPhone && !stillNeedsEmail {
			log.Printf("All jobs completed for enrichment %s", enrichmentID)
			return
		}

		log.Printf("Checking provider %s for enrichment %s (needs phone: %v, needs email: %v)", provider.Name, enrichmentID, stillNeedsPhone, stillNeedsEmail)

		// Update the current provider being processed
		if err := w.db.UpdateEnrichmentStatusWithProvider(enrichmentID, models.EnrichmentStatusInProgress, nil, &provider.ID); err != nil {
			log.Printf("Error updating current provider for enrichment %s: %v", enrichmentID, err)
		}

		// Wait 5 seconds ± 1 second (4-6 seconds)
		// rand.Intn(2001) gives 0-2000ms, add 4000ms base to get 4000-6000ms range (4-6 seconds)
		delay := 4*time.Second + time.Duration(rand.Intn(2001))*time.Millisecond
		time.Sleep(delay)

		// Check what this provider can find (30% chance for each job type)
		foundPhone := false
		foundEmail := false

		if stillNeedsPhone && rand.Float32() < 0.3 {
			foundPhone = true
			// Only set phone if contact has one
			if contact.Phone != "" {
				result.Phone = contact.Phone
				log.Printf("Provider %s found phone number for enrichment %s", provider.Name, enrichmentID)
			} else {
				log.Printf("Provider %s would have found phone but contact has no phone number", provider.Name)
			}
		}

		if stillNeedsEmail && rand.Float32() < 0.3 {
			foundEmail = true
			// Only set email if contact has one
			if contact.Email != "" {
				result.Email = contact.Email
				log.Printf("Provider %s found email for enrichment %s", provider.Name, enrichmentID)
			} else {
				log.Printf("Provider %s would have found email but contact has no email", provider.Name)
			}
		}

		// Update result and mark jobs as completed
		if foundPhone || foundEmail {
			// Save the result
			if err := w.db.UpdateEnrichmentStatusWithProvider(enrichmentID, models.EnrichmentStatusInProgress, result, &provider.ID); err != nil {
				log.Printf("Error updating result for enrichment %s: %v", enrichmentID, err)
			}

			// Mark jobs as completed
			if foundPhone {
				if err := w.db.AddCompletedJob(enrichmentID, "phone"); err != nil {
					log.Printf("Error marking phone as completed for enrichment %s: %v", enrichmentID, err)
				}
			}
			if foundEmail {
				if err := w.db.AddCompletedJob(enrichmentID, "email"); err != nil {
					log.Printf("Error marking email as completed for enrichment %s: %v", enrichmentID, err)
				}
			}

			// Check if enrichment was completed by AddCompletedJob (if all jobs were done)
			enrichment, err := w.db.GetEnrichment(enrichmentID)
			if err != nil {
				log.Printf("Error checking enrichment %s status: %v", enrichmentID, err)
				return
			}
			if enrichment != nil && enrichment.Status == models.EnrichmentStatusCompleted {
				log.Printf("All jobs completed for enrichment %s, enrichment already marked as completed", enrichmentID)
				return
			}
		} else {
			log.Printf("Provider %s did not find any requested data for enrichment %s, continuing...", provider.Name, enrichmentID)
		}
	}

	// After checking all providers, ensure all requested jobs have values (set to empty string if not found)
	_, completedJobs, err = w.db.GetEnrichmentJobs(enrichmentID)
	if err != nil {
		log.Printf("Error getting completed jobs for enrichment %s: %v", enrichmentID, err)
		return
	}

	// Set missing values to empty strings and mark as completed
	if needsPhone {
		phoneCompleted := false
		for _, completed := range completedJobs {
			if completed == "phone" {
				phoneCompleted = true
				break
			}
		}
		if !phoneCompleted {
			// Phone was requested but not found, set to empty string
			result.Phone = ""
			log.Printf("Phone job not found after checking all providers, setting to empty string for enrichment %s", enrichmentID)
			// Mark as completed (even though empty)
			if err := w.db.AddCompletedJob(enrichmentID, "phone"); err != nil {
				log.Printf("Error marking phone as completed for enrichment %s: %v", enrichmentID, err)
			}
		}
	}

	if needsEmail {
		emailCompleted := false
		for _, completed := range completedJobs {
			if completed == "email" {
				emailCompleted = true
				break
			}
		}
		if !emailCompleted {
			// Email was requested but not found, set to empty string
			result.Email = ""
			log.Printf("Email job not found after checking all providers, setting to empty string for enrichment %s", enrichmentID)
			// Mark as completed (even though empty)
			if err := w.db.AddCompletedJob(enrichmentID, "email"); err != nil {
				log.Printf("Error marking email as completed for enrichment %s: %v", enrichmentID, err)
			}
		}
	}

	// Complete the enrichment with final result (may include empty strings for missing values)
	// AddCompletedJob will have already completed it if all jobs are now done, but we need to ensure
	// the result with empty strings is saved
	enrichment, err = w.db.GetEnrichment(enrichmentID)
	if err != nil {
		log.Printf("Error checking enrichment %s status: %v", enrichmentID, err)
		return
	}
	if enrichment != nil && enrichment.Status != models.EnrichmentStatusCompleted {
		log.Printf("All providers checked for enrichment %s, completing enrichment", enrichmentID)
		// Clear current provider when completed
		w.db.UpdateEnrichmentStatusWithProvider(enrichmentID, models.EnrichmentStatusCompleted, result, nil)
	} else {
		// Already completed by AddCompletedJob, but ensure result is saved with empty strings
		log.Printf("All providers checked for enrichment %s, updating result with final values", enrichmentID)
		w.db.UpdateEnrichmentStatusWithProvider(enrichmentID, models.EnrichmentStatusCompleted, result, nil)
	}
}
