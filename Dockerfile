# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source
COPY . .

# Build the binary
RUN CGO_ENABLED=1 go build -o nexus-agent ./cmd/agent

# -------------------------------------------
# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies for SQLite
RUN apk add --no-cache ca-certificates libc6-compat

# Copy binary from builder
COPY --from=builder /app/nexus-agent .

# Create config directory
RUN mkdir -p /etc/nexus-agent

EXPOSE 9000

ENTRYPOINT ["./nexus-agent"]
CMD ["-config", "/etc/nexus-agent/config.yml"]
