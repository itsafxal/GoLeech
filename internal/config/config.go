package config

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config represents the application's centralized strongly-typed configuration
type Config struct {
	// ==========================================
	// 1. Essential Bot Configuration (Required)
	// ==========================================
	BotToken     string `json:"bot_token"`
	TelegramAPI  int32  `json:"telegram_api"`
	TelegramHash string `json:"telegram_hash"`
	OwnerID      int64  `json:"owner_id"`
	MongoURI     string `json:"mongo_uri"`

	// ==========================================
	// 2. Core Application Settings
	// ==========================================
	BotName         string  `json:"bot_name"`
	SudoUsers       []int64 `json:"sudo_users"`
	AuthorizedChats []int64 `json:"authorized_chats"`
	CmdSuffix       string  `json:"cmd_suffix"`
	Timezone        string  `json:"timezone"`
	LogLevel        string  `json:"log_level"`

	// ==========================================
	// 3. Database Configuration
	// ==========================================
	DatabaseName  string `json:"database_name"`
	DatabasePath  string `json:"database_path"`
	DBMaxPoolSize uint64 `json:"db_max_pool_size"`
	DBMinPoolSize uint64 `json:"db_min_pool_size"`
	DBTimeout     int    `json:"db_timeout"`

	// ==========================================
	// 4. Telegram & MTProto Settings
	// ==========================================
	UserSessionString    string `json:"user_session_string"`
	LeechSplitSize       int64  `json:"leech_split_size"`
	LeechLogChat         int64  `json:"leech_log_chat"`
	LeechAsDocument      bool   `json:"leech_as_document"`
	LeechPrefix          string `json:"leech_prefix"`
	LeechSuffix          string `json:"leech_suffix"`
	LeechCaption         string `json:"leech_caption"`
	StatusLimit          int    `json:"status_limit"`
	StatusUpdateInterval int    `json:"status_update_interval"`

	// ==========================================
	// 5. File Transfer & Storage
	// ==========================================
	DownloadDir   string `json:"download_dir"`
	DefaultUpload string `json:"default_upload"`
	QueueAll      int    `json:"queue_all"`
	QueueDownload int    `json:"queue_download"`
	QueueUpload   int    `json:"queue_upload"`
	TorrentLimit  int64  `json:"torrent_limit"`
	DirectLimit   int64  `json:"direct_limit"`
	StorageLimit  int64  `json:"storage_limit"`

	// ==========================================
	// 6. Optional Integrations & External Services
	// ==========================================
	WebPort           int    `json:"web_port"`
	WebAccessPassword string `json:"web_access_password"`
	RclonePath        string `json:"rclone_path"`
	RcloneFlags       string `json:"rclone_flags"`
	RcloneServeURL    string `json:"rclone_serve_url"`
	RssDelay          int    `json:"rss_delay"`

	// ==========================================
	// 7. Advanced & Feature Toggles
	// ==========================================
	Debug           bool `json:"debug"`
	DisableLeech    bool `json:"disable_leech"`
	DisableMirror   bool `json:"disable_mirror"`
	DisableTorrents bool `json:"disable_torrents"`
	DisableRSS      bool `json:"disable_rss"`
	DisableSearch   bool `json:"disable_search"`
}

var (
	GlobalConfig *Config
	configMutex  sync.RWMutex
)

// Validate checks for presence of mandatory startup credentials
func (c *Config) Validate() []string {
	var missing []string
	if c.BotToken == "" {
		missing = append(missing, "BOT_TOKEN")
	}
	if c.TelegramAPI == 0 {
		missing = append(missing, "TELEGRAM_API / API_ID")
	}
	if c.TelegramHash == "" {
		missing = append(missing, "TELEGRAM_HASH / API_HASH")
	}
	if c.OwnerID == 0 {
		missing = append(missing, "OWNER_ID")
	}
	return missing
}

