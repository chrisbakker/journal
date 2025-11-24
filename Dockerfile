# Build stage for frontend
FROM node:18-alpine AS frontend-builder

WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Build stage for backend
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /journal ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary and migrations
COPY --from=backend-builder /journal .
COPY --from=backend-builder /app/db/migrations ./db/migrations
COPY --from=frontend-builder /app/web/dist ./web/dist

# Create config directory
RUN mkdir -p /app/config

# Expose port
EXPOSE 8080

# Run the application
CMD ["./journal"]
