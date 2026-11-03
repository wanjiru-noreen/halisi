# ------------------------------------------------------------------------------
# Stage 1: Build & Code Generation
# ------------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

# Install templ CLI
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate templ components and compile standalone binary
RUN templ generate
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/halisi-server ./cmd/api

# ------------------------------------------------------------------------------
# Stage 2: Minimal Non-Root Runtime
# ------------------------------------------------------------------------------
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create dedicated unprivileged user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary and static assets from builder stage
COPY --from=builder /build/halisi-server /app/halisi-server
COPY --from=builder /build/web/static /app/web/static

# Ensure unprivileged ownership
RUN chown -R appuser:appgroup /app

USER appuser:appgroup

EXPOSE 8080

CMD ["/app/halisi-server"]
