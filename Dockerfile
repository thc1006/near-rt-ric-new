# Multi-stage build for Nephio R5 / O-RAN L Release
FROM golang:1.24.6-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Enable FIPS mode for O-RAN security requirements
ENV GODEBUG=fips140=on

WORKDIR /build

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with security hardening
RUN CGO_ENABLED=1 GOOS=linux go build \
    -buildmode=pie \
    -ldflags="-linkmode external -extldflags -static-pie -s -w" \
    -tags netgo,osusergo \
    -o /ric ./cmd/ric

# Production image
FROM scratch

# Import the user and group files from the builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy SSL certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /ric /ric

# Create non-root user
USER nobody:nogroup

# Expose ports for O-RAN interfaces
EXPOSE 8080 38412 38422 36421 36422

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/ric", "health"]

# Run the application
ENTRYPOINT ["/ric"]