# PAW Blockchain - Production Dockerfile
# Multi-stage build for Cosmos SDK + Node.js/TypeScript application

# ============================================================================
# Stage 1: Go Builder - Compile Cosmos SDK blockchain
# ============================================================================
FROM golang:1.23.1-alpine as go-builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    linux-headers

# Set working directory
WORKDIR /build

# Copy go modules
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build blockchain binary with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -X github.com/cosmos/cosmos-sdk/version.AppName=paw" \
    -o paw-node ./cmd/pawd/main.go

# Build API server
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s" \
    -o paw-api ./api/cmd/main.go

# ============================================================================
# Stage 2: Node.js Builder - Build frontend and API
# ============================================================================
FROM node:18-alpine as node-builder

# Install build dependencies
RUN apk add --no-cache python3 make g++

# Set working directory
WORKDIR /build

# Copy package files
COPY package.json package-lock.json ./

# Install dependencies
RUN npm ci --only=production

# Copy source
COPY external ./external

# Build frontend (if exists)
RUN if [ -d "./external/frontend" ]; then \
    cd external/frontend && npm run build; \
    fi

# ============================================================================
# Stage 3: Runtime - Minimal production image
# ============================================================================
FROM alpine:3.18

# Metadata labels
LABEL maintainer="PAW Blockchain Team"
LABEL description="PAW Blockchain - Privacy-Aware Wallet Cosmos SDK Node"
LABEL version="1.0.0"
LABEL org.opencontainers.image.source="https://github.com/paw-chain/paw"

# Set environment variables
ENV PAW_ENV=production \
    PAW_HOME=/app \
    PAW_DATA_DIR=/data \
    PAW_LOG_DIR=/logs \
    NODE_ENV=production \
    PATH="/app/bin:$PATH"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    curl \
    libssl3 \
    libcrypto3 \
    tini \
    node \
    npm

# Create non-root user
RUN addgroup -S paw -g 1000 && \
    adduser -S paw -G paw -u 1000 -h /home/paw && \
    mkdir -p /app /data /logs && \
    chown -R paw:paw /app /data /logs

# Set working directory
WORKDIR /app

# Copy Go binaries
COPY --from=go-builder /build/paw-node /build/paw-api /app/bin/

# Copy Node.js dependencies and code
COPY --from=node-builder /build/node_modules /app/node_modules
COPY --from=node-builder /build/external /app/external

# Copy configuration
COPY --chown=paw:paw config/ ./config/
COPY --chown=paw:paw docs/ ./docs/

# Create data directories
RUN mkdir -p \
    /data/blockchain \
    /data/state \
    /logs/node \
    /logs/api && \
    chown -R paw:paw /data /logs /app

# Switch to non-root user
USER paw

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:${PAW_API_PORT:-26657}/health || exit 1

# Expose ports
# 26656 - P2P network
# 26657 - RPC API
# 1317  - REST API
# 9090  - gRPC
# 9091  - gRPC-Web
# 5000  - Node.js API server
EXPOSE 26656 26657 1317 9090 9091 5000

# Volumes for persistent data
VOLUME ["/data", "/logs"]

# Use tini to handle signals
ENTRYPOINT ["/sbin/tini", "--"]

# Default command - start blockchain node
CMD ["/app/bin/paw-node", "start"]
