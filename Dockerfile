# Multi-stage Dockerfile for mcp-serve
# Produces a minimal ~20MB final image

# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# -ldflags="-s -w" strips debug info for smaller binary
# CGO_ENABLED=0 for static linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.VERSION=${VERSION:-1.0.0}" \
    -o mcp-serve \
    .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 mcp && \
    adduser -D -u 1000 -G mcp mcp

# Create directories
RUN mkdir -p /app/agents && \
    chown -R mcp:mcp /app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/mcp-serve /app/mcp-serve

# Copy example agents (optional)
COPY --chown=mcp:mcp agents/*.yaml /app/agents/

# Switch to non-root user
USER mcp

# Expose port for SSE transport
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/mcp-serve", "-version"]

# Default command (stdio mode)
ENTRYPOINT ["/app/mcp-serve"]
CMD ["-agents", "/app/agents", "-transport", "stdio"]
