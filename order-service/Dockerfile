# Stage 1: Build
FROM golang:1.26rc1-alpine3.23 AS builder

WORKDIR /app

# Copy the entire project because we need the 'proto' folder too!
# In a real monorepo, context handling is key. 
# For simplicity, we assume we run build from the PROJECT ROOT.
COPY . .

# Build the order service
WORKDIR /app/order-service
RUN go build -o order-service .

# Stage 2: Run
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/order-service/order-service .

EXPOSE 50051
CMD ["./order-service"]