package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/surfe/mock-api/docs"
	"github.com/surfe/mock-api/internal/data"
	"github.com/surfe/mock-api/internal/database"
	"github.com/surfe/mock-api/internal/handlers"
	"github.com/surfe/mock-api/internal/worker"
)

// @title           Surfe Mock API
// @version         1.0
// @description     A mock API for frontend interview purposes
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    https://surfe.com
// @contact.email  support@surfe.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

func main() {
	// Initialize in-memory SQLite database (fresh on each restart)
	db, err := database.New(":memory:")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Seed static enrichments (always available after restart)
	if err := db.SeedStaticEnrichments(); err != nil {
		log.Fatalf("Failed to seed static enrichments: %v", err)
	}

	// Initialize mock data for contacts and third-party info
	mockData := data.NewMockData()

	// Initialize handlers
	h := handlers.NewHandler(mockData, db)

	// Start background worker for enrichment processing
	w := worker.New(db, mockData, worker.DefaultConfig())
	w.Start()

	// Setup routes
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/contact/", h.GetContact)
	mux.HandleFunc("/enrichment/start", h.StartEnrichment)
	mux.HandleFunc("/enrichment/", h.GetEnrichment)
	mux.HandleFunc("/thirdparty/", h.GetThirdPartyInfo)
	mux.HandleFunc("/health", h.HealthCheck)

	// Swagger documentation
	mux.HandleFunc("/docs/", httpSwagger.WrapHandler)

	// Apply middleware chain: logging -> CORS -> handler
	handler := loggingMiddleware(corsMiddleware(mux))

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		w.Stop()
		os.Exit(0)
	}()

	log.Printf("Starting server on :%s", port)
	log.Printf("Swagger docs available at http://localhost:%s/docs/", port)
	log.Println("Using in-memory database (data resets on restart, seed data always available)")

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs all incoming HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
