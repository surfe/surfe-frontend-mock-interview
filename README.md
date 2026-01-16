# Surfe Frontend Interview - Mock API

A lightweight Go mock API for frontend interview exercises. Provides endpoints for contact management, data enrichment, and third-party lookups.

## Quick Start

### Using Docker Compose (Recommended)

```bash
docker compose up --build
```

The API will be available at `http://localhost:8080`

### Using Go directly

```bash
go run ./cmd/server
```

### Using Make

```bash
make run          # Run locally
make docker-run   # Run with Docker Compose
```

### Docker compose

```bash
docker-compose up -d
```

## API Documentation

Swagger UI is available at: **http://localhost:8080/docs/**

## Endpoints

| Method | Endpoint                  | Description                   |
| ------ | ------------------------- | ----------------------------- |
| `GET`  | `/contact/{id}`           | Get contact by UUID           |
| `POST` | `/enrichment/start`       | Start a new enrichment        |
| `GET`  | `/enrichment/{id}`        | Get enrichment status by UUID |
| `GET`  | `/thirdparty/{full_name}` | Get third-party info by name  |
| `GET`  | `/health`                 | Health check                  |

---

## Available Test Data

### Contacts

| UUID                                   | Name           | Company     | Job Title         |
| -------------------------------------- | -------------- | ----------- | ----------------- |
| `a1b2c3d4-e5f6-7890-abcd-ef1234567890` | John Doe       | Acme Corp   | Software Engineer |
| `b2c3d4e5-f6a7-8901-bcde-f12345678901` | Jane Smith     | TechCo      | Product Manager   |
| `c3d4e5f6-a7b8-9012-cdef-123456789012` | Bob Johnson    | StartupDev  | CTO               |
| `d4e5f6a7-b8c9-0123-def1-234567890123` | Alice Williams | BigCorp Inc | Sales Director    |

### Pre-seeded Enrichments (Static - for testing UI states)

These enrichments **never change** and are always available for testing different UI states:

| UUID                                   | Status        | User           |
| -------------------------------------- | ------------- | -------------- |
| `e5f6a7b8-c9d0-1234-ef12-345678901234` | `pending`     | John Doe       |
| `f6a7b8c9-d0e1-2345-f123-456789012345` | `in_progress` | Jane Smith     |
| `a7b8c9d0-e1f2-3456-0123-567890123456` | `completed`   | Bob Johnson    |
| `b8c9d0e1-f2a3-4567-1234-678901234567` | `failed`      | Alice Williams |

### Providers

The enrichment system searches through 6 providers to find phone numbers and email addresses:

| UUID                                   | Name             |
| -------------------------------------- | ---------------- |
| `e5f6a7b8-c9d0-1234-efab-345678901234` | Acme Corp        |
| `f6a7b8c9-d0e1-2345-fabc-456789012345` | TechCo           |
| `a7b8c9d0-e1f2-3456-abcd-567890123456` | StartupDev       |
| `b8c9d0e1-f2a3-4567-bcde-678901234567` | BigCorp Inc      |
| `c9d0e1f2-a3b4-5678-cdef-789012345678` | CloudSync        |
| `d0e1f2a3-b4c5-6789-defa-890123456789` | DataFlow Systems |

Each provider:

- Takes 5 seconds ± 1 second (4-6 seconds) to respond
- Has a 20% chance of finding the requested value (phone or email)
- Is searched sequentially until a value is found or all providers are checked

### Third-Party Lookups (by Full Name)

| Full Name      | Example URL                    |
| -------------- | ------------------------------ |
| John Doe       | `/thirdparty/John%20Doe`       |
| Jane Smith     | `/thirdparty/Jane%20Smith`     |
| Bob Johnson    | `/thirdparty/Bob%20Johnson`    |
| Alice Williams | `/thirdparty/Alice%20Williams` |

---

## Example Requests

### Get a contact

```bash
curl http://localhost:8080/contact/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

Response:

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "firstName": "John",
  "lastName": "Doe",
  "email": "john.doe@example.com",
  "phone": "+1-555-123-4567",
  "company": "Acme Corp",
  "jobTitle": "Software Engineer"
}
```

### Start a new enrichment

You can start an enrichment for phone, email, or both by specifying the `jobs` array:

```bash
# Start enrichment for both phone and email
curl -X POST http://localhost:8080/enrichment/start \
  -H "Content-Type: application/json" \
  -d '{"userId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "jobs": ["phone", "email"]}'

# Start enrichment for phone only
curl -X POST http://localhost:8080/enrichment/start \
  -H "Content-Type: application/json" \
  -d '{"userId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "jobs": ["phone"]}'

# Start enrichment for email only
curl -X POST http://localhost:8080/enrichment/start \
  -H "Content-Type: application/json" \
  -d '{"userId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "jobs": ["email"]}'
```

Response:

```json
{
  "id": "generated-uuid-here",
  "status": "pending",
  "message": "Enrichment started successfully"
}
```

### Get enrichment status

```bash
curl http://localhost:8080/enrichment/a7b8c9d0-e1f2-3456-0123-567890123456
```

Response (completed example with both phone and email):

