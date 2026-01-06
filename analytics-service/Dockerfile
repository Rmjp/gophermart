FROM golang:1.26rc1-alpine3.23 AS builder
WORKDIR /app
COPY . .
RUN go build -o analytics-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/analytics-service .
CMD ["./analytics-service"]