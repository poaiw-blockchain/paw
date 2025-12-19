# syntax=docker/dockerfile:1.6
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o status-page

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/status-page /app/status-page
EXPOSE 11090
ENTRYPOINT ["/app/status-page"]
