# =============================================================================
# technitium-companion - Multi-Stage Dockerfile
# =============================================================================
#
# Image Strategy:
#   :dev     - Development/integration testing (develop branch)
#   :edge    - Bleeding edge from main branch
#   :latest  - Latest stable release (version tags)
#   :vX.Y.Z  - Specific version
#   :sha-XXX - Specific commit for debugging
#
# Build commands:
#   docker build -t technitium-companion:latest .
#   docker build --platform linux/amd64,linux/arm64 -t technitium-companion:latest .
#
# Multi-arch support: amd64 + arm64
# =============================================================================

ARG GO_VERSION=1.24
ARG ALPINE_VERSION=3.20

# -----------------------------------------------------------------------------
# Stage 1: Go Builder (Multi-Arch Cross-Compilation)
# -----------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

# Build arguments for multi-arch support
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with cross-compilation for target architecture
# CGO_ENABLED=0 ensures pure Go build (no C dependencies)
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o technitium-companion \
    ./cmd/technitium-companion

# Verify binary
RUN ls -la technitium-companion && file technitium-companion || true

# -----------------------------------------------------------------------------
# Stage 2: Runtime (Alpine)
# -----------------------------------------------------------------------------
FROM alpine:${ALPINE_VERSION}

# Labels
LABEL org.opencontainers.image.title="technitium-companion" \
      org.opencontainers.image.description="Automatic DNS record management for Docker containers via Technitium" \
      org.opencontainers.image.source="https://gitlab.bluewillows.net/root/technitium-companion" \
      org.opencontainers.image.vendor="bluewillows.net"

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 companion && \
    adduser -u 1000 -G companion -s /bin/sh -D companion

# Copy binary from builder
COPY --from=builder /build/technitium-companion /usr/local/bin/technitium-companion

# Ensure binary is executable
RUN chmod +x /usr/local/bin/technitium-companion

# Default environment variables (can be overridden)
ENV TECHNITIUM_URL="" \
    TECHNITIUM_TOKEN="" \
    TECHNITIUM_ZONE="" \
    TARGET_IP="" \
    LOG_LEVEL="info" \
    HEALTH_PORT="8080"

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# Run as non-root user
# Note: When mounting Docker socket, ensure socket has appropriate permissions
# or run as root if needed for Docker API access
USER companion

# Expose health port
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/technitium-companion"]
