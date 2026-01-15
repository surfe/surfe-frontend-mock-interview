package data

import (
	"strings"
	"sync"

	"github.com/surfe/mock-api/internal/models"
)

// ============================================
// CONTACT UUIDs - Use these in your frontend
// ============================================
const (
	ContactJohnDoe       = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	ContactJaneSmith     = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	ContactBobJohnson    = "c3d4e5f6-a7b8-9012-cdef-123456789012"
	ContactAliceWilliams = "d4e5f6a7-b8c9-0123-def1-234567890123"
)

// MockData holds all the mock data for the API
// Edit this file to update mock responses
type MockData struct {
	mu         sync.RWMutex
	Contacts   map[string]models.Contact
	ThirdParty map[string]models.ThirdPartyInfo
}

// NewMockData initializes the mock data store with sample data
func NewMockData() *MockData {
	md := &MockData{
		Contacts:   make(map[string]models.Contact),
		ThirdParty: make(map[string]models.ThirdPartyInfo),
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

// GetThirdPartyInfo retrieves third-party info by full name (case-insensitive)
func (md *MockData) GetThirdPartyInfo(fullName string) (models.ThirdPartyInfo, bool) {
	md.mu.RLock()
	defer md.mu.RUnlock()
	info, exists := md.ThirdParty[strings.ToLower(fullName)]
	return info, exists
}
