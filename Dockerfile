FROM golang:1.13-alpine AS builder

# Set necessary environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Create and set working directory
WORKDIR /build

# Install dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o zkhealth ./cmd/server

# Create a minimal production image
FROM alpine:3.11

# Add CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/zkhealth .

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app/zkhealth"]
