package db

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type UserSettings struct {
	UserID     int64  `json:"user_id"`
	DumpChat   int64  `json:"dump_chat"`
	Prefix     string `json:"prefix"`
	Suffix     string `json:"suffix"`
	Caption    string `json:"caption"`
	SplitSize  int64  `json:"split_size"`
	AsDoc      bool   `json:"as_doc"`
	ThumbPath  string `json:"thumb_path"`
	UploadDest string `json:"upload_dest"`
}

type UserRecord struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	IsSudo        bool      `json:"is_sudo"`
	IsAuth        bool      `json:"is_auth"`
	IsBlacklisted bool      `json:"is_blacklisted"`
	CreatedAt     time.Time `json:"created_at"`
}

type TaskRecord struct {
	GID       string    `json:"gid"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Engine    string    `json:"engine"`
	Status    string    `json:"status"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RSSFeedRecord struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	FilterInclude string    `json:"filter_include"`
	FilterExclude string    `json:"filter_exclude"`
	Command       string    `json:"command"`
	ChatID        int64     `json:"chat_id"`
	LastChecked   time.Time `json:"last_checked"`
}

type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

var GlobalStore *Store

func InitDB(dbPath string) (*Store, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	conn.SetMaxOpenConns(1) // SQLite works best with single writer or WAL mode

	schema := `
	PRAGMA journal_mode = WAL;

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		username TEXT,
		is_sudo INTEGER DEFAULT 0,
		is_auth INTEGER DEFAULT 0,
		is_blacklisted INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS user_settings (
		user_id INTEGER PRIMARY KEY,
		dump_chat INTEGER DEFAULT 0,
		prefix TEXT DEFAULT '',
		suffix TEXT DEFAULT '',
		caption TEXT DEFAULT '',
		split_size INTEGER DEFAULT 0,
		as_doc INTEGER DEFAULT 0,
		thumb_path TEXT DEFAULT '',
		upload_dest TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS tasks (
		gid TEXT PRIMARY KEY,
		user_id INTEGER,
		name TEXT,
		type TEXT,
		engine TEXT,
		status TEXT,
		size INTEGER,
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS rss_feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		url TEXT UNIQUE,
		filter_include TEXT DEFAULT '',
		filter_exclude TEXT DEFAULT '',
		command TEXT DEFAULT 'mirror',
		chat_id INTEGER,
		last_checked TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS rss_seen (
		feed_id INTEGER,
		item_id TEXT,
		seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (feed_id, item_id)
	);

	CREATE TABLE IF NOT EXISTS bot_settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);
	`

	if _, err := conn.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to execute migrations: %w", err)
	}

	// Safe column migration for existing databases
	_, _ = conn.Exec("ALTER TABLE users ADD COLUMN is_blacklisted INTEGER DEFAULT 0;")

	store := &Store{db: conn}
	GlobalStore = store
	return store, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) IsSudo(userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sudo int
	err := s.db.QueryRow("SELECT is_sudo FROM users WHERE id = ?", userID).Scan(&sudo)
	return err == nil && sudo == 1
}

func (s *Store) IsAuthorized(userID, chatID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var auth int
	err := s.db.QueryRow("SELECT is_auth FROM users WHERE id = ? OR id = ?", userID, chatID).Scan(&auth)
	return err == nil && auth == 1
}

func (s *Store) AddSudo(userID int64, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT INTO users (id, username, is_sudo, is_auth) VALUES (?, ?, 1, 1)
		ON CONFLICT(id) DO UPDATE SET is_sudo = 1, is_auth = 1, username = excluded.username
	`, userID, username)
	return err
}

func (s *Store) RemoveSudo(userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE users SET is_sudo = 0 WHERE id = ?", userID)
	return err
}

func (s *Store) AuthorizeChat(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT INTO users (id, is_auth) VALUES (?, 1)
		ON CONFLICT(id) DO UPDATE SET is_auth = 1
	`, chatID)
	return err
}

func (s *Store) UnauthorizeChat(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE users SET is_auth = 0 WHERE id = ?", chatID)
	return err
}

func (s *Store) IsBlacklisted(id int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var bl int
	err := s.db.QueryRow("SELECT is_blacklisted FROM users WHERE id = ?", id).Scan(&bl)
	return err == nil && bl == 1
}

