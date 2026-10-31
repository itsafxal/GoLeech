# ==============================================================================
# GoLeech Multi-Stage Dockerfile
# ==============================================================================

# Stage 1: Build the GoLeech binary
FROM golang:alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/goleech ./cmd/goleech
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/goleech-login ./cmd/login

# Stage 2: Minimal runtime image with essential media and download tools
FROM alpine:latest

WORKDIR /app

# Install runtime utilities (FFmpeg, Aria2, 7zip, yt-dlp, rclone, ca-certificates)
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    ffmpeg \
    aria2 \
    p7zip \
    curl \
    python3 \
    py3-pip \
    && curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp \
    && chmod a+rx /usr/local/bin/yt-dlp \
    && curl -O https://downloads.rclone.org/rclone-current-linux-amd64.zip \
    && unzip rclone-current-linux-amd64.zip \
    && cp rclone-*-linux-amd64/rclone /usr/local/bin/ \
    && chmod a+rx /usr/local/bin/rclone \
    && rm -rf rclone-*

COPY --from=builder /bin/goleech /usr/local/bin/goleech
COPY --from=builder /bin/goleech-login /usr/local/bin/goleech-login

VOLUME ["/app/downloads", "/app/data"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/goleech"]

