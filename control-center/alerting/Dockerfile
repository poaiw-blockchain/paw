# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o alert-manager ./cmd/alert-manager

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 alertmanager && \
    adduser -D -u 1000 -G alertmanager alertmanager

# Set working directory
WORKDIR /app

# Copy binary from build stage
COPY --from=builder /build/alert-manager .

# Copy configuration template
COPY config.example.yaml /app/config.example.yaml

# Change ownership
RUN chown -R alertmanager:alertmanager /app

# Switch to non-root user
USER alertmanager

# Expose port
EXPOSE 11210

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:11210/health || exit 1

# Run the application
ENTRYPOINT ["/app/alert-manager"]
CMD ["-config", "/app/config.yaml"]
