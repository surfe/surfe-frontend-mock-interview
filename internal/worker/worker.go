package worker

import (
	"log"
	"time"

	"github.com/surfe/mock-api/internal/database"
	"github.com/surfe/mock-api/internal/models"
)

// Config holds worker configuration
type Config struct {
	// PollInterval is how often the worker checks for enrichments to process
	PollInterval time.Duration

	// PendingToInProgressDelay is how long an enrichment stays pending before moving to in_progress
	PendingToInProgressDelay time.Duration

	// InProgressToCompletedDelay is how long an enrichment stays in_progress before completing
	InProgressToCompletedDelay time.Duration
}

// DefaultConfig returns the default worker configuration
func DefaultConfig() Config {
	return Config{
		PollInterval:               10 * time.Second,
		PendingToInProgressDelay:   10 * time.Second,  // Move to in_progress after 10s
		InProgressToCompletedDelay: 50 * time.Second,  // Complete after another 50s (1 min total)
	}
}

// Worker processes enrichments in the background
type Worker struct {
	db     *database.DB
	config Config
	stopCh chan struct{}
}

// New creates a new background worker
func New(db *database.DB, config Config) *Worker {
	return &Worker{
		db:     db,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start begins the background processing loop
func (w *Worker) Start() {
	log.Printf("Starting enrichment worker (poll: %v, pending→in_progress: %v, in_progress→completed: %v)",
		w.config.PollInterval,
		w.config.PendingToInProgressDelay,
		w.config.InProgressToCompletedDelay,
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

	// Process in_progress → completed
	w.processInProgressEnrichments()
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
	}
}

func (w *Worker) processInProgressEnrichments() {
	enrichments, err := w.db.GetInProgressEnrichments(w.config.InProgressToCompletedDelay)
	if err != nil {
		log.Printf("Error fetching in_progress enrichments: %v", err)
		return
	}

	for _, e := range enrichments {
		log.Printf("Completing enrichment %s", e.ID)

		// Generate mock result data
		result := &models.EnrichmentResult{
			LinkedInURL: "https://linkedin.com/in/enriched-user",
			TwitterURL:  "https://twitter.com/enriched_user",
			Skills:      []string{"Communication", "Problem Solving", "Leadership", "Go", "React"},
			Experience:  5,
		}

		err := w.db.UpdateEnrichmentStatus(e.ID, models.EnrichmentStatusCompleted, result)
		if err != nil {
			log.Printf("Error completing enrichment %s: %v", e.ID, err)
			continue
		}
	}
}
