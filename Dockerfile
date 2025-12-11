# Dockerfile for VPS Tools
# Multi-stage build for minimal final image

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
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" \
    -o vps-tools ./cmd/vps-tools

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata openssh-client curl

# Create non-root user
RUN addgroup -g 1001 -S vps-tools && \
    adduser -u 1001 -S vps-tools -G vps-tools

# Set working directory
WORKDIR /opt/vps-tools

# Create necessary directories
RUN mkdir -p /opt/vps-tools/{config,data,logs,backups,plugins} && \
    chown -R vps-tools:vps-tools /opt/vps-tools

# Copy binary from builder stage
COPY --from=builder /app/vps-tools /opt/vps-tools/vps-tools

# Copy configuration files
COPY --chown=vps-tools:vps-tools config/default.yaml /opt/vps-tools/config/

# Copy scripts
COPY --chown=vps-tools:vps-tools scripts/docker-entrypoint.sh /opt/vps-tools/
RUN chmod +x /opt/vps-tools/docker-entrypoint.sh

# Set permissions
RUN chmod +x /opt/vps-tools/vps-tools

# Switch to non-root user
USER vps-tools

# Expose volume mounts
VOLUME ["/opt/vps-tools/config", "/opt/vps-tools/data", "/opt/vps-tools/logs", "/opt/vps-tools/backups"]

# Set environment variables
ENV VPS_TOOLS_CONFIG="/opt/vps-tools/config/config.yaml"
ENV VPS_TOOLS_DATA_PATH="/opt/vps-tools/data"
ENV VPS_TOOLS_LOG_PATH="/opt/vps-tools/logs"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD /opt/vps-tools/vps-tools --version || exit 1

# Set entrypoint
ENTRYPOINT ["/opt/vps-tools/docker-entrypoint.sh"]

# Default command
CMD ["daemon"]

# Labels
LABEL maintainer="VPS Tools Team <team@vpstools.dev>" \
      version="${VERSION}" \
      description="VPS Tools - Modern VPS management suite" \
      org.opencontainers.image.title="VPS Tools" \
      org.opencontainers.image.description="Modern VPS management suite with CLI and TUI interfaces" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.source="https://github.com/pgd1001/vps-tools" \
      org.opencontainers.image.licenses="MIT"