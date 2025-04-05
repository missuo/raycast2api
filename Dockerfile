FROM golang:1.24.2-alpine AS builder

# Set working directory
WORKDIR /go/src/github.com/missuo/raycast2api

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the application with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o raycast2api .

# Use a small alpine image for the final container
FROM alpine:latest
WORKDIR /app
# Copy the binary from the builder stage
COPY --from=builder /go/src/github.com/missuo/raycast2api/raycast2api .
# Expose the service port
EXPOSE 8080
# Run the binary
CMD ["/app/raycast2api"]