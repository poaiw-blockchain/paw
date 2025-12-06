# PAW Blockchain - Production Dockerfile
# Multi-stage build for Cosmos SDK + Node.js/TypeScript application

# ============================================================================
# Stage 1: Go Builder - Compile Cosmos SDK blockchain
# ============================================================================
FROM golang:1.24.4-alpine as go-builder

# Build arguments
ARG VERSION="v1.0.0"
ARG COMMIT="unknown"

# Install build dependencies
RUN apk add --no-cache \
     \
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
    -ldflags="-w -s -X example.com/cosmos/cosmos-sdk/version.AppName=paw -X example.com/cosmos/cosmos-sdk/version.Version=$VERSION -X example.com/cosmos/cosmos-sdk/version.Commit=$COMMIT" \
    -o paw-node ./cmd/pawd

# ============================================================================
# Stage 2: Runtime - Minimal production image
# ============================================================================
FROM alpine:3.18.9

# Metadata labels
LABEL maintainer="PAW Blockchain Team"
LABEL description="PAW Blockchain - Privacy-Aware Wallet Cosmos SDK Node"
LABEL version="1.0.0"
LABEL org.opencontainers.image.source="https://example.com/paw-chain/paw"

# Set environment variables
ENV PAW_ENV=production \
    PAW_HOME=/data \
    PAW_DATA_DIR=/data \
    PAW_LOG_DIR=/logs \
    NODE_ENV=production \
    PATH="/app/bin:$PATH" \
    PAW_P2P_PORT=26656 \
    PAW_RPC_PORT=26657 \
    PAW_API_PORT=1317 \
    PAW_GRPC_PORT=9090 \
    PAW_GRPC_WEB_PORT=9091 \
    NODE_API_PORT=5000

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    curl \
    libssl3 \
    libcrypto3 \
    tini \
    nodejs \
    npm

# Create non-root user
RUN addgroup -S paw -g 1000 && \
    adduser -S paw -G paw -u 1000 -h /home/paw && \
    mkdir -p /app /data /logs && \
    chown -R paw:paw /app /data /logs

# Set working directory
WORKDIR /app

# Copy Go binary
COPY --from=go-builder /build/paw-node /app/bin/

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
    CMD curl -f http://localhost:${PAW_API_PORT}/health || exit 1

# Expose ports
EXPOSE ${PAW_P2P_PORT} ${PAW_RPC_PORT} ${PAW_API_PORT} ${PAW_GRPC_PORT} ${PAW_GRPC_WEB_PORT} ${NODE_API_PORT}

# Volumes for persistent data
VOLUME ["/data", "/logs"]

# Use tini to handle signals
ENTRYPOINT ["/sbin/tini", "--"]

# Default command - start blockchain node
CMD ["/app/bin/paw-node", "start", "--home", "/data"]
