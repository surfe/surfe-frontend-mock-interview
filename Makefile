.PHONY: run build test docker-build docker-run clean

# Run the server locally
run:
	go run ./cmd/server

# Build the binary
build:
	go build -o bin/server ./cmd/server

# Run tests
test:
	go test -v ./...

# Build Docker image
docker-build:
	docker build -t surfe-mock-api .

# Run with Docker Compose
docker-run:
	docker compose up --build

# Run with Docker Compose in background
docker-up:
	docker compose up -d --build

# Stop Docker Compose
docker-down:
	docker compose down

# Clean build artifacts
clean:
	rm -rf bin/

# Download dependencies
deps:
	go mod download
	go mod tidy

# Generate Swagger docs (requires swag: go install github.com/swaggo/swag/cmd/swag@latest)
swagger:
	swag init -g cmd/server/main.go -o docs
