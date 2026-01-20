package handlers

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/surfe/mock-api/internal/data"
	"github.com/surfe/mock-api/internal/database"
	"github.com/surfe/mock-api/internal/models"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	data *data.MockData
	db   *database.DB
}

// NewHandler creates a new handler with the given mock data and database
func NewHandler(d *data.MockData, db *database.DB) *Handler {
	return &Handler{data: d, db: db}
}

// GetContacts godoc
// @Summary      Get all contacts
// @Description  Returns all available contacts from the database
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Success      200  {array}   models.Contact
// @Router       /contacts [get]
func (h *Handler) GetContacts(w http.ResponseWriter, r *http.Request) {
	contacts := h.data.GetAllContacts()
	writeJSON(w, http.StatusOK, contacts)
}

// GetContact godoc
// @Summary      Get contact by ID
// @Description  Returns the basic information around the contact based on their ID
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Contact ID"
// @Success      200  {object}  models.Contact
// @Failure      404  {object}  models.ErrorResponse
// @Router       /contact/{id} [get]
func (h *Handler) GetContact(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/contact/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing contact ID")
		return
	}

	contact, exists := h.data.GetContact(id)
	if !exists {
		writeError(w, http.StatusNotFound, "contact not found")
		return
	}

	writeJSON(w, http.StatusOK, contact)
}

// StartEnrichment godoc
// @Summary      Start an enrichment
// @Description  Starts an enrichment process, taking the userID and additional optional payload
// @Tags         enrichment
// @Accept       json
// @Produce      json
// @Param        request  body      models.EnrichmentStartRequest  true  "Enrichment request"
// @Success      201      {object}  models.EnrichmentStartResponse
// @Failure      400      {object}  models.ErrorResponse
// @Router       /enrichment/start [post]
func (h *Handler) StartEnrichment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.EnrichmentStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}

	// Validate jobs if provided
	jobs := req.Jobs
	if len(jobs) > 0 {
		validJobs := make([]string, 0, len(jobs))
		for _, job := range jobs {
			if job == "phone" || job == "email" {
				validJobs = append(validJobs, job)
			}
		}
		jobs = validJobs
		if len(jobs) == 0 {
			writeError(w, http.StatusBadRequest, "jobs must contain 'phone' and/or 'email'")
			return
		}
	}

	enrichment, err := h.db.CreateEnrichment(req.UserID, jobs, req.Contact)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create enrichment")
		return
	}

	response := models.EnrichmentStartResponse{
		ID:      enrichment.ID,
		Status:  enrichment.Status,
		Message: "Enrichment started successfully",
	}

	writeJSON(w, http.StatusCreated, response)
}

// GetEnrichment godoc
// @Summary      Get enrichment status
// @Description  Returns the status of the enrichment based on the enrichment ID
// @Tags         enrichment
// @Accept       json
// @Produce      json
// @Param        enrichmentId   path      string  true  "Enrichment ID"
// @Success      200  {object}  models.Enrichment
// @Failure      404  {object}  models.ErrorResponse
// @Router       /enrichment/{enrichmentId} [get]
func (h *Handler) GetEnrichment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/enrichment/")
	if id == "" || id == "start" {
		writeError(w, http.StatusBadRequest, "missing enrichment ID")
		return
	}

	enrichment, phoneProviderID, emailProviderID, err := h.db.GetEnrichmentWithProviders(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get enrichment")
		return
	}
	if enrichment == nil {
		writeError(w, http.StatusNotFound, "enrichment not found")
		return
	}

	// Get jobs and completed jobs to determine status
	jobs, completedJobs, err := h.db.GetEnrichmentJobs(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get enrichment jobs")
		return
	}

	// Populate Phone JobStatus
	phoneRequested := false
	phoneCompleted := false
	for _, job := range jobs {
		if job == "phone" {
			phoneRequested = true
			break
		}
	}
	for _, completed := range completedJobs {
		if completed == "phone" {
			phoneCompleted = true
			break
		}
	}

	if phoneRequested {
		phoneStatus := &models.JobStatus{
			Pending: !phoneCompleted,
		}

		// Set current provider
		if phoneProviderID != nil && *phoneProviderID != "" {
			provider, exists := h.data.GetProvider(*phoneProviderID)
			if exists {
				phoneStatus.CurrentProvider = &provider
			}
		}

		// Set result and message
		if phoneCompleted {
			if enrichment.Result != nil && enrichment.Result.Phone != "" {
				phoneStatus.Result = enrichment.Result.Phone
				phoneStatus.Message = "Phone number found successfully"
			} else {
				phoneStatus.Result = ""
				phoneStatus.Message = "Phone number not found after checking all providers"
			}
		} else {
			phoneStatus.Message = "Searching for phone number..."
		}

		enrichment.Phone = phoneStatus
	}

	// Populate Email JobStatus
	emailRequested := false
	emailCompleted := false
	for _, job := range jobs {
		if job == "email" {
			emailRequested = true
			break
		}
	}
	for _, completed := range completedJobs {
		if completed == "email" {
			emailCompleted = true
			break
		}
	}

	if emailRequested {
		emailStatus := &models.JobStatus{
			Pending: !emailCompleted,
		}

		// Set current provider
		if emailProviderID != nil && *emailProviderID != "" {
			provider, exists := h.data.GetProvider(*emailProviderID)
			if exists {
				emailStatus.CurrentProvider = &provider
			}
		}

		// Set result and message
		if emailCompleted {
			if enrichment.Result != nil && enrichment.Result.Email != "" {
				emailStatus.Result = enrichment.Result.Email
				emailStatus.Message = "Email found successfully"
			} else {
				emailStatus.Result = ""
				emailStatus.Message = "Email not found after checking all providers"
			}
		} else {
			emailStatus.Message = "Searching for email..."
		}

		enrichment.Email = emailStatus
	}

	writeJSON(w, http.StatusOK, enrichment)
}

// GetThirdPartyInfo godoc
// @Summary      Get third-party information
// @Description  Returns additional information about the user based on their full name
// @Tags         thirdparty
// @Accept       json
// @Produce      json
// @Param        full_name   path      string  true  "Full name (URL encoded)"
// @Success      200         {object}  models.ThirdPartyInfo
// @Failure      404         {object}  models.ErrorResponse
// @Router       /thirdparty/{full_name} [get]
func (h *Handler) GetThirdPartyInfo(w http.ResponseWriter, r *http.Request) {
	// Add artificial latency (500ms - 2000ms) to simulate real third-party API
	delay := 500 + rand.Intn(1500)
	time.Sleep(time.Duration(delay) * time.Millisecond)

	fullName := strings.TrimPrefix(r.URL.Path, "/thirdparty/")
	if fullName == "" {
		writeError(w, http.StatusBadRequest, "missing full name")
		return
	}

	// URL decode the full name
	decodedName, err := url.PathUnescape(fullName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid full name encoding")
		return
	}

	info, exists := h.data.GetThirdPartyInfo(decodedName)
	if !exists {
		writeError(w, http.StatusNotFound, "third-party information not found")
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// HealthCheck godoc
// @Summary      Health check
// @Description  Returns the health status of the API
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{
		Error:   http.StatusText(status),
		Code:    status,
		Message: message,
	})
}