```json
{
  "id": "a7b8c9d0-e1f2-3456-0123-567890123456",
  "userId": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "status": "completed",
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-15T10:05:00Z",
  "result": {
    "phone": "+1-555-123-4567",
    "email": "john.doe@example.com"
  },
  "phone": {
    "currentProvider": null,
    "result": "+1-555-123-4567",
    "message": "Phone number found successfully",
    "pending": false
  },
  "email": {
    "currentProvider": null,
    "result": "john.doe@example.com",
    "message": "Email found successfully",
    "pending": false
  }
}
```

Response (in progress example):

```json
{
  "id": "abc-123",
  "userId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "in_progress",
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-15T10:02:00Z",
  "phone": {
    "currentProvider": {
      "id": "e5f6a7b8-c9d0-1234-efab-345678901234",
      "name": "Acme Corp",
      "imageUrl": "https://acme-corp.com/logo.png"
    },
    "message": "Searching for phone number...",
    "pending": true
  },
  "email": {
    "currentProvider": {
      "id": "f6a7b8c9-d0e1-2345-fabc-456789012345",
      "name": "TechCo",
      "imageUrl": "https://techco.io/logo.png"
    },
    "message": "Searching for email...",
    "pending": true
  }
}
```

Response (completed with one value not found):

```json
{
  "id": "abc-123",
  "userId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "completed",
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-15T10:05:00Z",
  "result": {
    "phone": "+1-555-123-4567",
    "email": ""
  },
  "phone": {
    "result": "+1-555-123-4567",
    "message": "Phone number found successfully",
    "pending": false
  },
  "email": {
    "result": "",
    "message": "Email not found after checking all providers",
    "pending": false
  }
}
```

### Get third-party information

```bash
curl http://localhost:8080/thirdparty/John%20Doe
```

Response:

```json
{
  "fullName": "John Doe",
  "linkedInUrl": "https://linkedin.com/in/johndoe",
  "twitterHandle": "@johndoe_dev",
  "githubUsername": "johndoe",
  "bio": "Passionate software engineer with 10+ years of experience",
  "location": "San Francisco, CA",
  "skills": ["Go", "Python", "Kubernetes", "AWS"],
  "companies": ["Acme Corp", "Google", "Meta"]
}
```

---

## Enrichment Status Flow

When you create a new enrichment via `POST /enrichment/start`:

```
pending ──> in_progress ──> completed
```

1. **Immediately**: Returns with status `pending`
2. **Worker starts**: Background worker begins processing the requested jobs (phone and/or email)
3. **In progress**: Each job searches through providers in parallel
4. **Completed**: All requested jobs finish (values found or set to empty string if not found)

### How Enrichment Works

- **Jobs**: You can request `["phone"]`, `["email"]`, or `["phone", "email"]`
- **Parallel Processing**: If both phone and email are requested, they run simultaneously
- **Provider Search**: Each job searches through all available providers (6 providers total)
- **Provider Timing**: Each provider takes 5 seconds ± 1 second (4-6 seconds) to respond
- **Success Rate**: Each provider has a 20% chance of finding the requested value
- **Completion**:
  - If a value is found, that job completes immediately
  - If a value is not found after checking all providers, it's set to an empty string
  - The enrichment is marked as `completed` when all requested jobs finish

### Response Structure

The `GET /enrichment/{id}` response includes separate objects for each requested job:

- **`phone`** (if requested): Contains `currentProvider`, `result`, `message`, and `pending`
- **`email`** (if requested): Contains `currentProvider`, `result`, `message`, and `pending`
- **`result`**: Contains the final `phone` and/or `email` values (may be empty strings if not found)

### Testing the flow

```bash
# 1. Start an enrichment for both phone and email
curl -X POST http://localhost:8080/enrichment/start \
  -H "Content-Type: application/json" \
  -d '{"userId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "jobs": ["phone", "email"]}'

# Response: {"id": "abc-123", "status": "pending", ...}

# 2. Poll for status changes
curl http://localhost:8080/enrichment/abc-123

# Status transitions:
# - "pending": Enrichment just started
# - "in_progress": Worker is searching through providers
# - "completed": All jobs finished (values found or set to empty string)
```

### Data persistence

- **In-memory database**: All dynamically created enrichments are lost on restart
- **Seed data always available**: The 4 static enrichments (pending, in_progress, completed, failed) are re-seeded on every startup

---

## Modifying Mock Data

Edit `internal/data/mock_data.go` to:

- Add/remove contacts
- Update third-party information

Edit `internal/database/database.go` (`SeedStaticEnrichments`) to:

- Change pre-seeded enrichment states

---

## Project Structure

```
.
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── models/models.go         # Data structures
│   ├── data/mock_data.go        # Contacts & third-party data
│   ├── database/database.go     # SQLite database layer
│   ├── handlers/handlers.go     # HTTP handlers
│   └── worker/worker.go         # Background enrichment processor
├── docs/                        # Swagger documentation
├── Dockerfile                   # Multi-stage build
├── docker-compose.yml
└── Makefile
```

---

## CORS

CORS is enabled for all origins (`*`), allowing requests from any frontend development server.
