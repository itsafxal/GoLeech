package db

import (
	"os"
	"testing"
	"time"
)

func TestDatabaseOperations(t *testing.T) {
	testDB := "test_goleech.db"
	_ = os.Remove(testDB)
	defer os.Remove(testDB)

	store, err := InitDB("", testDB)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer store.Close()

	// 1. Sudo and Auth tests
	if store.IsSudo(12345) {
		t.Errorf("did not expect 12345 to be sudo initially")
	}
	if err := store.AddSudo(12345, "testuser"); err != nil {
		t.Fatalf("AddSudo failed: %v", err)
	}
	if !store.IsSudo(12345) {
		t.Errorf("expected 12345 to be sudo after adding")
	}
	if err := store.RemoveSudo(12345); err != nil {
		t.Fatalf("RemoveSudo failed: %v", err)
	}
	if store.IsSudo(12345) {
		t.Errorf("did not expect 12345 to be sudo after removal")
	}

	// 2. User settings
	settings, err := store.GetUserSettings(555)
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	if settings.Prefix != "" {
		t.Errorf("expected empty prefix initially")
	}

	settings.Prefix = "[GoLeech]"
	settings.AsDoc = true
	settings.SplitSize = 1000
	if err := store.SaveUserSettings(settings); err != nil {
		t.Fatalf("SaveUserSettings failed: %v", err)
	}

	loaded, err := store.GetUserSettings(555)
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	if loaded.Prefix != "[GoLeech]" || !loaded.AsDoc || loaded.SplitSize != 1000 {
		t.Errorf("loaded settings do not match saved settings: %+v", loaded)
	}

	// 3. Task persistence
	taskRec := &TaskRecord{
		GID:       "abcd1234",
		UserID:    555,
		Name:      "test_video.mp4",
		Type:      "Leech",
		Engine:    "HTTP",
		Status:    "Complete",
		Size:      102400,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.SaveTask(taskRec); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	tasks, err := store.GetRecentTasks(10)
	if err != nil || len(tasks) == 0 {
		t.Fatalf("GetRecentTasks failed or returned empty: %v", err)
	}
	if tasks[0].GID != "abcd1234" {
		t.Errorf("expected GID 'abcd1234', got '%s'", tasks[0].GID)
	}

	// 4. RSS Feed and item seen tests
	feed := &RSSFeedRecord{
		Title:       "Test Feed",
		URL:         "https://example.com/rss.xml",
		Command:     "leech",
		ChatID:      -100123,
		LastChecked: time.Now(),
	}
	if err := store.AddRSSFeed(feed); err != nil {
		t.Fatalf("AddRSSFeed failed: %v", err)
	}
	if feed.ID == 0 {
		t.Errorf("expected non-zero feed ID")
	}

	if store.IsRSSItemSeen(feed.ID, "item-1") {
		t.Errorf("did not expect item-1 to be seen initially")
	}
	if err := store.MarkRSSItemSeen(feed.ID, "item-1"); err != nil {
		t.Fatalf("MarkRSSItemSeen failed: %v", err)
	}
	if !store.IsRSSItemSeen(feed.ID, "item-1") {
		t.Errorf("expected item-1 to be marked as seen")
	}

	// 5. Thumbnail Persistence and Restoration test
	uid := int64(889900)
	thumbSampleData := []byte("fake-jpeg-binary-content-12345")
	tPath := "data/thumbnails/889900.jpg"
	_ = os.Remove(tPath)

	stThumb := &UserSettings{
		UserID:      uid,
		ThumbFileID: "telegram-file-id-xyz",
		ThumbPath:   tPath,
		ThumbData:   thumbSampleData,
	}
	if err := store.SaveUserSettings(stThumb); err != nil {
		t.Fatalf("SaveUserSettings with thumbnail failed: %v", err)
	}

	// Verify restoration from DB
	loadedSt, err := store.GetUserSettings(uid)
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	if string(loadedSt.ThumbData) != string(thumbSampleData) {
		t.Errorf("expected thumb data to match, got: %s", string(loadedSt.ThumbData))
	}
	if loadedSt.ThumbFileID != "telegram-file-id-xyz" {
		t.Errorf("expected thumb file ID 'telegram-file-id-xyz', got: %s", loadedSt.ThumbFileID)
	}
	if _, err := os.Stat(tPath); err != nil {
		t.Errorf("expected thumb file to be restored to disk: %v", err)
	}
	_ = os.Remove(tPath)
}
