FROM golang:1.24.2-alpine AS builder

# Set working directory
WORKDIR /app

# Install necessary dependencies for building
RUN apk add --no-cache git

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the application with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o raycast2api .

# Use a small alpine image for the final container
FROM alpine:3.17

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/raycast2api .

# Set environment variables (these will be overridden by docker-compose)
ENV PORT=8080
ENV RAYCAST_BEARER_TOKEN=""
ENV API_KEY=""

# Expose the service port
EXPOSE 8080

# Run the binary
CMD ["./raycast2api"]