# Build stage
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# git is already installed in the bookworm image

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o resetpwd ./cmd/resetpwd

# Final stage
FROM debian:stable-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

# Copy binaries from builder
COPY --from=builder /app/api .
COPY --from=builder /app/resetpwd .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/doc ./doc

# Expose port
EXPOSE 8080

# Run the application
CMD ["./api"]
