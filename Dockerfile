# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for potential dependencies
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Install openssh-keygen if needed, though Wish handles key generation
# ca-certificates for Firebase HTTPS connection
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/server .

# Expose the SSH port
EXPOSE 2324

# Set environment variables defaults (can be overridden)
ENV PORT=2324
ENV HOST=0.0.0.0

CMD ["./server"]
