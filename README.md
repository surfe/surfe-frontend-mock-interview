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

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/contact/{id}` | Get contact by UUID |
| `POST` | `/enrichment/start` | Start a new enrichment |
| `GET` | `/enrichment/{id}` | Get enrichment status by UUID |
| `GET` | `/thirdparty/{full_name}` | Get third-party info by name |
| `GET` | `/health` | Health check |

---

## Available Test Data

### Contacts

| UUID | Name | Company | Job Title |
|------|------|---------|-----------|
| `a1b2c3d4-e5f6-7890-abcd-ef1234567890` | John Doe | Acme Corp | Software Engineer |
| `b2c3d4e5-f6a7-8901-bcde-f12345678901` | Jane Smith | TechCo | Product Manager |
| `c3d4e5f6-a7b8-9012-cdef-123456789012` | Bob Johnson | StartupDev | CTO |
| `d4e5f6a7-b8c9-0123-def1-234567890123` | Alice Williams | BigCorp Inc | Sales Director |

### Pre-seeded Enrichments (Different States)

| UUID | Status | User |
|------|--------|------|
| `e5f6a7b8-c9d0-1234-ef12-345678901234` | `pending` | John Doe |
| `f6a7b8c9-d0e1-2345-f123-456789012345` | `in_progress` | Jane Smith |
| `a7b8c9d0-e1f2-3456-0123-567890123456` | `completed` | Bob Johnson |
| `b8c9d0e1-f2a3-4567-1234-678901234567` | `failed` | Alice Williams |

### Third-Party Lookups (by Full Name)

| Full Name | Example URL |
|-----------|-------------|
| John Doe | `/thirdparty/John%20Doe` |
| Jane Smith | `/thirdparty/Jane%20Smith` |
| Bob Johnson | `/thirdparty/Bob%20Johnson` |
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

```bash
curl -X POST http://localhost:8080/enrichment/start \
  -H "Content-Type: application/json" \
  -d '{"userId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}'
```

Response:
```json
{
  "id": "generated-uuid-here",
  "status": "in_progress",
  "message": "Enrichment started successfully"
}
```

> **Note:** New enrichments start as `in_progress` and automatically transition to `completed` after 3 seconds.

### Get enrichment status (completed example)

```bash
curl http://localhost:8080/enrichment/a7b8c9d0-e1f2-3456-0123-567890123456
```

Response:
```json
{
  "id": "a7b8c9d0-e1f2-3456-0123-567890123456",
  "userId": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "status": "completed",
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-15T10:05:00Z",
  "result": {
    "linkedInUrl": "https://linkedin.com/in/bobjohnson",
    "twitterUrl": "https://twitter.com/bob_johnson",
    "skills": ["Leadership", "Architecture", "Go", "Kubernetes"],
    "experienceYears": 12
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

1. Returns immediately with status `in_progress`
2. After **3 seconds**, automatically transitions to `completed` with mock result data

Pre-seeded enrichments keep their static states (`pending`, `in_progress`, `completed`, `failed`) for testing different UI states.

---

## Modifying Mock Data

Edit `internal/data/mock_data.go` to:
- Add/remove contacts
- Change enrichment states
- Update third-party information

The file contains clearly marked sections for each data type.

---

## Project Structure

```
.
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── models/models.go         # Data structures
│   ├── data/mock_data.go        # Mock data (edit this!)
│   └── handlers/handlers.go     # HTTP handlers
├── docs/                        # Swagger documentation
├── Dockerfile                   # Multi-stage build (~10MB image)
├── docker-compose.yml
└── Makefile
```

---

## CORS

CORS is enabled for all origins (`*`), allowing requests from any frontend development server.
