package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/surfe/mock-api/internal/data"
	"github.com/surfe/mock-api/internal/models"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	data *data.MockData
}

// NewHandler creates a new handler with the given mock data
func NewHandler(d *data.MockData) *Handler {
	return &Handler{data: d}
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

	enrichment := h.data.CreateEnrichment(req.UserID)

	response := models.EnrichmentStartResponse{
		ID:      enrichment.ID,
		Status:  enrichment.Status,
		Message: "Enrichment started successfully",
	}

	writeJSON(w, http.StatusCreated, response)
}

// GetEnrichment godoc
// @Summary      Get enrichment status
// @Description  Returns the status of the enrichment based on the ID
// @Tags         enrichment
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Enrichment ID"
// @Success      200  {object}  models.Enrichment
// @Failure      404  {object}  models.ErrorResponse
// @Router       /enrichment/{id} [get]
func (h *Handler) GetEnrichment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/enrichment/")
	if id == "" || id == "start" {
		writeError(w, http.StatusBadRequest, "missing enrichment ID")
		return
	}

	enrichment, exists := h.data.GetEnrichment(id)
	if !exists {
		writeError(w, http.StatusNotFound, "enrichment not found")
		return
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
