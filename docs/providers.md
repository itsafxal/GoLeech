# GoLeech Providers Guide

GoLeech uses a decoupled provider architecture with unified progress callbacks and context-based cancellation.

---

## 1. Download Providers

### A. Direct HTTP / HTTPS (`internal/downloader/http`)
- **Engine Name:** `HTTP`
- **Capabilities:**
  - Multi-connection chunked downloading with HTTP `Range` headers.
  - Resume interrupted downloads.
  - Custom HTTP headers (`-h`) and Basic Auth (`-au`, `-ap`).
  - Automatic `Content-Disposition` filename parsing.
- **Direct Link Resolvers:**
  - Integrated bypass for Pixeldrain (`/u/ID`), Mediafire, Buzzheavier, GitHub releases, and direct CDNs.

### B. Telegram MTProto (`internal/downloader/telegram`)
- **Engine Name:** `Telegram`
- **Capabilities:**
  - Direct MTProto binary chunk streaming via Gogram.
  - Download documents, videos, audios, and photos from private or group chats.
  - Up to 4 concurrent worker threads per file transmission.

### C. Torrents & Magnet Links (`internal/downloader/torrent`)
- **Engine Name:** `Torrent`
- **Capabilities:**
  - High-speed BitTorrent downloading via Aria2 JSON-RPC daemon.
  - Magnet URI resolution and `.torrent` file support.
  - Real-time peer, seeder, and leecher stats.
  - Selective file downloading (`-s` flag).

### D. Streaming Video Sites (`internal/downloader/ytdlp`)
- **Engine Name:** `yt-dlp`
- **Capabilities:**
  - Invokes `yt-dlp` CLI with real-time newline JSON progress parsing.
  - Supports 1,000+ video hosting services (YouTube, Instagram, TikTok, Twitter/X, Twitch, Facebook, Reddit).
  - Best video + best audio auto-merge into `.mp4` or `.mkv`.

---

## 2. Upload Providers

### A. Telegram Leech (`internal/providers/telegram.go`)
- **Destination:** Telegram Chat / Log Channel / User PM
- **Capabilities:**
  - Automated 2GB (Bot) and 4GB (Telegram Premium User session) file splitting.
  - Video thumbnail generation with FFmpeg.
  - Fast-start metadata injection for instant Telegram streaming (`DocumentAttributeVideo`).
  - Dynamic user-customizable captions, prefixes, and suffixes.

### B. Rclone Cloud Storage (`internal/providers/rclone.go`)
- **Destination:** 40+ Cloud Providers (Google Drive, OneDrive, Mega, S3, WebDAV, Dropbox, etc.)
- **Capabilities:**
  - Direct upload using configured Rclone remotes.
  - Real-time percentage, transfer speed, and ETA monitoring.
  - Configurable custom transfer flags in `.env` (`RCLONE_FLAGS`).

### C. DDL File Hosts (`internal/providers/uphoster.go`)
- **Destination:** Pixeldrain, Gofile, Buzzheavier
- **Capabilities:**
  - Fast multipart streaming uploads directly to public file hosts.
  - Immediate return of public shareable download URLs.