// LoadConfig loads configuration from environment variables and root .env file
func LoadConfig() *Config {
	loadDotEnv(".env")

	cfg := &Config{
		// Sensible defaults
		BotName:              "GoLeech",
		Timezone:             "UTC",
		LogLevel:             "INFO",
		DatabaseName:         "goleech",
		DatabasePath:         "goleech.db",
		DBMaxPoolSize:        50,
		DBMinPoolSize:        5,
		DBTimeout:            10,
		LeechSplitSize:       2097152000, // 2GB
		LeechAsDocument:      true,
		StatusLimit:          10,
		StatusUpdateInterval: 10,
		DownloadDir:          "downloads",
		DefaultUpload:        "tg",
		QueueAll:             10,
		QueueDownload:        5,
		QueueUpload:          5,
		WebPort:              8080,
		RclonePath:           "remote:backup",
		RcloneFlags:          "--transfers=4 --checkers=8",
		RssDelay:             600,
	}

	// Try reading JSON config if present as fallback
	if data, err := os.ReadFile("configs/config.json"); err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	// 1. Essential Bot Configuration (Required)
	if val := getEnv("BOT_TOKEN"); val != "" {
		cfg.BotToken = val
	}
	if val := getEnv("TELEGRAM_API", "API_ID"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.TelegramAPI = int32(i)
		}
	}
	if val := getEnv("TELEGRAM_HASH", "API_HASH"); val != "" {
		cfg.TelegramHash = val
	}
	if val := getEnv("OWNER_ID"); val != "" {
		if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.OwnerID = id
		}
	}
	if val := getEnv("MONGO_URI", "DATABASE_URL", "MONGODB_URI"); val != "" {
		cfg.MongoURI = val
	}

	// 2. Core Application Settings
	if val := getEnv("BOT_NAME"); val != "" {
		cfg.BotName = val
	}
	if val := getEnv("SUDO_USERS"); val != "" {
		cfg.SudoUsers = parseIntList(val)
	}
	if val := getEnv("AUTHORIZED_CHATS"); val != "" {
		cfg.AuthorizedChats = parseIntList(val)
	}
	if val := getEnv("CMD_SUFFIX"); val != "" {
		cfg.CmdSuffix = val
	}
	if val := getEnv("TIMEZONE"); val != "" {
		cfg.Timezone = val
	}
	if val := getEnv("LOG_LEVEL"); val != "" {
		cfg.LogLevel = strings.ToUpper(val)
	}

	// 3. Database Configuration
	if val := getEnv("DATABASE_NAME"); val != "" {
		cfg.DatabaseName = val
	}
	if val := getEnv("DATABASE_PATH"); val != "" {
		cfg.DatabasePath = val
	}
	if val := getEnv("DB_MAX_POOL_SIZE"); val != "" {
		cfg.DBMaxPoolSize = parseUint64(val, cfg.DBMaxPoolSize)
	}
	if val := getEnv("DB_MIN_POOL_SIZE"); val != "" {
		cfg.DBMinPoolSize = parseUint64(val, cfg.DBMinPoolSize)
	}
	if val := getEnv("DB_TIMEOUT"); val != "" {
		cfg.DBTimeout = parseInt(val, cfg.DBTimeout)
	}

	// 4. Telegram & MTProto Settings
	if val := getEnv("USER_SESSION_STRING", "SESSION_STRING"); val != "" {
		cfg.UserSessionString = val
	}
	if val := getEnv("LEECH_SPLIT_SIZE", "TG_UPLOAD_LIMIT"); val != "" {
		cfg.LeechSplitSize = parseInt64(val, cfg.LeechSplitSize)
	}
	if val := getEnv("LEECH_LOG_CHAT"); val != "" {
		cfg.LeechLogChat = parseInt64(val, cfg.LeechLogChat)
	}
	if val := getEnv("LEECH_AS_DOCUMENT", "AS_DOCUMENT"); val != "" {
		cfg.LeechAsDocument = parseBool(val, true)
	}
	if val := getEnv("LEECH_PREFIX", "DEFAULT_PREFIX"); val != "" {
		cfg.LeechPrefix = val
	}
	if val := getEnv("LEECH_SUFFIX", "DEFAULT_SUFFIX"); val != "" {
		cfg.LeechSuffix = val
	}
	if val := getEnv("LEECH_CAPTION", "DEFAULT_CAPTION"); val != "" {
		cfg.LeechCaption = val
	}
	if val := getEnv("STATUS_LIMIT"); val != "" {
		cfg.StatusLimit = parseInt(val, cfg.StatusLimit)
	}
	if val := getEnv("STATUS_UPDATE_INTERVAL"); val != "" {
		cfg.StatusUpdateInterval = parseInt(val, cfg.StatusUpdateInterval)
	}

	// 5. File Transfer & Storage
	if val := getEnv("DOWNLOAD_DIR", "TEMP_DIR"); val != "" {
		cfg.DownloadDir = val
	}
	if val := getEnv("DEFAULT_UPLOAD"); val != "" {
		cfg.DefaultUpload = strings.ToLower(val)
	}
	if val := getEnv("QUEUE_ALL"); val != "" {
		cfg.QueueAll = parseInt(val, cfg.QueueAll)
	}
	if val := getEnv("QUEUE_DOWNLOAD", "MAX_CONCURRENT_DOWNLOADS"); val != "" {
		cfg.QueueDownload = parseInt(val, cfg.QueueDownload)
	}
	if val := getEnv("QUEUE_UPLOAD", "MAX_CONCURRENT_UPLOADS"); val != "" {
		cfg.QueueUpload = parseInt(val, cfg.QueueUpload)
	}
	if val := getEnv("TORRENT_LIMIT"); val != "" {
		cfg.TorrentLimit = parseInt64(val, cfg.TorrentLimit)
	}
	if val := getEnv("DIRECT_LIMIT"); val != "" {
		cfg.DirectLimit = parseInt64(val, cfg.DirectLimit)
	}
	if val := getEnv("STORAGE_LIMIT"); val != "" {
		cfg.StorageLimit = parseInt64(val, cfg.StorageLimit)
	}

	// 6. Optional Integrations & External Services
	if val := getEnv("WEB_PORT"); val != "" {
		cfg.WebPort = parseInt(val, cfg.WebPort)
	}
	if val := getEnv("WEB_ACCESS_PASSWORD"); val != "" {
		cfg.WebAccessPassword = val
	}
	if val := getEnv("RCLONE_PATH"); val != "" {
		cfg.RclonePath = val
	}
	if val := getEnv("RCLONE_FLAGS"); val != "" {
		cfg.RcloneFlags = val
	}
	if val := getEnv("RCLONE_SERVE_URL"); val != "" {
		cfg.RcloneServeURL = val
	}
	if val := getEnv("RSS_DELAY"); val != "" {
		cfg.RssDelay = parseInt(val, cfg.RssDelay)
	}

	// 7. Advanced & Feature Toggles
	if val := getEnv("DEBUG"); val != "" {
		cfg.Debug = parseBool(val, false)
	}
	if val := getEnv("DISABLE_LEECH"); val != "" {
		cfg.DisableLeech = parseBool(val, false)
	}
	if val := getEnv("DISABLE_MIRROR"); val != "" {
		cfg.DisableMirror = parseBool(val, false)
	}
	if val := getEnv("DISABLE_TORRENTS"); val != "" {
		cfg.DisableTorrents = parseBool(val, false)
	}
	if val := getEnv("DISABLE_RSS"); val != "" {
		cfg.DisableRSS = parseBool(val, false)
	}
	if val := getEnv("DISABLE_SEARCH"); val != "" {
		cfg.DisableSearch = parseBool(val, false)
	}

	configMutex.Lock()
	GlobalConfig = cfg
	configMutex.Unlock()

	return cfg
}

