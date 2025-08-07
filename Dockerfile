# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o logfiend .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN adduser -D -s /bin/sh logfiend

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/logfiend .

# Copy example configs
COPY --from=builder /app/examples ./examples/
COPY --from=builder /app/config.yml ./

# Change ownership
RUN chown -R logfiend:logfiend /app

# Switch to non-root user
USER logfiend

# Expose any ports if needed (none for this CLI tool)

# Set entrypoint
ENTRYPOINT ["./logfiend"]

# Default command
CMD ["-config=config.yml"]
