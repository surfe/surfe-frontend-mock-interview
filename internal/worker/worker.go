package worker

import (
	"log"
	"math/rand"
	"strings"
	"sync"
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

	// ProviderSuccessRate is the probability (0.0 to 1.0) that a provider will find the requested value
	ProviderSuccessRate float32
}

// DefaultConfig returns the default worker configuration
func DefaultConfig() Config {
	return Config{
		PollInterval:             10 * time.Second,
		PendingToInProgressDelay: 10 * time.Second, // Move to in_progress after 10s
		ProviderSuccessRate:      0.2,              // 20% chance of finding the value
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
// Runs phone and email jobs in parallel if both are requested.
func (w *Worker) processEnrichmentThroughProviders(enrichmentID, userID string) {
	// Get the contact
	contact, exists := w.mockData.GetContact(userID)
	if !exists {
		log.Printf("Contact not found for enrichment %s (userID: %s), marking as failed", enrichmentID, userID)
		w.db.UpdateEnrichmentStatus(enrichmentID, models.EnrichmentStatusFailed, nil)
		return
	}

	// Get the jobs for this enrichment
	jobs, _, err := w.db.GetEnrichmentJobs(enrichmentID)
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

	// Get contact info and check if it matches third-party data to boost success rate
	contactInfo, err := w.db.GetEnrichmentContactInfo(enrichmentID)
	if err != nil {
		log.Printf("Error getting contact info for enrichment %s: %v", enrichmentID, err)
	}

	// Calculate success rate: 80% if contact info matches, otherwise use base rate
	successRate := w.config.ProviderSuccessRate
	if contactInfo != nil {
		// Get the full name from contact
		fullName := contact.FirstName + " " + contact.LastName
		thirdPartyInfo, exists := w.mockData.GetThirdPartyInfo(fullName)
		if exists && w.contactInfoMatches(contactInfo, &thirdPartyInfo) {
			successRate = 0.8 // 80% success rate when contact info matches
			log.Printf("✅ SUCCESS RATE BOOSTED: Contact info matches third-party data for enrichment %s (user: %s). Success rate increased from %.0f%% to 80%%", enrichmentID, fullName, w.config.ProviderSuccessRate*100)
		} else {
			log.Printf("Contact info provided for enrichment %s (user: %s) but does not match third-party data. Using base success rate of %.0f%%", enrichmentID, fullName, w.config.ProviderSuccessRate*100)
		}
	}

	log.Printf("Processing enrichment %s through %d providers for jobs: %v (success rate: %.0f%%)", enrichmentID, len(providers), jobs, successRate*100)

	// Use WaitGroup to wait for all job types to complete
	var wg sync.WaitGroup

	// Process phone job in parallel if needed
	if needsPhone {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.processJobForEnrichment(enrichmentID, contact, "phone", providers, successRate)
		}()
	}

	// Process email job in parallel if needed
	if needsEmail {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.processJobForEnrichment(enrichmentID, contact, "email", providers, successRate)
		}()
	}

	// Wait for all jobs to complete
	wg.Wait()

	// Final check: ensure all requested jobs have values (set to empty string if not found)
	_, completedJobs, err := w.db.GetEnrichmentJobs(enrichmentID)
	if err != nil {
		log.Printf("Error getting completed jobs for enrichment %s: %v", enrichmentID, err)
		return
	}

	// Set missing values to empty strings and mark as completed
	// Each job type updates only its own field using UpdateEnrichmentResultField
	if needsPhone {
		phoneCompleted := false
		for _, completed := range completedJobs {
			if completed == "phone" {
				phoneCompleted = true
				break
			}
		}
		if !phoneCompleted {
			// Phone was requested but not found, set to empty string using field-specific update
			log.Printf("Phone job not found after checking all providers, setting to empty string for enrichment %s", enrichmentID)
			if err := w.db.UpdateEnrichmentResultField(enrichmentID, "phone", ""); err != nil {
				log.Printf("Error setting empty phone for enrichment %s: %v", enrichmentID, err)
			}
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
			// Email was requested but not found, set to empty string using field-specific update
			log.Printf("Email job not found after checking all providers, setting to empty string for enrichment %s", enrichmentID)
			if err := w.db.UpdateEnrichmentResultField(enrichmentID, "email", ""); err != nil {
				log.Printf("Error setting empty email for enrichment %s: %v", enrichmentID, err)
			}
			// Mark as completed (even though empty)
			if err := w.db.AddCompletedJob(enrichmentID, "email"); err != nil {
				log.Printf("Error marking email as completed for enrichment %s: %v", enrichmentID, err)
			}
		}
	}

	// Complete the enrichment - status will be set to completed by AddCompletedJob when all jobs are done
	// No need to update result here since each job type already updated its own field using UpdateEnrichmentResultField
	enrichment, err := w.db.GetEnrichment(enrichmentID)
	if err != nil {
		log.Printf("Error checking enrichment %s status: %v", enrichmentID, err)
		return
	}

	// If not already completed, it will be completed by AddCompletedJob when the last job finishes
	if enrichment != nil && enrichment.Status != models.EnrichmentStatusCompleted {
		log.Printf("All providers checked for enrichment %s, waiting for final job completion", enrichmentID)
	}
}

