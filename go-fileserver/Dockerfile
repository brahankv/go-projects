# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy source code
COPY main.go .

# Build the binary
# -o go-fileserver: output name
# CGO_ENABLED=0: static binary
RUN CGO_ENABLED=0 go build -o go-fileserver main.go

# Runtime Stage
FROM alpine:latest

WORKDIR /app

# Copy binary and static assets from builder
COPY --from=builder /app/go-fileserver .
COPY static ./static

# Expose the default port
EXPOSE 8080

# Create a volume for data to be served
VOLUME /data

# Run the server
# By default, serve the /data volume on port 8080
CMD ["./go-fileserver", "-folders", "/data", "-port", "8080"]
