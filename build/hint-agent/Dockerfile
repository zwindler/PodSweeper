# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS (needed at runtime)
RUN apk add --no-cache ca-certificates

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the hint-agent binary
# CGO_ENABLED=0 for static binary, ldflags to strip debug info and set version
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o /hint-agent \
    ./cmd/hint-agent

# Runtime stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Copy the binary from builder
COPY --from=builder /hint-agent /hint-agent

# Use non-root user (distroless default is nonroot:65532)
USER 65532:65532

# Expose the default port
EXPOSE 8080

# Health check (note: distroless doesn't have curl, k8s probes handle this)
# HEALTHCHECK handled by Kubernetes liveness/readiness probes

ENTRYPOINT ["/hint-agent"]
