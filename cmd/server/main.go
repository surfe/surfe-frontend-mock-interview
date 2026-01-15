package main

import (
	"log"
	"net/http"
	"os"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/surfe/mock-api/docs"
	"github.com/surfe/mock-api/internal/data"
	"github.com/surfe/mock-api/internal/handlers"
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
	// Initialize mock data
	mockData := data.NewMockData()

	// Initialize handlers
	h := handlers.NewHandler(mockData)

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

	// CORS middleware wrapper
	handler := corsMiddleware(mux)

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	log.Printf("Swagger docs available at http://localhost:%s/docs/", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
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
