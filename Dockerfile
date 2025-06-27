# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o kube-state-logs .

# Final stage
FROM alpine:3.18

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 kube-state-logs && \
    adduser -D -s /bin/sh -u 1000 -G kube-state-logs kube-state-logs

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/kube-state-logs .

# Change ownership to non-root user
RUN chown -R kube-state-logs:kube-state-logs /app

# Switch to non-root user
USER kube-state-logs

# Expose port (if needed for health checks)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run the application
ENTRYPOINT ["./kube-state-logs"] 