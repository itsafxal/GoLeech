# Changelog

All notable changes to **GoLeech** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-10-31

### Added
- **Core Architecture:**
  - Complete native Go implementation powered by Gogram MTProto framework (`amarnathcjd/gogram`).
  - Thread-safe priority task and queue manager with fine-grained concurrency control (`queue_all`, `queue_download`, `queue_upload`).
  - Anti-flood dynamic status updater with pagination and status filters (`ALL`, `DL`, `UP`).
- **Download Providers:**
  - Multi-connection HTTP/HTTPS streaming downloader with Range headers and resume support.
  - Direct link resolvers for Pixeldrain, Mediafire, Buzzheavier, GitHub releases, etc.
  - Telegram MTProto media downloader with 4 parallel worker threads.
  - BitTorrent & magnet link downloader via Aria2 JSON-RPC.
  - `yt-dlp` video downloader with real-time newline progress parsing.
- **Upload Providers:**
  - Telegram Leech uploader with automated 2GB/4GB file splitting, FFmpeg thumbnail generation, video streaming metadata, and custom captions.
  - Rclone cloud storage uploader for 40+ providers (Google Drive, OneDrive, Mega, S3, WebDAV).
  - DDL multipart uphosters (Pixeldrain, Gofile, Buzzheavier).
- **Media & File Operations:**
  - Archive extraction (`-e`) and creation (`-z`) with ZipSlip path sanitization.
  - FFmpeg screenshot thumbnail generator (`-ss`) and 30-second sample video generator (`-sv`).
  - Video and audio metadata extraction via `ffprobe`.
- **Session Management:**
  - Native Gogram session generator CLI tool (`cmd/login`) supporting Phone, OTP, and 2FA cloud password directly with Telegram servers.
- **Persistence & DB:**
  - Embedded pure Go SQLite persistence layer (`modernc.org/sqlite`) in WAL mode.
- **Web Interface:**
  - Modern, responsive HTML5 dashboard.
  - Authenticated REST APIs for tasks, stats, and healthcheck (`/healthz`).
  - HTTP Range media streaming endpoint (`/stream/{gid}`).
- **Automation:**
  - Background RSS monitor with regex filtering and automatic task dispatching.
  - Extensible plugin architecture with lifecycle hooks and command registration.
- **Deployment & QA:**
  - Multi-stage Dockerfile and Docker Compose configuration.
  - Comprehensive unit test suite with 100% pass rate.
  - GitHub Actions CI workflow.

