package config

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config represents application configuration
type Config struct {
	BotToken             string  `json:"bot_token"`
	TelegramAPI          int32   `json:"telegram_api"`
	TelegramHash         string  `json:"telegram_hash"`
	OwnerID              int64   `json:"owner_id"`
	UserSessionString    string  `json:"user_session_string"`
	DatabasePath         string  `json:"database_path"`
	DownloadDir          string  `json:"download_dir"`
	SudoUsers            []int64 `json:"sudo_users"`
	AuthorizedChats      []int64 `json:"authorized_chats"`
	StatusLimit          int     `json:"status_limit"`
	StatusUpdateInterval int     `json:"status_update_interval"`
	QueueAll             int     `json:"queue_all"`
	QueueDownload        int     `json:"queue_download"`
	QueueUpload          int     `json:"queue_upload"`
	TorrentLimit         int64   `json:"torrent_limit"`
	DirectLimit          int64   `json:"direct_limit"`
	StorageLimit         int64   `json:"storage_limit"`
	WebPort              int     `json:"web_port"`
	WebAccessPassword    string  `json:"web_access_password"`
	DefaultUpload        string  `json:"default_upload"`
	LeechSplitSize       int64   `json:"leech_split_size"`
	LeechPrefix          string  `json:"leech_prefix"`
	LeechSuffix          string  `json:"leech_suffix"`
	LeechCaption         string  `json:"leech_caption"`
	LeechLogChat         int64   `json:"leech_log_chat"`
	RclonePath           string  `json:"rclone_path"`
	RcloneFlags          string  `json:"rclone_flags"`
	RcloneServeURL       string  `json:"rclone_serve_url"`
	RssDelay             int     `json:"rss_delay"`
	CmdSuffix            string  `json:"cmd_suffix"`
	LeechAsDocument      bool    `json:"leech_as_document"`
	DisableLeech         bool    `json:"disable_leech"`
	DisableMirror        bool    `json:"disable_mirror"`
	DisableTorrents      bool    `json:"disable_torrents"`
	DisableRSS           bool    `json:"disable_rss"`
	DisableSearch        bool    `json:"disable_search"`
}

var (
	GlobalConfig *Config
	configMutex  sync.RWMutex
)

// LoadConfig loads configuration from environment variables and optional config files
func LoadConfig() *Config {
	loadDotEnv(".env")

	cfg := &Config{
		DownloadDir:          "downloads",
		DatabasePath:         "goleech.db",
		StatusLimit:          10,
		StatusUpdateInterval: 10,
		WebPort:              8080,
		DefaultUpload:        "tg",
		LeechAsDocument:      true,
		LeechSplitSize:       2097152000, // 2GB - 2MB safety margin
		RssDelay:             600,
	}

	// Try reading JSON config if present
	if data, err := os.ReadFile("configs/config.json"); err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	// Override from Environment Variables
	if val := os.Getenv("BOT_TOKEN"); val != "" {
		cfg.BotToken = val
	}
	if val := os.Getenv("TELEGRAM_API"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.TelegramAPI = int32(i)
		}
	}
	if val := os.Getenv("TELEGRAM_HASH"); val != "" {
		cfg.TelegramHash = val
	}
	if val := os.Getenv("OWNER_ID"); val != "" {
		if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.OwnerID = id
		}
	}
	if val := os.Getenv("USER_SESSION_STRING"); val != "" {
		cfg.UserSessionString = val
	}
	if val := os.Getenv("DATABASE_PATH"); val != "" {
		cfg.DatabasePath = val
	}
	if val := os.Getenv("DOWNLOAD_DIR"); val != "" {
		cfg.DownloadDir = val
	}
	if val := os.Getenv("SUDO_USERS"); val != "" {
		cfg.SudoUsers = parseIntList(val)
	}
	if val := os.Getenv("AUTHORIZED_CHATS"); val != "" {
		cfg.AuthorizedChats = parseIntList(val)
	}
	if val := os.Getenv("STATUS_LIMIT"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.StatusLimit = i
		}
	}
	if val := os.Getenv("STATUS_UPDATE_INTERVAL"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.StatusUpdateInterval = i
		}
	}
	if val := os.Getenv("QUEUE_ALL"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.QueueAll = i
		}
	}
	if val := os.Getenv("QUEUE_DOWNLOAD"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.QueueDownload = i
		}
	}
	if val := os.Getenv("QUEUE_UPLOAD"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.QueueUpload = i
		}
	}
	if val := os.Getenv("WEB_PORT"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.WebPort = i
		}
	}
	if val := os.Getenv("WEB_ACCESS_PASSWORD"); val != "" {
		cfg.WebAccessPassword = val
	}
	if val := os.Getenv("DEFAULT_UPLOAD"); val != "" {
		cfg.DefaultUpload = strings.ToLower(val)
	}
	if val := os.Getenv("LEECH_SPLIT_SIZE"); val != "" {
		if s, err := strconv.ParseInt(val, 10, 64); err == nil && s > 0 {
			cfg.LeechSplitSize = s
		}
	}
	if val := os.Getenv("LEECH_LOG_CHAT"); val != "" {
		if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.LeechLogChat = id
		}
	}
	if val := os.Getenv("RCLONE_PATH"); val != "" {
		cfg.RclonePath = val
	}
	if val := os.Getenv("RCLONE_FLAGS"); val != "" {
		cfg.RcloneFlags = val
	}
	if val := os.Getenv("RCLONE_SERVE_URL"); val != "" {
		cfg.RcloneServeURL = val
	}
	if val := os.Getenv("CMD_SUFFIX"); val != "" {
		cfg.CmdSuffix = val
	}
	if val := os.Getenv("LEECH_AS_DOCUMENT"); val != "" {
		cfg.LeechAsDocument = (val != "false" && val != "0" && val != "no")
	} else if val := os.Getenv("AS_DOCUMENT"); val != "" {
		cfg.LeechAsDocument = (val != "false" && val != "0" && val != "no")
	}

	configMutex.Lock()
	GlobalConfig = cfg
	configMutex.Unlock()

	return cfg
}

// Get returns the active global configuration safely
func Get() *Config {
	configMutex.RLock()
	if GlobalConfig != nil {
		cfg := GlobalConfig
		configMutex.RUnlock()
		return cfg
	}
	configMutex.RUnlock()

	return LoadConfig()
}

func parseIntList(s string) []int64 {
	var list []int64
	for _, part := range strings.Split(s, " ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if id, err := strconv.ParseInt(part, 10, 64); err == nil {
			list = append(list, id)
		}
	}
	return list
}

func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}