func (s *Store) Blacklist(id int64, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT INTO users (id, username, is_blacklisted, is_auth, is_sudo) VALUES (?, ?, 1, 0, 0)
		ON CONFLICT(id) DO UPDATE SET is_blacklisted = 1, is_auth = 0, is_sudo = 0, username = excluded.username
	`, id, username)
	return err
}

func (s *Store) RemoveBlacklist(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE users SET is_blacklisted = 0 WHERE id = ?", id)
	return err
}

func (s *Store) GetUserStats() (totalUsers, totalSudos, totalAuth, totalBlacklisted int64, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_ = s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_sudo = 1").Scan(&totalSudos)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_auth = 1").Scan(&totalAuth)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_blacklisted = 1").Scan(&totalBlacklisted)
	return
}

func (s *Store) GetAllUsers() ([]*UserRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT id, username, is_sudo, is_auth, is_blacklisted, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*UserRecord
	for rows.Next() {
		var u UserRecord
		var sudo, auth, bl int
		if err := rows.Scan(&u.ID, &u.Username, &sudo, &auth, &bl, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.IsSudo = (sudo == 1)
		u.IsAuth = (auth == 1)
		u.IsBlacklisted = (bl == 1)
		list = append(list, &u)
	}
	return list, nil
}

func (s *Store) SetBotSetting(key, val string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT INTO bot_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, val)
	return err
}

func (s *Store) GetBotSetting(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var val string
	err := s.db.QueryRow("SELECT value FROM bot_settings WHERE key = ?", key).Scan(&val)
	return val, err
}

func (s *Store) GetUserSettings(userID int64) (*UserSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	settings := &UserSettings{UserID: userID}
	var asDocInt int
	err := s.db.QueryRow(`
		SELECT dump_chat, prefix, suffix, caption, split_size, as_doc, thumb_path, upload_dest
		FROM user_settings WHERE user_id = ?
	`, userID).Scan(
		&settings.DumpChat,
		&settings.Prefix,
		&settings.Suffix,
		&settings.Caption,
		&settings.SplitSize,
		&asDocInt,
		&settings.ThumbPath,
		&settings.UploadDest,
	)
	if err == sql.ErrNoRows {
		return settings, nil
	}
	if err != nil {
		return nil, err
	}
	settings.AsDoc = (asDocInt == 1)
	return settings, nil
}

func (s *Store) SaveUserSettings(st *UserSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	asDocInt := 0
	if st.AsDoc {
		asDocInt = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO user_settings (user_id, dump_chat, prefix, suffix, caption, split_size, as_doc, thumb_path, upload_dest)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			dump_chat = excluded.dump_chat,
			prefix = excluded.prefix,
			suffix = excluded.suffix,
			caption = excluded.caption,
			split_size = excluded.split_size,
			as_doc = excluded.as_doc,
			thumb_path = excluded.thumb_path,
			upload_dest = excluded.upload_dest
	`, st.UserID, st.DumpChat, st.Prefix, st.Suffix, st.Caption, st.SplitSize, asDocInt, st.ThumbPath, st.UploadDest)
	return err
}

func (s *Store) SaveTask(t *TaskRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT INTO tasks (gid, user_id, name, type, engine, status, size, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(gid) DO UPDATE SET
			status = excluded.status,
			size = excluded.size,
			updated_at = excluded.updated_at
	`, t.GID, t.UserID, t.Name, t.Type, t.Engine, t.Status, t.Size, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *Store) GetRecentTasks(limit int) ([]*TaskRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`
		SELECT gid, user_id, name, type, engine, status, size, created_at, updated_at
		FROM tasks ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*TaskRecord
	for rows.Next() {
		var tr TaskRecord
		if err := rows.Scan(&tr.GID, &tr.UserID, &tr.Name, &tr.Type, &tr.Engine, &tr.Status, &tr.Size, &tr.CreatedAt, &tr.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &tr)
	}
	return list, nil
}

func (s *Store) AddRSSFeed(feed *RSSFeedRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`
		INSERT INTO rss_feeds (title, url, filter_include, filter_exclude, command, chat_id, last_checked)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, feed.Title, feed.URL, feed.FilterInclude, feed.FilterExclude, feed.Command, feed.ChatID, feed.LastChecked)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		feed.ID = id
	}
	return err
}

func (s *Store) GetRSSFeeds() ([]*RSSFeedRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`
		SELECT id, title, url, filter_include, filter_exclude, command, chat_id, last_checked
		FROM rss_feeds
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*RSSFeedRecord
	for rows.Next() {
		var r RSSFeedRecord
		if err := rows.Scan(&r.ID, &r.Title, &r.URL, &r.FilterInclude, &r.FilterExclude, &r.Command, &r.ChatID, &r.LastChecked); err != nil {
			return nil, err
		}
		list = append(list, &r)
	}
	return list, nil
}

func (s *Store) DeleteRSSFeed(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM rss_feeds WHERE id = ?", id)
	return err
}

func (s *Store) IsRSSItemSeen(feedID int64, itemID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM rss_seen WHERE feed_id = ? AND item_id = ?", feedID, itemID).Scan(&count)
	return count > 0
}

func (s *Store) MarkRSSItemSeen(feedID int64, itemID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("INSERT OR IGNORE INTO rss_seen (feed_id, item_id) VALUES (?, ?)", feedID, itemID)
	return err
}