// Get returns the active global configuration thread-safely
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

func getEnv(keys ...string) string {
	for _, k := range keys {
		if val := strings.TrimSpace(os.Getenv(k)); val != "" {
			return val
		}
	}
	return ""
}

func parseBool(val string, defaultVal bool) bool {
	if val == "" {
		return defaultVal
	}
	val = strings.ToLower(strings.TrimSpace(val))
	return val == "true" || val == "1" || val == "yes" || val == "on" || val == "enable" || val == "enabled"
}

func parseInt(val string, defaultVal int) int {
	if val == "" {
		return defaultVal
	}
	if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
		return i
	}
	return defaultVal
}

func parseInt64(val string, defaultVal int64) int64 {
	if val == "" {
		return defaultVal
	}
	if i, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
		return i
	}
	return defaultVal
}

func parseUint64(val string, defaultVal uint64) uint64 {
	if val == "" {
		return defaultVal
	}
	if i, err := strconv.ParseUint(strings.TrimSpace(val), 10, 64); err == nil {
		return i
	}
	return defaultVal
}

func parseIntList(s string) []int64 {
	var list []int64
	cleaned := strings.ReplaceAll(s, ",", " ")
	for _, part := range strings.Fields(cleaned) {
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
			// Only set if not already defined in the platform environment
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}
