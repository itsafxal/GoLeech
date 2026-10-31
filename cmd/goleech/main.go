package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/GoLeech/GoLeech/internal/bot"
	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/internal/plugin"
	"github.com/GoLeech/GoLeech/internal/rss"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/internal/web"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func initBotClient(cfg *config.Config) (*tg.Client, error) {
	// Use persistent session file to avoid repeated DH handshakes and Wrong new_nonce_hash1
	sessionDir := "data"
	_ = os.MkdirAll(sessionDir, 0755)
	sessionPath := filepath.Join(sessionDir, "bot.session")

	client, err := tg.NewClient(tg.ClientConfig{
		AppID:   cfg.TelegramAPI,
		AppHash: cfg.TelegramHash,
		Session: sessionPath,
	})
	if err != nil {
		client, err = tg.NewClient(tg.ClientConfig{
			AppID:   cfg.TelegramAPI,
			AppHash: cfg.TelegramHash,
		})
		if err != nil {
			return nil, fmt.Errorf("creating client: %w", err)
		}
	}

	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		if err := client.Connect(); err != nil {
			lastErr = err
			log.Printf("⚠️ Telegram connection attempt %d/5 failed: %v. Retrying in 2s...", attempt, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if err := client.LoginBot(cfg.BotToken); err != nil {
			lastErr = err
			log.Printf("⚠️ Telegram bot login attempt %d/5 failed: %v. Retrying in 2s...", attempt, err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Pre-cache bot info to avoid RPC deadlocks during update handling
		if me, err := client.GetMe(); err == nil && me != nil {
			log.Printf("🤖 Bot authenticated as @%s (ID: %d - %s)", me.Username, me.ID, me.FirstName)
		}

		// Synchronize initial update stream state
		if state, err := client.UpdatesGetState(); err == nil && state != nil {
			log.Printf("🔄 Update stream active (PTS: %d, Date: %d)", state.Pts, state.Date)
			go client.FetchDifferenceOnStartup(state.Pts)
		}

		return client, nil
	}

	return nil, fmt.Errorf("failed after 5 attempts: %w", lastErr)
}

func main() {
	fmt.Print(`
   ______      __                   __  
  / ____/___  / /   ___  ___  _____/ /_ 
 / / __/ __ \/ /   / _ \/ _ \/ ___/ __ \
/ /_/ / /_/ / /___/  __/  __/ /__/ / / /
\____/\____/_____/\___/\___/\___/_/ /_/ 
`)
	fmt.Println("🚀 Initializing GoLeech - Powered by Go & Gogram MTProto...")

	// 1. Load Configuration & Startup Validation
	cfg := config.LoadConfig()
	if missing := cfg.Validate(); len(missing) > 0 {
		log.Printf("⚠️ WARNING: Missing required startup configuration: %s", strings.Join(missing, ", "))
		log.Println("👉 Please configure your credentials in .env (refer to .env.example)")
	}

	// 2. Initialize Database Layer (MongoDB authoritative permanent storage with SQLite fallback)
	store, err := db.InitDB(cfg.MongoURI, cfg.DatabasePath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer store.Close()

	// 3. Initialize Task Manager
	taskMgr := task.InitManager()
	_ = taskMgr
	log.Println("⚡️ Task & Queue Manager ready")

	// 4. Initialize MTProto Bot Client
	var botClient *tg.Client

	if cfg.BotToken != "" && cfg.TelegramAPI != 0 && cfg.TelegramHash != "" {
		var bErr error
		botClient, bErr = initBotClient(cfg)
		if bErr != nil {
			log.Fatalf("❌ Failed to initialize Gogram Bot client: %v", bErr)
		}
		log.Println("🤖 Gogram MTProto Bot client connected successfully")

		// 5. Register Routes & Handlers immediately
		bot.RegisterRoutes(botClient)
		log.Println("📡 Telegram command handlers & callback router registered")

		// Automatically register all supported commands with Telegram native command menu
		if err := bot.RegisterBotCommands(botClient); err != nil {
			log.Printf("⚠️ Automatic bot command registration failed: %v", err)
		}

		// 6. Initialize Plugin Registry
		pluginReg := plugin.InitRegistry(botClient)
		_ = pluginReg.RegisterPlugin(context.Background(), &plugin.MediaInfoPlugin{})
		log.Println("🔌 Plugin Registry initialized with built-in extensions")

		// Optional: Asynchronously initialize Telegram User session if configured
		if cfg.UserSessionString != "" {
			go func() {
				log.Println("👤 Connecting Telegram User Client in background...")
				uClient, uErr := tg.NewClient(tg.ClientConfig{
					AppID:         cfg.TelegramAPI,
					AppHash:       cfg.TelegramHash,
					StringSession: cfg.UserSessionString,
				})
				if uErr != nil {
					log.Printf("⚠️ User Client initialization skipped: %v", uErr)
					return
				}

				connCh := make(chan error, 1)
				go func() {
					connCh <- uClient.Connect()
				}()

				select {
				case err := <-connCh:
					if err == nil {
						bot.SetUserClient(uClient)
						log.Println("👤 Telegram 4GB Premium User Client connected successfully")
					} else {
						log.Printf("⚠️ User Client connection error: %v (generate session with 'go run ./cmd/login')", err)
					}
				case <-time.After(6 * time.Second):
					log.Println("⚠️ User Client connection timed out (6s). Running in standard 2GB Bot mode.")
				}
			}()
		}
	} else {
		log.Println("⚠️ Telegram credentials not fully provided. Bot will run in Web/API mode.")
	}

	// 7. Start Web Dashboard and HTTP API
	webServer := web.StartServer(cfg.WebPort)
	log.Printf("🌐 Web Dashboard & Streaming Server running at http://localhost:%d", cfg.WebPort)

	// 8. Start Background RSS Monitor
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()
	rssMonitor := rss.StartRSSMonitor(rootCtx)
	_ = rssMonitor
	log.Println("📰 Background RSS Feed Monitor active")

	log.Println("🎉 GoLeech is running and ready for transmissions!")

	// 9. Process Life-Cycle & Graceful Shutdown
	// We wait directly on OS termination signals so internal MTProto DC migrations or TCP reconnects
	// do NOT cause the application process to prematurely exit!
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	sig := <-sigCh
	log.Printf("🛑 Received termination signal (%v). Initiating graceful shutdown...", sig)

	rootCancel()
	if webServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = webServer.Shutdown(shutdownCtx)
	}

	if botClient != nil {
		_ = botClient.Stop()
	}
	if uClient := bot.GetUserClient(); uClient != nil {
		_ = uClient.Stop()
	}

	// Cancel all remaining active tasks and cleanup
	task.GlobalManager.CancelAll("all", 0, true)
	log.Println("✅ All active tasks stopped and resources cleared. Goodbye!")
}