// contactInfoMatches checks if the provided contact info matches the third-party data
func (w *Worker) contactInfoMatches(contactInfo *models.EnrichmentContactInfo, thirdPartyInfo *models.ThirdPartyInfo) bool {
	// Compare all fields (case-insensitive for strings, order-independent for slices)
	if contactInfo.LinkedInURL != "" && contactInfo.LinkedInURL != thirdPartyInfo.LinkedInURL {
		return false
	}
	if contactInfo.TwitterHandle != "" && contactInfo.TwitterHandle != thirdPartyInfo.TwitterHandle {
		return false
	}
	if contactInfo.GitHubUsername != "" && contactInfo.GitHubUsername != thirdPartyInfo.GitHubUsername {
		return false
	}
	if contactInfo.Bio != "" && contactInfo.Bio != thirdPartyInfo.Bio {
		return false
	}
	if contactInfo.Location != "" && contactInfo.Location != thirdPartyInfo.Location {
		return false
	}

	// Compare skills (order-independent)
	if len(contactInfo.Skills) > 0 {
		if !w.stringSlicesEqual(contactInfo.Skills, thirdPartyInfo.Skills) {
			return false
		}
	}

	// Compare companies (order-independent)
	if len(contactInfo.Companies) > 0 {
		if !w.stringSlicesEqual(contactInfo.Companies, thirdPartyInfo.Companies) {
			return false
		}
	}

	return true
}

// stringSlicesEqual checks if two string slices contain the same elements (order-independent)
func (w *Worker) stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps for comparison
	mapA := make(map[string]int)
	mapB := make(map[string]int)

	for _, s := range a {
		mapA[strings.ToLower(s)]++
	}
	for _, s := range b {
		mapB[strings.ToLower(s)]++
	}

	// Compare maps
	if len(mapA) != len(mapB) {
		return false
	}

	for k, v := range mapA {
		if mapB[k] != v {
			return false
		}
	}

	return true
}

