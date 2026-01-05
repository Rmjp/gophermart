# Stage 1: Build the application
FROM golang:1.26rc1-alpine3.23 AS builder

WORKDIR /app

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN go build -o api-gateway .

# Stage 2: Run the application
FROM alpine:latest

WORKDIR /app

# Copy only the binary from the builder stage
COPY --from=builder /app/api-gateway .

# Expose the port
EXPOSE 8080

# Run the binary
CMD ["./api-gateway"]