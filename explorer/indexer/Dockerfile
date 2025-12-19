# syntax=docker/dockerfile:1

FROM golang:1.24 AS builder
WORKDIR /src

# Preload module metadata
COPY go.mod go.sum go.work go.work.sum ./
RUN go mod download

# Copy the full source (indexer depends on shared modules)
COPY . .

# Build the indexer binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/paw-indexer ./explorer/indexer/cmd

FROM gcr.io/distroless/base-debian12
WORKDIR /app

# Copy binary and staging config
COPY --from=builder /bin/paw-indexer /bin/paw-indexer
COPY explorer/indexer/config ./config

USER nonroot:nonroot
EXPOSE 8080 9100

ENTRYPOINT ["/bin/paw-indexer", "-config", "/app/config/staging.yaml"]
