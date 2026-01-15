# Build stage
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary for target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /server ./cmd/server

# Final stage - using scratch for minimal image
FROM scratch

# Copy the binary
COPY --from=builder /server /server

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/server"]
