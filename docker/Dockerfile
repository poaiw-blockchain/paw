# Multi-stage Dockerfile for PAW Blockchain Node
# Stage 1: Build the application
FROM golang:1.23.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    linux-headers \
    ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo \
    -ldflags="-w -s -X github.com/cosmos/cosmos-sdk/version.Name=paw \
    -X github.com/cosmos/cosmos-sdk/version.AppName=pawd \
    -X github.com/cosmos/cosmos-sdk/version.Version=$(git describe --tags --always) \
    -X github.com/cosmos/cosmos-sdk/version.Commit=$(git rev-parse HEAD) \
    -X github.com/cosmos/cosmos-sdk/version.BuildTags=netgo,ledger" \
    -tags "netgo,ledger" \
    -o /build/bin/pawd ./cmd/pawd

# Verify the binary
RUN /build/bin/pawd version

# Stage 2: Runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    bash \
    curl \
    jq \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1000 paw && \
    adduser -D -u 1000 -G paw paw

# Copy binary from builder
COPY --from=builder /build/bin/pawd /usr/local/bin/pawd

# Copy entrypoint script
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create directories and set permissions
RUN mkdir -p /paw/.paw/config /paw/.paw/data /paw/.paw/wasm && \
    chown -R paw:paw /paw

# Switch to non-root user
USER paw

# Set working directory
WORKDIR /paw

# Expose ports
# 26656: P2P
# 26657: RPC
# 26660: Prometheus metrics
# 9090: gRPC
# 1317: REST API
# 5000: Custom API server
EXPOSE 26656 26657 26660 9090 1317 5000

# Set environment variables
ENV PATH="/usr/local/bin:$PATH"
ENV HOME="/paw"
ENV CHAIN_ID="paw-1"
ENV MONIKER="paw-node"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:26657/health || exit 1

# Use entrypoint script
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# Default command
CMD ["start"]
