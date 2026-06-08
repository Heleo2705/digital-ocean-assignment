# Multi-stage Dockerfile for the main API (Go 1.25, Alpine)
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags='-s -w' -o /app/main ./cmd/main

FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates && update-ca-certificates
COPY --from=builder /app/main /app/main
EXPOSE 8000
ENTRYPOINT ["/app/main"]
