# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS (needed for Kubernetes API)
RUN apk add --no-cache ca-certificates git

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the gamemaster binary
# CGO_ENABLED=0 for static binary, ldflags to strip debug info and set version
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o /gamemaster \
    ./cmd/gamemaster

# Runtime stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Copy the binary from builder
COPY --from=builder /gamemaster /gamemaster

# Use non-root user (distroless default is nonroot:65532)
USER 65532:65532

# The controller doesn't expose any ports itself
# Metrics/health endpoints can be added later

ENTRYPOINT ["/gamemaster"]
