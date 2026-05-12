# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -trimpath -ldflags="-s -w" -o main .
RUN go build -trimpath -ldflags="-s -w" -o worker ./cmd/worker
RUN go build -trimpath -ldflags="-s -w" -o seeder ./cmd/seed/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S photobooth && \
    adduser -S photobooth -G photobooth

# Copy the binaries from builder
COPY --from=builder /app/main .
COPY --from=builder /app/worker .
COPY --from=builder /app/seeder .
# Copy documentation and .env
COPY docs/ ./docs/
COPY .env.example .env

# Create uploads directory
RUN mkdir -p uploads && chown -R photobooth:photobooth /app

USER photobooth

# Expose port
EXPOSE 8082

# Command to run
CMD ["./main"]
