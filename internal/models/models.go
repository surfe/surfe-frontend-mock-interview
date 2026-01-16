package models

// Contact represents basic contact information
type Contact struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone,omitempty"`
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

// Enrichment represents an enrichment process
type Enrichment struct {
	ID              string            `json:"id"`
	UserID          string            `json:"userId"`
	Status          EnrichmentStatus  `json:"status"`
	CreatedAt       string            `json:"createdAt"`
	UpdatedAt       string            `json:"updatedAt"`
	Result          *EnrichmentResult `json:"result,omitempty"`
	CurrentProvider *Provider         `json:"currentProvider,omitempty"`
}

// EnrichmentResult contains the enriched data
type EnrichmentResult struct {
	LinkedInURL string   `json:"linkedInUrl,omitempty"`
	TwitterURL  string   `json:"twitterUrl,omitempty"`
	Skills      []string `json:"skills,omitempty"`
	Experience  int      `json:"experienceYears,omitempty"`
	Phone       string   `json:"phone,omitempty"`
}

// EnrichmentStartRequest is the payload for starting an enrichment
type EnrichmentStartRequest struct {
	UserID  string                 `json:"userId"`
	Options map[string]interface{} `json:"options,omitempty"`
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
