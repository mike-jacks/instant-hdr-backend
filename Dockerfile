# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code explicitly to avoid Railway build context issues
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY docs/ ./docs/

# Build the application
RUN go build -o server ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/server .

# Expose port (Railway will set PORT env var)
EXPOSE 8080

# Run the binary
CMD ["./server"]

