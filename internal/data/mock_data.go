package data

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/surfe/mock-api/internal/models"
)

// ============================================
// CONTACT UUIDs - Use these in your frontend
// ============================================
const (
	ContactJohnDoe      = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	ContactJaneSmith    = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	ContactBobJohnson   = "c3d4e5f6-a7b8-9012-cdef-123456789012"
	ContactAliceWilliams = "d4e5f6a7-b8c9-0123-def1-234567890123"
)

// ============================================
// ENRICHMENT UUIDs - Use these in your frontend
// ============================================
const (
	EnrichmentPending    = "e5f6a7b8-c9d0-1234-ef12-345678901234"
	EnrichmentInProgress = "f6a7b8c9-d0e1-2345-f123-456789012345"
	EnrichmentCompleted  = "a7b8c9d0-e1f2-3456-0123-567890123456"
	EnrichmentFailed     = "b8c9d0e1-f2a3-4567-1234-678901234567"
)

// MockData holds all the mock data for the API
// Edit this file to update mock responses
type MockData struct {
	mu          sync.RWMutex
	Contacts    map[string]models.Contact
	Enrichments map[string]models.Enrichment
	ThirdParty  map[string]models.ThirdPartyInfo
}

// NewMockData initializes the mock data store with sample data
func NewMockData() *MockData {
	md := &MockData{
		Contacts:    make(map[string]models.Contact),
		Enrichments: make(map[string]models.Enrichment),
		ThirdParty:  make(map[string]models.ThirdPartyInfo),
	}

	// ============================================
	// CONTACTS - Edit here to add/modify contacts
	// ============================================
	md.Contacts[ContactJohnDoe] = models.Contact{
		ID:        ContactJohnDoe,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		Phone:     "+1-555-123-4567",
		Company:   "Acme Corp",
		JobTitle:  "Software Engineer",
	}

	md.Contacts[ContactJaneSmith] = models.Contact{
		ID:        ContactJaneSmith,
		FirstName: "Jane",
		LastName:  "Smith",
		Email:     "jane.smith@techco.io",
		Phone:     "+1-555-987-6543",
		Company:   "TechCo",
		JobTitle:  "Product Manager",
	}

	md.Contacts[ContactBobJohnson] = models.Contact{
		ID:        ContactBobJohnson,
		FirstName: "Bob",
		LastName:  "Johnson",
		Email:     "bob.johnson@startup.dev",
		Company:   "StartupDev",
		JobTitle:  "CTO",
	}

	md.Contacts[ContactAliceWilliams] = models.Contact{
		ID:        ContactAliceWilliams,
		FirstName: "Alice",
		LastName:  "Williams",
		Email:     "alice.w@bigcorp.com",
		Phone:     "+1-555-456-7890",
		Company:   "BigCorp Inc",
		JobTitle:  "Sales Director",
	}

	// ============================================
	// PRE-SEEDED ENRICHMENTS - Different states
	// ============================================
	now := time.Now().UTC()

	md.Enrichments[EnrichmentPending] = models.Enrichment{
		ID:        EnrichmentPending,
		UserID:    ContactJohnDoe,
		Status:    models.EnrichmentStatusPending,
		CreatedAt: now.Add(-10 * time.Minute).Format(time.RFC3339),
		UpdatedAt: now.Add(-10 * time.Minute).Format(time.RFC3339),
	}

	md.Enrichments[EnrichmentInProgress] = models.Enrichment{
		ID:        EnrichmentInProgress,
		UserID:    ContactJaneSmith,
		Status:    models.EnrichmentStatusInProgress,
		CreatedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
		UpdatedAt: now.Add(-2 * time.Minute).Format(time.RFC3339),
	}

	md.Enrichments[EnrichmentCompleted] = models.Enrichment{
		ID:        EnrichmentCompleted,
		UserID:    ContactBobJohnson,
		Status:    models.EnrichmentStatusCompleted,
		CreatedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
		UpdatedAt: now.Add(-55 * time.Minute).Format(time.RFC3339),
		Result: &models.EnrichmentResult{
			LinkedInURL: "https://linkedin.com/in/bobjohnson",
			TwitterURL:  "https://twitter.com/bob_johnson",
			Skills:      []string{"Leadership", "Architecture", "Go", "Kubernetes"},
			Experience:  12,
		},
	}

	md.Enrichments[EnrichmentFailed] = models.Enrichment{
		ID:        EnrichmentFailed,
		UserID:    ContactAliceWilliams,
		Status:    models.EnrichmentStatusFailed,
		CreatedAt: now.Add(-30 * time.Minute).Format(time.RFC3339),
		UpdatedAt: now.Add(-28 * time.Minute).Format(time.RFC3339),
	}

	// ============================================
	// THIRD PARTY INFO - Edit here to add/modify third-party data
	// Key is the full name (case-insensitive matching will be used)
	// ============================================
	md.ThirdParty["john doe"] = models.ThirdPartyInfo{
		FullName:       "John Doe",
		LinkedInURL:    "https://linkedin.com/in/johndoe",
		TwitterHandle:  "@johndoe_dev",
		GitHubUsername: "johndoe",
		Bio:            "Passionate software engineer with 10+ years of experience",
		Location:       "San Francisco, CA",
		Skills:         []string{"Go", "Python", "Kubernetes", "AWS"},
		Companies:      []string{"Acme Corp", "Google", "Meta"},
	}

	md.ThirdParty["jane smith"] = models.ThirdPartyInfo{
		FullName:       "Jane Smith",
		LinkedInURL:    "https://linkedin.com/in/janesmith",
		TwitterHandle:  "@janesmith_pm",
		GitHubUsername: "janesmith",
		Bio:            "Product leader focused on developer tools",
		Location:       "New York, NY",
		Skills:         []string{"Product Management", "Agile", "User Research"},
		Companies:      []string{"TechCo", "Stripe", "Shopify"},
	}

	md.ThirdParty["bob johnson"] = models.ThirdPartyInfo{
		FullName:       "Bob Johnson",
		LinkedInURL:    "https://linkedin.com/in/bobjohnson",
		GitHubUsername: "bobjohnson",
		Bio:            "Serial entrepreneur and tech leader",
		Location:       "Austin, TX",
		Skills:         []string{"Leadership", "Architecture", "Fundraising"},
		Companies:      []string{"StartupDev", "Oracle"},
	}

	md.ThirdParty["alice williams"] = models.ThirdPartyInfo{
		FullName:       "Alice Williams",
		LinkedInURL:    "https://linkedin.com/in/alicewilliams",
		TwitterHandle:  "@alice_sales",
		Bio:            "Enterprise sales expert with a track record of success",
		Location:       "Chicago, IL",
		Skills:         []string{"Enterprise Sales", "Negotiation", "CRM"},
		Companies:      []string{"BigCorp Inc", "Salesforce", "HubSpot"},
	}

	return md
}

