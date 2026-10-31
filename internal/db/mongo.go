package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func initMongoDB(uri string) (*Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri).
		SetMaxPoolSize(50).
		SetMinPoolSize(5).
		SetMaxConnIdleTime(30 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create mongodb client: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongodb ping failed: %w", err)
	}

	dbName := "goleech"
	// Check if database name is specified in URI path
	if parts := strings.Split(uri, "/"); len(parts) > 3 {
		pathPart := strings.Split(parts[3], "?")[0]
		if pathPart != "" {
			dbName = pathPart
		}
	}

	mDB := client.Database(dbName)
	store := &Store{
		mongoClient: client,
		mongoDB:     mDB,
		isMongo:     true,
	}

	// Create and ensure collections & indexes
	store.ensureMongoIndexes(ctx)

	log.Printf("📦 MongoDB connected successfully: database '%s' (authoritative storage)", dbName)
	return store, nil
}

func (s *Store) ensureMongoIndexes(ctx context.Context) {
	if s.mongoDB == nil {
		return
	}

	uniqueOpt := options.Index().SetUnique(true)

	// users: unique on id
	_, _ = s.mongoDB.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"id": 1},
		Options: uniqueOpt,
	})

	// user_settings: unique on user_id
	_, _ = s.mongoDB.Collection("user_settings").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"user_id": 1},
		Options: uniqueOpt,
	})

	// tasks: unique on gid
	_, _ = s.mongoDB.Collection("tasks").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"gid": 1},
		Options: uniqueOpt,
	})

	// bot_settings: unique on key
	_, _ = s.mongoDB.Collection("bot_settings").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"key": 1},
		Options: uniqueOpt,
	})

	// rss_feeds: unique on url
	_, _ = s.mongoDB.Collection("rss_feeds").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"url": 1},
		Options: uniqueOpt,
	})

	// rss_seen: compound unique on feed_id and item_id
	_, _ = s.mongoDB.Collection("rss_seen").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "feed_id", Value: 1}, {Key: "item_id", Value: 1}},
		Options: uniqueOpt,
	})
}

func (s *Store) mongoClose() error {
	if s.mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.mongoClient.Disconnect(ctx)
	}
	return nil
}

func (s *Store) mongoIsSudo(userID int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u UserRecord
	err := s.mongoDB.Collection("users").FindOne(ctx, bson.M{"id": userID}).Decode(&u)
	return err == nil && u.IsSudo
}

func (s *Store) mongoIsAuthorized(userID, chatID int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u UserRecord
	err := s.mongoDB.Collection("users").FindOne(ctx, bson.M{
		"$or": []bson.M{
			{"id": userID, "is_auth": true},
			{"id": chatID, "is_auth": true},
		},
	}).Decode(&u)
	return err == nil
}

func (s *Store) mongoAddSudo(userID int64, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Update().SetUpsert(true)
	_, err := s.mongoDB.Collection("users").UpdateOne(ctx,
		bson.M{"id": userID},
		bson.M{
			"$set": bson.M{
				"id":         userID,
				"username":   username,
				"is_sudo":    true,
				"is_auth":    true,
				"updated_at": time.Now(),
			},
			"$setOnInsert": bson.M{
				"created_at": time.Now(),
			},
		},
		opts,
	)
	return err
}

func (s *Store) mongoRemoveSudo(userID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.mongoDB.Collection("users").UpdateOne(ctx,
		bson.M{"id": userID},
		bson.M{"$set": bson.M{"is_sudo": false, "updated_at": time.Now()}},
	)
	return err
}

func (s *Store) mongoAuthorizeChat(chatID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Update().SetUpsert(true)
	_, err := s.mongoDB.Collection("users").UpdateOne(ctx,
		bson.M{"id": chatID},
		bson.M{
			"$set": bson.M{
				"id":         chatID,
				"is_auth":    true,
				"updated_at": time.Now(),
			},
			"$setOnInsert": bson.M{
				"created_at": time.Now(),
			},
		},
		opts,
	)
	return err
}

func (s *Store) mongoUnauthorizeChat(chatID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.mongoDB.Collection("users").UpdateOne(ctx,
		bson.M{"id": chatID},
		bson.M{"$set": bson.M{"is_auth": false, "updated_at": time.Now()}},
	)
	return err
}

