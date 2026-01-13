# syntax=docker/dockerfile:1
FROM golang:1.24 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/paw-indexer ./cmd

FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=builder /bin/paw-indexer /bin/paw-indexer

USER nonroot:nonroot
EXPOSE 18080 19090

ENTRYPOINT ["/bin/paw-indexer", "-config", "/app/config/config.yaml"]