// GetContact retrieves a contact by ID
func (md *MockData) GetContact(id string) (models.Contact, bool) {
	md.mu.RLock()
	defer md.mu.RUnlock()
	contact, exists := md.Contacts[id]
	return contact, exists
}

// GetEnrichment retrieves an enrichment by ID
func (md *MockData) GetEnrichment(id string) (models.Enrichment, bool) {
	md.mu.RLock()
	defer md.mu.RUnlock()
	enrichment, exists := md.Enrichments[id]
	return enrichment, exists
}

// CreateEnrichment creates a new enrichment and returns it
func (md *MockData) CreateEnrichment(userID string) models.Enrichment {
	md.mu.Lock()
	defer md.mu.Unlock()

	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	enrichment := models.Enrichment{
		ID:        id,
		UserID:    userID,
		Status:    models.EnrichmentStatusInProgress,
		CreatedAt: now,
		UpdatedAt: now,
	}

	md.Enrichments[id] = enrichment

	// Simulate async completion after a short delay
	go md.simulateEnrichmentCompletion(id)

	return enrichment
}

// simulateEnrichmentCompletion simulates the enrichment process completing
func (md *MockData) simulateEnrichmentCompletion(id string) {
	// Wait 3 seconds to simulate processing
	time.Sleep(3 * time.Second)

	md.mu.Lock()
	defer md.mu.Unlock()

	if enrichment, exists := md.Enrichments[id]; exists {
		enrichment.Status = models.EnrichmentStatusCompleted
		enrichment.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		enrichment.Result = &models.EnrichmentResult{
			LinkedInURL: "https://linkedin.com/in/enriched-user",
			TwitterURL:  "https://twitter.com/enriched_user",
			Skills:      []string{"Communication", "Problem Solving", "Leadership"},
			Experience:  5,
		}
		md.Enrichments[id] = enrichment
	}
}

// GetThirdPartyInfo retrieves third-party info by full name (case-insensitive)
func (md *MockData) GetThirdPartyInfo(fullName string) (models.ThirdPartyInfo, bool) {
	md.mu.RLock()
	defer md.mu.RUnlock()
	info, exists := md.ThirdParty[strings.ToLower(fullName)]
	return info, exists
}
