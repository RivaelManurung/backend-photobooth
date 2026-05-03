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
RUN go build -o main .
RUN go build -o seeder ./cmd/seed/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binaries from builder
COPY --from=builder /app/main .
COPY --from=builder /app/seeder .
# Copy documentation and .env
COPY docs/ ./docs/
COPY .env.example .env

# Create uploads directory
RUN mkdir -p uploads

# Expose port
EXPOSE 8080

# Command to run
CMD ["./main"]
