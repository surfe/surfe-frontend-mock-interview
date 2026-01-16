package models

// Contact represents basic contact information
type Contact struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Company   string `json:"company,omitempty"`
	JobTitle  string `json:"jobTitle,omitempty"`
}

// EnrichmentStatus represents the possible states of an enrichment
type EnrichmentStatus string

const (
	EnrichmentStatusPending    EnrichmentStatus = "pending"
	EnrichmentStatusInProgress EnrichmentStatus = "in_progress"
	EnrichmentStatusCompleted  EnrichmentStatus = "completed"
	EnrichmentStatusFailed     EnrichmentStatus = "failed"
)

// JobStatus represents the status of a specific job (phone or email)
type JobStatus struct {
	CurrentProvider *Provider `json:"currentProvider,omitempty"`
	Result          string    `json:"result,omitempty"`
	Message         string    `json:"message,omitempty"`
	Pending         bool      `json:"pending"`
}

// Enrichment represents an enrichment process
type Enrichment struct {
	ID        string            `json:"id"`
	UserID    string            `json:"userId"`
	Status    EnrichmentStatus  `json:"status"`
	CreatedAt string            `json:"createdAt"`
	UpdatedAt string            `json:"updatedAt"`
	Result    *EnrichmentResult `json:"result,omitempty"`
	Phone     *JobStatus        `json:"phone,omitempty"`
	Email     *JobStatus        `json:"email,omitempty"`
}

// EnrichmentResult contains the enriched data
type EnrichmentResult struct {
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
}

// EnrichmentContactInfo contains optional third-party contact information
// If provided and matches the mock third-party data, success rate increases to 80%
type EnrichmentContactInfo struct {
	LinkedInURL    string   `json:"linkedInUrl,omitempty"`
	TwitterHandle  string   `json:"twitterHandle,omitempty"`
	GitHubUsername string   `json:"githubUsername,omitempty"`
	Bio            string   `json:"bio,omitempty"`
	Location       string   `json:"location,omitempty"`
	Skills         []string `json:"skills,omitempty"`
	Companies      []string `json:"companies,omitempty"`
}

// EnrichmentStartRequest is the payload for starting an enrichment
type EnrichmentStartRequest struct {
	UserID  string                 `json:"userId"`
	Jobs    []string               `json:"jobs,omitempty"`    // Array of "email" and/or "phone"
	Contact *EnrichmentContactInfo `json:"contact,omitempty"` // Optional contact info to boost success rate
}

// EnrichmentStartResponse is returned when an enrichment is started
type EnrichmentStartResponse struct {
	ID      string           `json:"id"`
	Status  EnrichmentStatus `json:"status"`
	Message string           `json:"message"`
}

// ThirdPartyInfo represents additional information from third-party sources
type ThirdPartyInfo struct {
	FullName       string   `json:"fullName"`
	LinkedInURL    string   `json:"linkedInUrl,omitempty"`
	TwitterHandle  string   `json:"twitterHandle,omitempty"`
	GitHubUsername string   `json:"githubUsername,omitempty"`
	Bio            string   `json:"bio,omitempty"`
	Location       string   `json:"location,omitempty"`
	Skills         []string `json:"skills,omitempty"`
	Companies      []string `json:"companies,omitempty"`
}

type Provider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl,omitempty"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}