func (s *Store) mongoIsBlacklisted(id int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u UserRecord
	err := s.mongoDB.Collection("users").FindOne(ctx, bson.M{"id": id}).Decode(&u)
	return err == nil && u.IsBlacklisted
}

func (s *Store) mongoBlacklist(id int64, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Update().SetUpsert(true)
	_, err := s.mongoDB.Collection("users").UpdateOne(ctx,
		bson.M{"id": id},
		bson.M{
			"$set": bson.M{
				"id":             id,
				"username":       username,
				"is_blacklisted": true,
				"is_auth":        false,
				"is_sudo":        false,
				"updated_at":     time.Now(),
			},
			"$setOnInsert": bson.M{
				"created_at": time.Now(),
			},
		},
		opts,
	)
	return err
}

func (s *Store) mongoRemoveBlacklist(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.mongoDB.Collection("users").UpdateOne(ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{"is_blacklisted": false, "updated_at": time.Now()}},
	)
	return err
}

func (s *Store) mongoGetUserStats() (totalUsers, totalSudos, totalAuth, totalBlacklisted int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	coll := s.mongoDB.Collection("users")
	totalUsers, _ = coll.CountDocuments(ctx, bson.M{})
	totalSudos, _ = coll.CountDocuments(ctx, bson.M{"is_sudo": true})
	totalAuth, _ = coll.CountDocuments(ctx, bson.M{"is_auth": true})
	totalBlacklisted, _ = coll.CountDocuments(ctx, bson.M{"is_blacklisted": true})
	return
}

func (s *Store) mongoGetAllUsers() ([]*UserRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cur, err := s.mongoDB.Collection("users").Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var list []*UserRecord
	for cur.Next(ctx) {
		var u UserRecord
		if err := cur.Decode(&u); err == nil {
			list = append(list, &u)
		}
	}
	return list, nil
}

func (s *Store) mongoSetBotSetting(key, val string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Update().SetUpsert(true)
	_, err := s.mongoDB.Collection("bot_settings").UpdateOne(ctx,
		bson.M{"key": key},
		bson.M{"$set": bson.M{"key": key, "value": val, "updated_at": time.Now()}},
		opts,
	)
	return err
}

func (s *Store) mongoGetBotSetting(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var res struct {
		Value string `bson:"value"`
	}
	err := s.mongoDB.Collection("bot_settings").FindOne(ctx, bson.M{"key": key}).Decode(&res)
	return res.Value, err
}

func (s *Store) mongoGetUserSettings(userID int64) (*UserSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	settings := &UserSettings{
		UserID:    userID,
		AsDoc:     true, // Default send mode is Document
		ExtraData: make(map[string]string),
	}

	err := s.mongoDB.Collection("user_settings").FindOne(ctx, bson.M{"user_id": userID}).Decode(settings)
	if err == mongo.ErrNoDocuments {
		return settings, nil
	}
	if err != nil {
		return nil, err
	}

	if settings.ExtraData == nil {
		settings.ExtraData = make(map[string]string)
	}

	// Honor explicit opt-out if set
	if settings.Get("AS_DOCUMENT") == "false" {
		settings.AsDoc = false
	} else if settings.Get("AS_DOCUMENT") == "true" {
		settings.AsDoc = true
	}

	// Automatic thumbnail restoration from MongoDB if file missing on disk
	restoreThumbnailIfNeeded(settings)

	return settings, nil
}

func (s *Store) mongoSaveUserSettings(st *UserSettings) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if st.ExtraData == nil {
		st.ExtraData = make(map[string]string)
	}
	if !st.AsDoc {
		st.ExtraData["AS_DOCUMENT"] = "false"
	} else {
		st.ExtraData["AS_DOCUMENT"] = "true"
	}

	// If thumbnail file exists on disk but ThumbData is missing, read bytes for MongoDB persistence
	if st.ThumbPath != "" && len(st.ThumbData) == 0 {
		if data, err := os.ReadFile(st.ThumbPath); err == nil && len(data) > 0 {
			st.ThumbData = data
		}
	}

	st.UpdatedAt = time.Now()

	opts := options.Update().SetUpsert(true)
	_, err := s.mongoDB.Collection("user_settings").UpdateOne(ctx,
		bson.M{"user_id": st.UserID},
		bson.M{"$set": st},
		opts,
	)
	return err
}

