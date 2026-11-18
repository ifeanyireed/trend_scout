# Multi-stage Docker build for TrendScout Studio Go + PostgreSQL Backend
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /trendscout-server ./cmd/server

# Lightweight production container
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /trendscout-server /app/trendscout-server

ENV PORT=4000
ENV STORAGE_DIR=/app/storage
EXPOSE 4000

CMD ["/app/trendscout-server"]
