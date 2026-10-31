package handlers

import (
	"fmt"
	"time"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/providers"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// HandleClone initiates a direct cloud-to-cloud clone without local disk downloading
func HandleClone(m *tg.NewMessage, botClient *tg.Client) error {
	senderID := m.SenderID()
	chatID := m.ChatID()

	if !auth.CanExecute(senderID, chatID) {
		_, err := m.Reply("⛔️ You are not authorized to execute tasks.", &tg.SendOptions{})
		return err
	}

	rawArgs := m.Args()
	opts := utils.ParseTaskOptions(rawArgs)

	if opts.Link == "" {
		usage := `⚠️ <b>Clone Usage:</b>
┃ ┠ <code>/clone &lt;source_remote:path&gt; [destination_remote:path]</code>
┃ ┖ <b>Example:</b> <code>/clone gd:Movies/2026 onedrive:Backup/2026</code>`
		_, err := m.Reply(usage, &tg.SendOptions{ParseMode: "html"})
		return err
	}

	src := opts.Link
	dst := opts.UploadDest
	if dst == "" {
		cfg := config.Get()
		dst = cfg.RclonePath
	}

	if dst == "" {
		_, err := m.Reply("❌ No destination cloud path specified. Use <code>-up remote:path</code> or configure <code>RCLONE_PATH</code>.", &tg.SendOptions{ParseMode: "html"})
		return err
	}

	// Create and register Clone task
	t := task.NewTask(senderID, m.Sender.Username, chatID, m.ID, task.TypeClone, opts)
	t.Name = fmt.Sprintf("Clone: %s -> %s", src, dst)
	t.Engine = "Rclone Clone"
	t.SetStatus(task.StatusUploading)

	task.GlobalManager.AddTask(t)
	defer task.GlobalManager.RemoveTask(t.GID)

	statusMsg, _ := m.Reply(fmt.Sprintf("🔄 <b>Clone Task Initiated:</b> <code>%s</code>\n┠ <b>Source:</b> <code>%s</code>\n┖ <b>Target:</b> <code>%s</code>", t.GID, src, dst), &tg.SendOptions{ParseMode: "html"})
	if statusMsg != nil {
		RegisterStatusSession(chatID, statusMsg.ID)
	}

	rclone := providers.NewRcloneUploader()
	res, err := rclone.Clone(t.Ctx, t, src, dst, func(processed, total int64, speed float64) {
		t.UpdateProgress(processed, total, speed)
	})

	if err != nil {
		if t.IsCancelled() {
			return nil
		}
		t.SetError(err)
		_, _ = m.Reply(fmt.Sprintf("❌ <b>Clone Failed:</b> %s", err.Error()), &tg.SendOptions{ParseMode: "html"})
		return err
	}

	t.SetStatus(task.StatusCompleted)
	elapsed := utils.FormatDuration(time.Since(t.StartTime))
	successText := fmt.Sprintf(`⌬ <b><u>Clone Transfer Finished!</u></b>
│
┠ <b>Source:</b> <code>%s</code>
┠ <b>Destination:</b> <code>%s</code>
┠ <b>Elapsed Time:</b> <code>%s</code>
┠ <b>By:</b> @%s (<code>#ID%d</code>)
│
┖ <i>Cloud-to-cloud sync completed without local storage!</i>`, src, res.URL, elapsed, t.UserName, t.UserID)

	_, _ = m.Reply(successText, &tg.SendOptions{ParseMode: "html"})
	return nil
}