// processJobForEnrichment processes a single job type (phone or email) through providers
// for a given enrichment. Runs independently and can complete while other jobs continue.
func (w *Worker) processJobForEnrichment(enrichmentID string, contact models.Contact, jobType string, providers []models.Provider, successRate float32) {
	log.Printf("Starting %s job processing for enrichment %s", jobType, enrichmentID)

	// Process through each provider
	for _, provider := range providers {
		// Check if enrichment was already completed or failed
		enrichment, err := w.db.GetEnrichment(enrichmentID)
		if err != nil {
			log.Printf("Error checking enrichment %s status: %v", enrichmentID, err)
			return
		}
		if enrichment == nil || enrichment.Status != models.EnrichmentStatusInProgress {
			// Check if this specific job is already completed
			_, completedJobs, err := w.db.GetEnrichmentJobs(enrichmentID)
			if err == nil {
				for _, completed := range completedJobs {
					if completed == jobType {
						log.Printf("%s job already completed for enrichment %s", jobType, enrichmentID)
						return
					}
				}
			}
			if enrichment == nil || enrichment.Status == models.EnrichmentStatusFailed {
				log.Printf("Enrichment %s is failed, stopping %s job processing", enrichmentID, jobType)
				return
			}
		}

		// Check if this job is already completed
		_, completedJobs, err := w.db.GetEnrichmentJobs(enrichmentID)
		if err != nil {
			log.Printf("Error getting completed jobs for enrichment %s: %v", enrichmentID, err)
			return
		}

		jobCompleted := false
		for _, completed := range completedJobs {
			if completed == jobType {
				jobCompleted = true
				break
			}
		}

		if jobCompleted {
			log.Printf("%s job already completed for enrichment %s", jobType, enrichmentID)
			return
		}

		log.Printf("Checking provider %s for %s job in enrichment %s", provider.Name, jobType, enrichmentID)

		// Update the current provider being processed for this specific job type
		if err := w.db.UpdateEnrichmentStatusWithJobProvider(enrichmentID, models.EnrichmentStatusInProgress, nil, &provider.ID, jobType); err != nil {
			log.Printf("Error updating current provider for enrichment %s: %v", enrichmentID, err)
		}

		// Wait 5 seconds ± 1 second (4-6 seconds)
		delay := 4*time.Second + time.Duration(rand.Intn(2001))*time.Millisecond
		time.Sleep(delay)

		// Check if this provider finds the requested data
		found := false
		if rand.Float32() < successRate {
			found = true
			var value string
			// Get enrichment data (phone/email that can be found)
			enrichmentPhone, enrichmentEmail, hasData := w.mockData.GetEnrichmentData(contact.ID)

			if jobType == "phone" {
				if hasData && enrichmentPhone != "" {
					value = enrichmentPhone
					log.Printf("Provider %s found phone number for enrichment %s", provider.Name, enrichmentID)
				} else {
					log.Printf("Provider %s would have found phone but no phone data available", provider.Name)
					found = false
				}
			} else if jobType == "email" {
				if hasData && enrichmentEmail != "" {
					value = enrichmentEmail
					log.Printf("Provider %s found email for enrichment %s", provider.Name, enrichmentID)
				} else {
					log.Printf("Provider %s would have found email but no email data available", provider.Name)
					found = false
				}
			}

			if found && value != "" {
				// Update only this job type's field in the result (preserves the other field)
				if err := w.db.UpdateEnrichmentResultField(enrichmentID, jobType, value); err != nil {
					log.Printf("Error updating %s result for enrichment %s: %v", jobType, enrichmentID, err)
					continue
				}

				// Update the provider ID for this job type
				if err := w.db.UpdateEnrichmentStatusWithJobProvider(enrichmentID, models.EnrichmentStatusInProgress, nil, &provider.ID, jobType); err != nil {
					log.Printf("Error updating provider for enrichment %s: %v", enrichmentID, err)
					continue
				}

				// Update contact with found value
				if jobType == "phone" {
					if err := w.mockData.UpdateContactPhone(contact.ID, value); err != nil {
						log.Printf("Error updating contact phone for enrichment %s: %v", enrichmentID, err)
					} else {
						log.Printf("Updated contact %s with phone number: %s", contact.ID, value)
					}
				} else if jobType == "email" {
					if err := w.mockData.UpdateContactEmail(contact.ID, value); err != nil {
						log.Printf("Error updating contact email for enrichment %s: %v", enrichmentID, err)
					} else {
						log.Printf("Updated contact %s with email: %s", contact.ID, value)
					}
				}

				// Mark job as completed (after result and contact are updated)
				// This will read the latest result from DB, so it should have our value
				if err := w.db.AddCompletedJob(enrichmentID, jobType); err != nil {
					log.Printf("Error marking %s as completed for enrichment %s: %v", jobType, enrichmentID, err)
					continue
				}

				// Clear provider ID since job is completed (result is already saved)
				if err := w.db.UpdateEnrichmentStatusWithJobProvider(enrichmentID, models.EnrichmentStatusInProgress, nil, nil, jobType); err != nil {
					log.Printf("Error clearing provider for completed %s job in enrichment %s: %v", jobType, enrichmentID, err)
				}

				log.Printf("%s job completed for enrichment %s by provider %s", jobType, enrichmentID, provider.Name)
				return // Job found, stop processing this job type
			}
		} else {
			log.Printf("Provider %s did not find %s for enrichment %s, continuing...", provider.Name, jobType, enrichmentID)
		}
	}

	// If we've checked all providers and didn't find the value, the job will be marked as completed
	// with empty string in the main function
	log.Printf("All providers checked for %s job in enrichment %s, value not found", jobType, enrichmentID)
}
