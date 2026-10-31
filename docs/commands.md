# GoLeech Command Reference

GoLeech commands can be customized with an optional suffix (`CMD_SUFFIX`) in `.env` (e.g. `/mirror_bot` if `CMD_SUFFIX=_bot`).

---

## 1. Core Task Commands

| Command | Aliases | Description |
|---|---|---|
| `/mirror [link/flag]` | `/m` | Download media/files and upload to cloud storage (Rclone, Google Drive, Pixeldrain, Gofile). |
| `/leech [link/flag]` | `/l` | Download media/files and upload directly to Telegram as video or document. |
| `/status` | `/s`, `/statusall` | Display live interactive status message with pagination and progress bars. |
| `/cancel [gid]` | `/c [gid]`, `/c_<gid>` | Cancel a running or queued task by its GID. |
| `/cancelall [filter]` | | Cancel tasks in bulk. Filter can be `all`, `dl`, or `up`. |

---

## 2. Command Flags

Flags can be appended to `/mirror` or `/leech`:

```
/mirror https://example.com/movie.mp4 -n "Title.mp4" -doc -z -up "remote:backup"
```

| Flag | Argument | Description | Example |
|---|---|---|---|
| `-n` | `<name>` | Rename the file / custom filename. | `-n "MyVideo.mp4"` |
| `-doc` | None | Upload media explicitly as a Telegram Document. | `-doc` |
| `-med` | None | Upload media explicitly as a streamable Telegram Video. | `-med` |
| `-e` | None | Extract archive (`.zip`, `.rar`, `.7z`, `.tar.gz`) before uploading. | `-e` |
| `-z` | None | Compress files into a `.zip` archive before uploading. | `-z` |
| `-ss` | None | Generate screenshot thumbnails from video files. | `-ss` |
| `-sv` | None | Generate a 30-second sample preview video clip. | `-sv` |
| `-f` | None | Force start the task, bypassing active queue concurrency limits. | `-f` |
| `-sp` | `<size>` | Override file split size (e.g. `1.5G`, `2G`, `4G`). | `-sp 2G` |
| `-up` | `<dest>` | Cloud upload destination (e.g. `remote:path` or `pixeldrain`). | `-up "drive:movies"` |
| `-ud` | `<chat_id>` | Custom Telegram dump chat ID for leech upload. | `-ud "-10012345"` |
| `-h` | `<header>` | Custom HTTP Header for restricted downloads. | `-h "Authorization: Bearer xyz"` |
| `-au` / `-ap` | `<user>` / `<pass>` | HTTP Basic Authentication credentials. | `-au user -ap pass` |

---

## 3. Settings & Administration

| Command | Access | Description |
|---|---|---|
| `/start` | Public | Welcome banner and user role verification. |
| `/help` | Public | Interactive command overview and manual. |
| `/ping` | Public | Measure Telegram MTProto message round-trip latency. |
| `/stats` | Public | Real-time RAM, CPU, Goroutines, and task counters. |
| `/usetting` | Authorized | Configure individual user preferences (prefix, caption, leech mode). |
| `/bsetting` | Sudo | View runtime bot parameters and queue thresholds. |
| `/authorize [id]` | Sudo | Grant task execution permissions to a user or group chat. |
| `/unauthorize [id]` | Sudo | Revoke task execution permissions. |
| `/addsudo [id]` | Owner | Promote an authorized user to Sudo administrator. |
| `/rmsudo [id]` | Owner | Demote a Sudo administrator. |