func (s *Store) mongoGetImages() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cur, err := s.mongoDB.Collection("bot_images").Find(ctx, bson.M{})
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	var list []string
	for cur.Next(ctx) {
		var img struct {
			URL string `bson:"url"`
		}
		if err := cur.Decode(&img); err == nil && img.URL != "" {
			list = append(list, img.URL)
		}
	}
	return list
}

func (s *Store) mongoAddImage(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.mongoDB.Collection("bot_images").InsertOne(ctx, bson.M{
		"url":        url,
		"created_at": time.Now(),
	})
	return err
}

func (s *Store) mongoRemoveImage(index int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cur, err := s.mongoDB.Collection("bot_images").Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	var urls []string
	for cur.Next(ctx) {
		var img struct {
			URL string `bson:"url"`
		}
		if err := cur.Decode(&img); err == nil {
			urls = append(urls, img.URL)
		}
	}

	if index >= 0 && index < len(urls) {
		_, err = s.mongoDB.Collection("bot_images").DeleteOne(ctx, bson.M{"url": urls[index]})
		return err
	}
	return nil
}

func (s *Store) mongoClearImages() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.mongoDB.Collection("bot_images").DeleteMany(ctx, bson.M{})
	return err
}

func (s *Store) mongoSaveTask(t *TaskRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Update().SetUpsert(true)
	_, err := s.mongoDB.Collection("tasks").UpdateOne(ctx,
		bson.M{"gid": t.GID},
		bson.M{"$set": t},
		opts,
	)
	return err
}

func (s *Store) mongoGetRecentTasks(limit int) ([]*TaskRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(int64(limit))
	cur, err := s.mongoDB.Collection("tasks").Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var list []*TaskRecord
	for cur.Next(ctx) {
		var tr TaskRecord
		if err := cur.Decode(&tr); err == nil {
			list = append(list, &tr)
		}
	}
	return list, nil
}

func (s *Store) mongoAddRSSFeed(feed *RSSFeedRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if feed.ID == 0 {
		feed.ID = time.Now().UnixNano()
	}
	_, err := s.mongoDB.Collection("rss_feeds").InsertOne(ctx, feed)
	return err
}

func (s *Store) mongoGetRSSFeeds() ([]*RSSFeedRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cur, err := s.mongoDB.Collection("rss_feeds").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var list []*RSSFeedRecord
	for cur.Next(ctx) {
		var r RSSFeedRecord
		if err := cur.Decode(&r); err == nil {
			list = append(list, &r)
		}
	}
	return list, nil
}

func (s *Store) mongoDeleteRSSFeed(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.mongoDB.Collection("rss_feeds").DeleteOne(ctx, bson.M{"id": id})
	return err
}

func (s *Store) mongoIsRSSItemSeen(feedID int64, itemID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := s.mongoDB.Collection("rss_seen").CountDocuments(ctx, bson.M{
		"feed_id": feedID,
		"item_id": itemID,
	})
	return err == nil && count > 0
}

func (s *Store) mongoMarkRSSItemSeen(feedID int64, itemID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Update().SetUpsert(true)
	_, err := s.mongoDB.Collection("rss_seen").UpdateOne(ctx,
		bson.M{"feed_id": feedID, "item_id": itemID},
		bson.M{"$set": bson.M{
			"feed_id": feedID,
			"item_id": itemID,
			"seen_at": time.Now(),
		}},
		opts,
	)
	return err
}

// restoreThumbnailIfNeeded ensures the thumbnail file exists on local disk
func restoreThumbnailIfNeeded(st *UserSettings) {
	if len(st.ThumbData) == 0 {
		return
	}
	if st.ThumbPath == "" {
		st.ThumbPath = filepath.Join("data", "thumbnails", fmt.Sprintf("%d.jpg", st.UserID))
	}
	if _, err := os.Stat(st.ThumbPath); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(st.ThumbPath), 0755)
		_ = os.WriteFile(st.ThumbPath, st.ThumbData, 0644)
	}
}
