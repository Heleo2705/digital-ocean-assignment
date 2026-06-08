# Multi-stage Dockerfile for the main API
FROM golang:1.21 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o /app/main ./cmd/main

FROM scratch
COPY --from=builder /app/main /app/main
EXPOSE 8000
ENTRYPOINT ["/app/main"]
