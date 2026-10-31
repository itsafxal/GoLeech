package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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

func main() {
	fmt.Print(`
   ______      __                   __  
  / ____/___  / /   ___  ___  _____/ /_ 
 / / __/ __ \/ /   / _ \/ _ \/ ___/ __ \
/ /_/ / /_/ / /___/  __/  __/ /__/ / / /
\____/\____/_____/\___/\___/\___/_/ /_/ 
`)
	fmt.Println("🚀 Initializing GoLeech - Powered by Go & Gogram MTProto...")

	// 1. Load Configuration
	cfg := config.LoadConfig()
	if cfg.BotToken == "" {
		log.Println("⚠️ WARNING: BOT_TOKEN is not set in environment or config. Please configure .env")
	}

	// 2. Initialize Database Layer
	store, err := db.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer store.Close()
	log.Printf("📦 Database connected: %s (SQLite WAL Mode)", cfg.DatabasePath)

	// 3. Initialize Task Manager
	taskMgr := task.InitManager()
	_ = taskMgr
	log.Println("⚡️ Task & Queue Manager ready")

	// 4. Initialize MTProto Bot Client
	var botClient *tg.Client
	var userClient *tg.Client

	if cfg.BotToken != "" && cfg.TelegramAPI != 0 && cfg.TelegramHash != "" {
		var bErr error
		botClient, bErr = tg.QuickBot(cfg.TelegramAPI, cfg.TelegramHash, cfg.BotToken)
		if bErr != nil {
			log.Fatalf("❌ Failed to initialize Gogram Bot client: %v", bErr)
		}
		log.Println("🤖 Gogram MTProto Bot client connected successfully")

		// Optional: Initialize Telegram User session if configured
		if cfg.UserSessionString != "" {
			var uErr error
			userClient, uErr = tg.NewClient(tg.ClientConfig{
				AppID:         cfg.TelegramAPI,
				AppHash:       cfg.TelegramHash,
				StringSession: cfg.UserSessionString,
			})
			if uErr != nil {
				log.Printf("⚠️ Failed to initialize Gogram User Client: %v", uErr)
			} else {
				if _, err := userClient.Conn(); err == nil {
					log.Println("👤 Telegram 4GB Premium User Client connected")
				} else {
					log.Printf("⚠️ User Client connection error: %v", err)
				}
			}
		}

		// 5. Initialize Plugin Registry
		pluginReg := plugin.InitRegistry(botClient)
		_ = pluginReg.RegisterPlugin(context.Background(), &plugin.MediaInfoPlugin{})
		log.Println("🔌 Plugin Registry initialized with built-in extensions")

		// 6. Register Routes & Handlers
		bot.RegisterRoutes(botClient, userClient)
		log.Println("📡 Telegram command handlers & callback router registered")
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

	// 9. Graceful Shutdown Signal Handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("🛑 Received termination signal (%v). Initiating graceful shutdown...", sig)

		rootCancel()
		if webServer != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = webServer.Shutdown(shutdownCtx)
		}

		// Cancel all remaining active tasks
		task.GlobalManager.CancelAll("all", 0, true)
		log.Println("✅ All active tasks stopped and resources cleared. Goodbye!")
		os.Exit(0)
	}()

	log.Println("🎉 GoLeech is running and ready for transmissions!")

	if botClient != nil {
		botClient.Idle()
	} else {
		select {}
	}
}
