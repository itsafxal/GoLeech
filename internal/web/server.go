package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
)

type Server struct {
	httpServer *http.Server
	startTime  time.Time
}

func StartServer(port int) *Server {
	s := &Server{
		startTime: time.Now(),
	}

	mux := http.NewServeMux()

	// Static & Dashboard
	mux.HandleFunc("/", s.handleDashboard)

	// REST APIs
	mux.HandleFunc("/api/tasks", s.handleAPITasks)
	mux.HandleFunc("/api/tasks/cancel", s.handleAPICancelTask)
	mux.HandleFunc("/api/stats", s.handleAPIStats)
	mux.HandleFunc("/healthz", s.handleHealthz)

	// Media Streaming & Range Download
	mux.HandleFunc("/stream/", s.handleStream)
	mux.HandleFunc("/dl/", s.handleStream)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // allow infinite for streaming
	}

	go func() {
		_ = s.httpServer.ListenAndServe()
	}()

	return s
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) checkAuth(r *http.Request) bool {
	cfg := config.Get()
	if cfg.WebAccessPassword == "" {
		return true // open if no password configured
	}

	// Check Authorization header or ?pass= query param
	pass := r.URL.Query().Get("pass")
	if pass == cfg.WebAccessPassword {
		return true
	}

	authHdr := r.Header.Get("Authorization")
	if authHdr == "Bearer "+cfg.WebAccessPassword {
		return true
	}

	return false
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","time":"` + time.Now().Format(time.RFC3339) + `"}`))
}

func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := map[string]interface{}{
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"goroutines":     runtime.NumGoroutine(),
		"cpu_cores":      runtime.NumCPU(),
		"uptime_seconds": int64(time.Since(s.startTime).Seconds()),
		"active_tasks":   len(task.GlobalManager.GetAllTasks()),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleAPITasks(w http.ResponseWriter, r *http.Request) {
	tasks := task.GlobalManager.GetAllTasks()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handleAPICancelTask(w http.ResponseWriter, r *http.Request) {
	if !s.checkAuth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	gid := r.URL.Query().Get("gid")
	if gid == "" {
		http.Error(w, "Missing gid parameter", http.StatusBadRequest)
		return
	}

	success, msg := task.GlobalManager.CancelTask(gid, 0, true)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"message": msg,
	})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	// e.g. /stream/{gid}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		http.NotFound(w, r)
		return
	}

	gid := pathParts[1]
	t := task.GlobalManager.GetTask(gid)
	if t == nil || t.LocalPath == "" {
		http.NotFound(w, r)
		return
	}

	// Find media file in task directory
	filePath := t.LocalPath
	info, err := os.Stat(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if info.IsDir() {
		entries, _ := os.ReadDir(filePath)
		for _, e := range entries {
			if !e.IsDir() {
				filePath = filepath.Join(filePath, e.Name())
				break
			}
		}
	}

	// Serve content with HTTP Range request support (essential for video seek in browser/VLC)
	http.ServeFile(w, r, filePath)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tasks := task.GlobalManager.GetAllTasks()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>GoLeech - Mirror Anything. Leech Anywhere.</title>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: 'Inter', sans-serif; background: #0f172a; color: #f8fafc; padding: 2rem; }
    .container { max-width: 1100px; margin: 0 auto; }
    header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; padding-bottom: 1rem; border-bottom: 1px solid #1e293b; }
    .logo { font-size: 1.5rem; font-weight: 700; color: #38bdf8; display: flex; align-items: center; gap: 0.5rem; }
    .tagline { color: #94a3b8; font-size: 0.9rem; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
    .card { background: #1e293b; border-radius: 0.75rem; padding: 1.25rem; border: 1px solid #334155; }
    .card h3 { color: #94a3b8; font-size: 0.85rem; text-transform: uppercase; margin-bottom: 0.5rem; }
    .card .val { font-size: 1.5rem; font-weight: 700; color: #f8fafc; }
    .section-title { font-size: 1.25rem; font-weight: 600; margin-bottom: 1rem; color: #e2e8f0; }
    .task-card { background: #1e293b; border-radius: 0.75rem; padding: 1.25rem; margin-bottom: 1rem; border: 1px solid #334155; }
    .task-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; }
    .task-name { font-weight: 600; font-size: 1.05rem; }
    .badge { padding: 0.25rem 0.6rem; border-radius: 0.375rem; font-size: 0.75rem; font-weight: 600; text-transform: uppercase; }
    .badge-dl { background: #0284c7; color: #fff; }
    .badge-up { background: #059669; color: #fff; }
    .badge-done { background: #16a34a; color: #fff; }
    .progress-bar-bg { background: #334155; height: 8px; border-radius: 4px; overflow: hidden; margin-bottom: 0.75rem; }
    .progress-bar-fill { background: #38bdf8; height: 100%%; transition: width 0.3s; }
    .task-meta { display: flex; justify-content: space-between; font-size: 0.85rem; color: #94a3b8; }
    .no-tasks { text-align: center; padding: 3rem; color: #64748b; font-size: 1.1rem; }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div>
        <div class="logo">⚡️ GoLeech</div>
        <div class="tagline">Mirror Anything. Leech Anywhere. Powered by Go.</div>
      </div>
      <div>
        <span class="badge" style="background:#0284c7;">MTProto Native</span>
      </div>
    </header>

    <div class="grid">
      <div class="card">
        <h3>Active Tasks</h3>
        <div class="val">%d</div>
      </div>
      <div class="card">
        <h3>RAM Allocation</h3>
        <div class="val">%d MB</div>
      </div>
      <div class="card">
        <h3>Goroutines</h3>
        <div class="val">%d</div>
      </div>
      <div class="card">
        <h3>Uptime</h3>
        <div class="val">%s</div>
      </div>
    </div>

    <div class="section-title">Active Transmissions</div>
`, len(tasks), m.Alloc/1024/1024, runtime.NumGoroutine(), utils.FormatDuration(time.Since(s.startTime)))

	if len(tasks) == 0 {
		fmt.Fprintf(w, `<div class="task-card no-tasks">No active tasks in progress. Send /mirror or /leech in Telegram.</div>`)
	} else {
		for _, t := range tasks {
			pct := t.Progress()
			badgeClass := "badge-dl"
			if t.Status == task.StatusUploading {
				badgeClass = "badge-up"
			} else if t.Status == task.StatusCompleted {
				badgeClass = "badge-done"
			}

			fmt.Fprintf(w, `
      <div class="task-card">
        <div class="task-header">
          <div class="task-name">%s</div>
          <span class="badge %s">%s</span>
        </div>
        <div class="progress-bar-bg">
          <div class="progress-bar-fill" style="width: %.1f%%%%;"></div>
        </div>
        <div class="task-meta">
          <span>Processed: %s / %s (%.1f%%%%)</span>
          <span>Speed: %s</span>
          <span>ETA: %s</span>
          <span>Engine: %s</span>
        </div>
      </div>`,
				t.Name, badgeClass, t.Status,
				pct,
				utils.FormatBytes(t.ProcessedBytes), utils.FormatBytes(t.TotalSize), pct,
				utils.FormatSpeed(t.Speed),
				utils.FormatSeconds(t.ETA),
				t.Engine,
			)
		}
	}

	fmt.Fprintf(w, `
  </div>
</body>
</html>`)
}
