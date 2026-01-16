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
// for the contact's phone number. Stops when found or all providers are checked.
func (w *Worker) processEnrichmentThroughProviders(enrichmentID, userID string) {
	// Get the contact to find their phone number
	contact, exists := w.mockData.GetContact(userID)
	if !exists {
		log.Printf("Contact not found for enrichment %s (userID: %s), marking as failed", enrichmentID, userID)
		w.db.UpdateEnrichmentStatus(enrichmentID, models.EnrichmentStatusFailed, nil)
		return
	}

	// If contact doesn't have a phone number, mark as failed
	if contact.Phone == "" {
		log.Printf("Contact %s has no phone number, marking enrichment %s as failed", userID, enrichmentID)
		w.db.UpdateEnrichmentStatus(enrichmentID, models.EnrichmentStatusFailed, nil)
		return
	}

	// Get all providers
	providers := w.mockData.GetAllProviders()
	if len(providers) == 0 {
		log.Printf("No providers available, marking enrichment %s as failed", enrichmentID)
		w.db.UpdateEnrichmentStatus(enrichmentID, models.EnrichmentStatusFailed, nil)
		return
	}

	log.Printf("Processing enrichment %s through %d providers", enrichmentID, len(providers))

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

		log.Printf("Checking provider %s for enrichment %s", provider.Name, enrichmentID)

		// Update the current provider being processed
		if err := w.db.UpdateEnrichmentStatusWithProvider(enrichmentID, models.EnrichmentStatusInProgress, nil, &provider.ID); err != nil {
			log.Printf("Error updating current provider for enrichment %s: %v", enrichmentID, err)
		}

		// Wait 5 seconds ± 1 second (4-6 seconds)
		// rand.Intn(2001) gives 0-2000ms, add 4000ms base to get 4000-6000ms range (4-6 seconds)
		delay := 4*time.Second + time.Duration(rand.Intn(2001))*time.Millisecond
		time.Sleep(delay)

		// Randomly determine if this provider finds the phone number (30% chance)
		if rand.Float32() < 0.3 {
			log.Printf("Provider %s found phone number for enrichment %s", provider.Name, enrichmentID)

			// Create result with the found phone number
			result := &models.EnrichmentResult{
				Phone: contact.Phone,
			}

			// Clear current provider when completed
			err := w.db.UpdateEnrichmentStatusWithProvider(enrichmentID, models.EnrichmentStatusCompleted, result, nil)
			if err != nil {
				log.Printf("Error completing enrichment %s: %v", enrichmentID, err)
			}
			return
		}

		log.Printf("Provider %s did not find phone number for enrichment %s, continuing...", provider.Name, enrichmentID)
	}

	// If we've checked all providers and none found the phone, mark as failed
	log.Printf("All providers checked for enrichment %s, no phone number found, marking as failed", enrichmentID)
	// Clear current provider when failed
	w.db.UpdateEnrichmentStatusWithProvider(enrichmentID, models.EnrichmentStatusFailed, nil, nil)
}
