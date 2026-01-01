# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /sumariza ./cmd/server

# Runtime image with Chrome
FROM alpine:3.19

# Install Chrome and dependencies
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont

# Set Chrome path for chromedp
ENV CHROME_PATH=/usr/bin/chromium-browser

WORKDIR /app

# Copy binary and assets
COPY --from=builder /sumariza .
COPY templates/ ./templates/
COPY static/ ./static/
COPY config/ ./config/

EXPOSE 3000

CMD ["./sumariza"]

