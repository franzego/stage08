FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o wallet-service .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary
COPY --from=builder /app/wallet-service .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/swagger.yaml ./swagger.yaml

EXPOSE 8080

CMD ["./wallet-service"]
