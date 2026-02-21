# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates git

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the CLI binary
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o /podsweeper \
    ./cmd/podsweeper

# Runtime stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Copy the binary from builder
COPY --from=builder /podsweeper /podsweeper

# Use non-root user
USER 65532:65532

ENTRYPOINT ["/podsweeper"]
