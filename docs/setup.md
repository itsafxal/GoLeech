# GoLeech Deployment & Setup Guide

## System Requirements
- **OS**: Linux (Debian, Ubuntu, Alpine, Arch), macOS, or Windows
- **Go**: 1.24+ (if building from source)
- **Memory**: 128 MB RAM minimum (baseline idle usage is ~40MB)
- **External Tools**:
  - `ffmpeg` & `ffprobe` (for video metadata, thumbnails, screenshots, and sample videos)
  - `aria2c` (for high-speed torrent and multi-connection downloads)
  - `p7zip` / `7z` (for archive extraction and compression)
  - `yt-dlp` (for video streaming site downloads)
  - `rclone` (optional, for 40+ cloud storage destinations)

---

## 1. Quick Start with Docker (Recommended)

### Clone the Repository
```bash
git clone https://github.com/GoLeech/GoLeech.git
cd GoLeech
```

### Configure Environment
```bash
cp .env.example .env
nano .env
```
Fill in your `BOT_TOKEN`, `TELEGRAM_API`, `TELEGRAM_HASH`, and `OWNER_ID`.

### Launch Container
```bash
docker compose up -d --build
```

View live logs:
```bash
docker compose logs -f
```

---

## 2. Bare-Metal / VPS Native Deployment

### 1. Install System Dependencies (Debian / Ubuntu)
```bash
sudo apt update && sudo apt install -y \
  git curl ffmpeg aria2 p7zip-full ca-certificates
```

Install `yt-dlp`:
```bash
sudo curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp
sudo chmod a+rx /usr/local/bin/yt-dlp
```

Install `rclone`:
```bash
sudo -v ; curl https://rclone.org/install.sh | sudo bash
```

### 2. Build GoLeech
```bash
git clone https://github.com/GoLeech/GoLeech.git
cd GoLeech
go mod download
go build -o bin/goleech ./cmd/goleech
```

### 3. Run the Bot
```bash
./bin/goleech
```

---

## 3. Web Dashboard & Streaming Access
Once launched, the embedded web server runs on port `8080` (or `WEB_PORT`):
- **Web Dashboard**: `http://localhost:8080/`
- **Healthcheck**: `http://localhost:8080/healthz`
- **JSON Stats API**: `http://localhost:8080/api/stats`
- **Task List API**: `http://localhost:8080/api/tasks`
- **Video Range Streaming**: `http://localhost:8080/stream/{gid}`

