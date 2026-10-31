package handlers

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/pkg/utils"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// PendingUserState tracks an active interactive input wait
type PendingUserState struct {
	Option    string
	Category  string
	ExpiresAt time.Time
}

var (
	pendingInputs   = make(map[int64]*PendingUserState)
	pendingInputsMu sync.RWMutex
)

// SetPendingInput records a prompt waiting for user response
func SetPendingInput(userID int64, option, category string) {
	pendingInputsMu.Lock()
	defer pendingInputsMu.Unlock()
	pendingInputs[userID] = &PendingUserState{
		Option:    option,
		Category:  category,
		ExpiresAt: time.Now().Add(60 * time.Second),
	}
}

// GetPendingInput returns the pending state if valid
func GetPendingInput(userID int64) *PendingUserState {
	pendingInputsMu.RLock()
	defer pendingInputsMu.RUnlock()
	st, exists := pendingInputs[userID]
	if !exists {
		return nil
	}
	if time.Now().After(st.ExpiresAt) {
		delete(pendingInputs, userID)
		return nil
	}
	return st
}

// ClearPendingInput removes the pending state
func ClearPendingInput(userID int64) {
	pendingInputsMu.Lock()
	defer pendingInputsMu.Unlock()
	delete(pendingInputs, userID)
}

// OptionInfo holds human-readable documentation for each setting
type OptionInfo struct {
	Title       string
	Description string
	Prompt      string
	ParentPage  string
}

var OptionCatalog = map[string]OptionInfo{
	"THUMBNAIL": {
		Title:       "Custom Thumbnail",
		Description: "Custom Thumbnail is used as the cover thumbnail for files uploaded to Telegram in media or document mode.",
		Prompt:      "Send a photo to save it as your custom thumbnail.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "thumb",
	},
	"THUMBNAIL_LAYOUT": {
		Title:       "Thumbnail Layout Grid",
		Description: "Grabs frames spread across the video and tiles them into a photo contact sheet. Example: 2x2 (4 frames), 3x3 (9 frames).",
		Prompt:      "Send thumbnail layout (width x height, e.g. 2x2, 3x3, 2x4). Example: <code>3x3</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "thumb",
	},
	"LEECH_SPLIT_SIZE": {
		Title:       "Leech Split Size",
		Description: "Maximum file size before chunking. 2GB for regular accounts, 4GB for Telegram Premium.",
		Prompt:      "Send Leech split size in bytes or formatted (e.g. <code>2GB</code>, <code>4000000000</code>, <code>1000MB</code>).\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "leech",
	},
	"LEECH_PREFIX": {
		Title:       "Leech Prefix",
		Description: "Text placed at the start of every leeched file name. HTML is rendered in caption and stripped from filename. Use \\s for a space.",
		Prompt:      "Send Leech Filename Prefix. Example: <code>[GoLeech]</code> or <code>@mychannel\\s</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "leech",
	},
	"LEECH_SUFFIX": {
		Title:       "Leech Suffix",
		Description: "Text placed at the end of the filename before extension.",
		Prompt:      "Send Leech Filename Suffix. Example: <code>_GoLeech</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "leech",
	},
	"LEECH_CAPTION": {
		Title:       "Leech Caption Template",
		Description: "Custom caption template. Variables: {filename}, {size}, {duration}, {quality}, {languages}, {subtitles}.",
		Prompt:      "Send custom caption template. Example:\n<code>🎬 <b>{filename}</b>\n📦 Size: {size}</code>\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "leech",
	},
	"LEECH_DUMP_CHATS": {
		Title:       "Leech Dump Chats",
		Description: "Target channels or groups where uploaded media is archived. Chat IDs can be separated by spaces or newlines.",
		Prompt:      "Send Dump Chat ID(s). Example: <code>-1001234567890</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "leech",
	},
	"METADATA": {
		Title:       "Global Metadata Title",
		Description: "Injected stream title into MKV/MP4 files without re-encoding.",
		Prompt:      "Send Metadata Title. Example: <code>Encoded by GoLeech</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ffset",
	},
	"AUDIO_METADATA": {
		Title:       "Audio Stream Metadata",
		Description: "Title applied to audio tracks.",
		Prompt:      "Send Audio Metadata. Example: <code>Stereo English [GoLeech]</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ffset",
	},
	"VIDEO_METADATA": {
		Title:       "Video Stream Metadata",
		Description: "Title applied to visual video streams.",
		Prompt:      "Send Video Metadata. Example: <code>1080p x264 GoLeech</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ffset",
	},
	"SUBTITLE_METADATA": {
		Title:       "Subtitle Stream Metadata",
		Description: "Title applied to subtitle tracks.",
		Prompt:      "Send Subtitle Metadata. Example: <code>English Full SDH</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ffset",
	},
	"FFMPEG_CMDS": {
		Title:       "FFmpeg Commands",
		Description: "Custom FFmpeg argument presets executed before uploading media.",
		Prompt:      "Send FFmpeg arguments. Example: <code>-c:v libx265 -crf 28 -c:a copy</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ffset",
	},
	"NAME_SWAP": {
		Title:       "Auto Name Swap",
		Description: "Regex or substring replacement run on filenames before upload. Format: find:replace",
		Prompt:      "Send swap rule in <code>find:replace</code> format. Example: <code>720p:1080p</code> or <code>@OldGroup:@MyChannel</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "advanced",
	},
	"EXCLUDED_EXTENSIONS": {
		Title:       "Excluded Extensions",
		Description: "Space-separated file extensions that will never be uploaded (e.g. sample.nfo, .txt).",
		Prompt:      "Send extensions without dots, separated by space. Example: <code>txt nfo html exe</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "advanced",
	},
	"UPLOAD_PATHS": {
		Title:       "Custom Upload Paths",
		Description: "Alias map for remote destinations.",
		Prompt:      "Send alias mapping. Example: <code>Movies:gdrive:/Archive/Movies</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "advanced",
	},
	"RCLONE_CONFIG": {
		Title:       "Rclone Config File",
		Description: "Upload your personal rclone.conf file for private remote transfers.",
		Prompt:      "Send your <code>rclone.conf</code> file as a document.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "rclone",
	},
	"RCLONE_PATH": {
		Title:       "Default Rclone Path",
		Description: "Default cloud destination path (e.g. remote:folder).",
		Prompt:      "Send default Rclone path. Example: <code>drive:Archive/Uploads</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "rclone",
	},
	"RCLONE_FLAGS": {
		Title:       "Rclone Flags",
		Description: "Extra flags passed to rclone commands, separated by |.",
		Prompt:      "Send flags separated by |. Example: <code>--buffer-size:16M|--transfers:4</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "rclone",
	},
	"GDRIVE_ID": {
		Title:       "Default GDrive ID",
		Description: "Google Drive Folder or Shared Drive ID.",
		Prompt:      "Send GDrive Folder ID or link.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "gdrive",
	},
	"INDEX_URL": {
		Title:       "Default Index URL",
		Description: "Direct HTTP Index URL matching your GDrive ID.",
		Prompt:      "Send your HTTP Index URL (starting with https://).\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "gdrive",
	},
	"TOKEN_PICKLE": {
		Title:       "GDrive token.pickle",
		Description: "Google Drive user authentication pickle token.",
		Prompt:      "Send your <code>token.pickle</code> file as a document.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "gdrive",
	},
	"DRIVE_CAT": {
		Title:       "User Drive Categories",
		Description: "Categorized upload destinations. Format: Name:FolderID|IndexURL",
		Prompt:      "Send category mapping. Example: <code>Anime:19jXo...|https://index.anime</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "gdrive",
	},
	"GOFILE_TOKEN": {
		Title:       "Gofile API Token",
		Description: "Gofile account API token for direct cloud uploads.",
		Prompt:      "Send your Gofile API Token.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "gofile",
	},
	"GOFILE_FOLDER_ID": {
		Title:       "Gofile Folder ID",
		Description: "Gofile destination folder ID. Leave empty for root.",
		Prompt:      "Send your Gofile Folder ID.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "gofile",
	},
	"BUZZHEAVIER_TOKEN": {
		Title:       "BuzzHeavier API Token",
		Description: "BuzzHeavier Account token.",
		Prompt:      "Send your BuzzHeavier API Token.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "buzzheavier",
	},
	"BUZZHEAVIER_FOLDER_ID": {
		Title:       "BuzzHeavier Folder ID",
		Description: "BuzzHeavier destination folder ID.",
		Prompt:      "Send your BuzzHeavier Folder ID.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "buzzheavier",
	},
	"PIXELDRAIN_KEY": {
		Title:       "PixelDrain API Key",
		Description: "PixelDrain user API Key.",
		Prompt:      "Send your PixelDrain API Key.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "pixeldrain",
	},
	"DEVUPLOADS_KEY": {
		Title:       "DevUploads API Key",
		Description: "DevUploads user account API key.",
		Prompt:      "Send your DevUploads API Key.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "devuploads",
	},
	"DEVUPLOADS_FOLDER": {
		Title:       "DevUploads Folder ID",
		Description: "Folder ID for DevUploads uploads.",
		Prompt:      "Send your DevUploads Folder ID.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "devuploads",
	},
	"VIKINGFILE_HASH": {
		Title:       "VikingFile User Hash",
		Description: "VikingFile authorization user hash.",
		Prompt:      "Send your VikingFile User Hash.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "vikingfile",
	},
	"VIKINGFILE_FOLDER": {
		Title:       "VikingFile Folder",
		Description: "Folder name/path on VikingFile.",
		Prompt:      "Send your VikingFile Folder path.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "vikingfile",
	},
	"MEGA_EMAIL": {
		Title:       "Mega Email",
		Description: "Mega.nz account email.",
		Prompt:      "Send your Mega.nz account email address.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "mega",
	},
	"MEGA_PASSWORD": {
		Title:       "Mega Password",
		Description: "Mega.nz account password.",
		Prompt:      "Send your Mega.nz account password.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "mega",
	},
	"SEEDR_EMAIL": {
		Title:       "Seedr Email",
		Description: "Seedr.cc account email.",
		Prompt:      "Send your Seedr.cc account email address.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "seedr",
	},
	"SEEDR_PASSWORD": {
		Title:       "Seedr Password",
		Description: "Seedr.cc account password.",
		Prompt:      "Send your Seedr.cc account password.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "seedr",
	},
	"USER_SESSION": {
		Title:       "User String Session",
		Description: "Telegram string session used for user account downloads & clones up to 4GB.",
		Prompt:      "Send your Telegram String Session. It will be encrypted securely.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "clone",
	},
	"CLONE_DUMP_CHATS": {
		Title:       "Clone Dump Chats",
		Description: "Default destination channels for /clone tasks.",
		Prompt:      "Send clone dump chat IDs. Example: <code>-1001234567890</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "clone",
	},
	"CLONE_CONTENT_TYPE": {
		Title:       "Clone Content Type",
		Description: "Filter files in clone tasks: doc, med, or all.",
		Prompt:      "Send one of: <code>doc</code>, <code>med</code>, <code>all</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "clone",
	},
	"CLONE_EXCLUDED_EXTENSIONS": {
		Title:       "Clone Excluded Extensions",
		Description: "Extensions ignored exclusively by /clone tasks.",
		Prompt:      "Send space-separated extensions. Example: <code>mkv srt txt</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "clone",
	},
	"CLONE_FILTERS": {
		Title:       "Clone Regex Filters",
		Description: "Regex applied to file names and captions in /clone.",
		Prompt:      "Send filter rule. Example: <code>1080p</code> or <code>sample:skip</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "clone",
	},
	"YT_DLP_OPTIONS": {
		Title:       "YT-DLP Options",
		Description: "Custom parameters passed into YT-DLP.",
		Prompt:      "Send YT-DLP options. Example: <code>format=bestvideo+bestaudio</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ytdlp",
	},
	"USER_COOKIE_FILE": {
		Title:       "User Cookie File",
		Description: "cookies.txt file for authenticating YT-DLP downloads.",
		Prompt:      "Send your <code>cookies.txt</code> file as a document.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ytdlp",
	},
	"YT_DESP": {
		Title:       "YouTube Description",
		Description: "Custom description for YouTube videos.",
		Prompt:      "Send custom YouTube description.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ytdlp",
	},
	"YT_TAGS": {
		Title:       "YouTube Tags",
		Description: "Comma-separated tags for YouTube metadata.",
		Prompt:      "Send tags separated by commas. Example: <code>anime,action,trailer</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ytdlp",
	},
	"YT_CATEGORY_ID": {
		Title:       "YouTube Category ID",
		Description: "Numeric category ID for YouTube uploads (e.g. 24 for Entertainment).",
		Prompt:      "Send numeric category ID (e.g. <code>24</code>).\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ytdlp",
	},
	"YT_PRIVACY_STATUS": {
		Title:       "YouTube Privacy Status",
		Description: "Upload visibility: public, private, or unlisted.",
		Prompt:      "Send one of: <code>public</code>, <code>private</code>, <code>unlisted</code>.\n\n⏳ <b>Time Left:</b> <code>60 sec</code>",
		ParentPage:  "ytdlp",
	},
}

// GetUserSettingsFull builds the WZML-X hierarchical settings view and buttons
func GetUserSettingsFull(userID int64, page string) (string, tg.ReplyMarkup) {
	var st *db.UserSettings
	if db.GlobalStore != nil {
		st, _ = db.GlobalStore.GetUserSettings(userID)
	}
	if st == nil {
		st = &db.UserSettings{UserID: userID, ExtraData: make(map[string]string)}
	}

	switch page {
	case "main", "":
		text := fmt.Sprintf(`⚜ <b><u>User Settings Dashboard</u></b>
│
┠ <b>Target:</b> <code>#ID%d</code>
┠ <b>Upload Destination:</b> <b>%s</b>
┠ <b>Leech Type:</b> <b>%s</b>
┠ <b>Split Size:</b> <b>%s</b>
┠ <b>Prefix:</b> %s
┠ <b>Suffix:</b> %s
┠ <b>Metadata:</b> %s
┖ <b>Thumbnail:</b> %s`,
			userID,
			formatOpt(st.Get("DEFAULT_UPLOAD"), "GDRIVE API"),
			formatBool(st.AsDoc, "📄 Document", "🎥 Media"),
			formatSplit(st.SplitSize),
			formatOpt(st.Prefix, "None"),
			formatOpt(st.Suffix, "None"),
			formatOpt(st.Get("METADATA"), "None"),
			formatThumbStatus(st.ThumbPath),
		)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("⚙️ General Settings", fmt.Sprintf("userset %d general", userID)),
				tg.Button.Data("📤 Leech Settings", fmt.Sprintf("userset %d leech", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🪞 Mirror Settings", fmt.Sprintf("userset %d mirror", userID)),
				tg.Button.Data("☁️ Uphoster Settings", fmt.Sprintf("userset %d uphoster", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🎬 FF Media Settings", fmt.Sprintf("userset %d ffset", userID)),
				tg.Button.Data("📑 Clone Settings", fmt.Sprintf("userset %d clone", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("📺 YT-DLP Settings", fmt.Sprintf("userset %d ytdlp", userID)),
				tg.Button.Data("⚡️ Misc Settings", fmt.Sprintf("userset %d advanced", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("📦 Export Settings", fmt.Sprintf("userset %d export", userID)),
				tg.Button.Data("📥 Import Settings", fmt.Sprintf("userset %d import", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔄 Reset All", fmt.Sprintf("userset %d confirm_reset_all", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "general":
		du := st.Get("DEFAULT_UPLOAD")
		if du == "" {
			du = "gd"
		}
		duLabel := "GDRIVE API"
		swapTarget := "RCLONE"
		if du == "rc" {
			duLabel = "RCLONE"
			swapTarget = "GDRIVE API"
		}

		userTokens := st.Get("USER_TOKENS") == "true"
		tokenMode := "OWNER"
		swapTokenTarget := "USER"
		if userTokens {
			tokenMode = "USER"
			swapTokenTarget = "OWNER"
		}

		text := fmt.Sprintf(`⚜ <b><u>General Settings</u></b>
│
┠ <b>Default Upload Package:</b> <b>%s</b>
┖ <b>Default Usage Mode:</b> <b>%s's</b> token/config`, duLabel, tokenMode)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data(fmt.Sprintf("Swap to %s Mode", swapTarget), fmt.Sprintf("userset %d %s", userID, du)),
			),
			tg.Button.Row(
				tg.Button.Data(fmt.Sprintf("Swap to %s token/config", swapTokenTarget), fmt.Sprintf("userset %d tog USER_TOKENS %s", userID, boolStr(!userTokens))),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "leech":
		docType := "MEDIA"
		docToggle := "Send As Document"
		docTarget := "t"
		if st.AsDoc {
			docType = "DOCUMENT"
			docToggle = "Send As Media"
			docTarget = "f"
		}

		eqSplits := st.Get("EQUAL_SPLITS") == "true"
		eqToggle := "Enable Equal Splits"
		eqTarget := "t"
		eqLabel := "Disabled"
		if eqSplits {
			eqToggle = "Disable Equal Splits"
			eqTarget = "f"
			eqLabel = "Enabled"
		}

		mediaGroup := st.Get("MEDIA_GROUP") == "true"
		mgToggle := "Enable Media Group"
		mgTarget := "t"
		mgLabel := "Disabled"
		if mediaGroup {
			mgToggle = "Disable Media Group"
			mgTarget = "f"
			mgLabel = "Enabled"
		}

		text := fmt.Sprintf(`⚜ <b><u>Leech Settings</u></b>
│
┠ <b>Leech Type:</b> <b>%s</b>
┠ <b>Leech Split Size:</b> <b>%s</b>
┠ <b>Equal Splits:</b> <b>%s</b>
┠ <b>Media Group:</b> <b>%s</b>
┠ <b>Leech Prefix:</b> %s
┠ <b>Leech Suffix:</b> %s
┠ <b>Leech Caption:</b> %s
┖ <b>Leech Dump Chats:</b> %s`,
			docType,
			formatSplit(st.SplitSize),
			eqLabel,
			mgLabel,
			formatOpt(st.Prefix, "Not Exists"),
			formatOpt(st.Suffix, "Not Exists"),
			formatOpt(st.Caption, "Not Exists"),
			formatOpt(st.Get("LEECH_DUMP_CHATS"), "None"),
		)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🖼 Thumbnail Settings", fmt.Sprintf("userset %d thumb", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("📏 Leech Split Size", fmt.Sprintf("userset %d menu LEECH_SPLIT_SIZE", userID)),
				tg.Button.Data("📦 Leech Dump Chats", fmt.Sprintf("userset %d menu LEECH_DUMP_CHATS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🏷 Leech Prefix", fmt.Sprintf("userset %d menu LEECH_PREFIX", userID)),
				tg.Button.Data("🏷 Leech Suffix", fmt.Sprintf("userset %d menu LEECH_SUFFIX", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("📝 Leech Caption", fmt.Sprintf("userset %d menu LEECH_CAPTION", userID)),
			),
			tg.Button.Row(
				tg.Button.Data(docToggle, fmt.Sprintf("userset %d tog AS_DOCUMENT %s", userID, docTarget)),
			),
			tg.Button.Row(
				tg.Button.Data(eqToggle, fmt.Sprintf("userset %d tog EQUAL_SPLITS %s", userID, eqTarget)),
				tg.Button.Data(mgToggle, fmt.Sprintf("userset %d tog MEDIA_GROUP %s", userID, mgTarget)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "thumb":
		hasThumb := false
		if st.ThumbPath != "" {
			if _, err := os.Stat(st.ThumbPath); err == nil {
				hasThumb = true
			}
		}

		autoThumb := st.Get("AUTO_THUMBNAIL") == "true"
		atLabel := "Disabled"
		atToggle := "Enable Auto Thumbnail"
		atTarget := "t"
		if autoThumb {
			atLabel = "Enabled"
			atToggle = "Disable Auto Thumbnail"
			atTarget = "f"
		}

		layout := st.Get("THUMBNAIL_LAYOUT")
		if layout == "" {
			layout = "None"
		}

		text := fmt.Sprintf(`⚜ <b><u>Thumbnail Settings</u></b>
│
┠ <b>Custom Thumbnail:</b> <b>%s</b>
┠ <b>Auto Thumbnail:</b> <b>%s</b>
┖ <b>Layout:</b> <b>%s</b>`,
			formatBool(hasThumb, "Exists ✅", "Not Exists ❌"),
			atLabel,
			layout,
		)

		var thumbButtons []*tg.KeyboardButtonRow
		if hasThumb {
			thumbButtons = append(thumbButtons, tg.Button.Row(
				tg.Button.Data("✏️ Change Thumbnail", fmt.Sprintf("userset %d menu THUMBNAIL", userID)),
				tg.Button.Data("👁 View Thumb", fmt.Sprintf("userset %d view THUMBNAIL", userID)),
			))
			thumbButtons = append(thumbButtons, tg.Button.Row(
				tg.Button.Data("🗑 Delete Thumb", fmt.Sprintf("userset %d remove THUMBNAIL", userID)),
			))
		} else {
			thumbButtons = append(thumbButtons, tg.Button.Row(
				tg.Button.Data("🖼 Set Thumbnail", fmt.Sprintf("userset %d menu THUMBNAIL", userID)),
			))
		}

		thumbButtons = append(thumbButtons, tg.Button.Row(
			tg.Button.Data(atToggle, fmt.Sprintf("userset %d tog AUTO_THUMBNAIL %s", userID, atTarget)),
			tg.Button.Data("📐 Layout", fmt.Sprintf("userset %d menu THUMBNAIL_LAYOUT", userID)),
		))
		thumbButtons = append(thumbButtons, tg.Button.Row(
			tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back leech", userID)),
			tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
		))

		markup := tg.Button.Keyboard(thumbButtons...)
		return text, markup

	case "ffset":
		text := fmt.Sprintf(`⚜ <b><u>FF Media & Metadata Settings</u></b>
│
┠ <b>FFmpeg Cmds:</b> <code>%s</code>
┠ <b>Global Metadata:</b> <code>%s</code>
┠ <b>Audio Metadata:</b> <code>%s</code>
┠ <b>Video Metadata:</b> <code>%s</code>
┖ <b>Subtitle Metadata:</b> <code>%s</code>`,
			formatOpt(st.Get("FFMPEG_CMDS"), "None"),
			formatOpt(st.Get("METADATA"), "None"),
			formatOpt(st.Get("AUDIO_METADATA"), "None"),
			formatOpt(st.Get("VIDEO_METADATA"), "None"),
			formatOpt(st.Get("SUBTITLE_METADATA"), "None"),
		)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("⚡️ FFmpeg Cmds", fmt.Sprintf("userset %d menu FFMPEG_CMDS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🏷 Global Metadata", fmt.Sprintf("userset %d menu METADATA", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🎵 Audio Metadata", fmt.Sprintf("userset %d menu AUDIO_METADATA", userID)),
				tg.Button.Data("🎬 Video Metadata", fmt.Sprintf("userset %d menu VIDEO_METADATA", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("💬 Subtitle Metadata", fmt.Sprintf("userset %d menu SUBTITLE_METADATA", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "mirror":
		sd := st.Get("STOP_DUPLICATE") == "true"
		sdLabel := "Disabled"
		sdToggle := "Enable Stop Duplicate"
		sdTarget := "t"
		if sd {
			sdLabel = "Enabled"
			sdToggle = "Disable Stop Duplicate"
			sdTarget = "f"
		}

		dc := st.Get("drive_cat_mode") == "true"
		dcLabel := "OFF"
		if dc {
			dcLabel = "ON"
		}

		text := fmt.Sprintf(`⚜ <b><u>Mirror Settings</u></b>
│
┠ <b>Bot Stop Duplicate:</b> <b>%s</b>
┖ <b>Drive Categories:</b> <b>%s</b>`, sdLabel, dcLabel)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("📁 RClone Tools", fmt.Sprintf("userset %d rclone", userID)),
				tg.Button.Data("☁️ GDrive Tools", fmt.Sprintf("userset %d gdrive", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔴 Mega Tools", fmt.Sprintf("userset %d mega", userID)),
				tg.Button.Data("🌱 Seedr Tools", fmt.Sprintf("userset %d seedr", userID)),
			),
			tg.Button.Row(
				tg.Button.Data(fmt.Sprintf("Drive Categories: %s", dcLabel), fmt.Sprintf("userset %d tog drive_cat_mode %s", userID, boolStr(!dc))),
				tg.Button.Data(sdToggle, fmt.Sprintf("userset %d tog STOP_DUPLICATE %s", userID, sdTarget)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "rclone":
		rccPath := st.Get("RCLONE_PATH")
		if rccPath == "" {
			rccPath = "None"
		}
		rcFlags := st.Get("RCLONE_FLAGS")
		if rcFlags == "" {
			rcFlags = "None"
		}

		text := fmt.Sprintf(`⚜ <b><u>RClone Settings</u></b>
│
┠ <b>Rclone Config:</b> <b>%s</b>
┠ <b>Rclone Flags:</b> <code>%s</code>
┖ <b>Rclone Path:</b> <code>%s</code>`,
			formatBool(st.Get("RCLONE_CONFIG") != "", "Exists ✅", "Not Exists ❌"),
			rcFlags,
			rccPath,
		)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("📄 Rclone Config", fmt.Sprintf("userset %d menu RCLONE_CONFIG", userID)),
				tg.Button.Data("📂 Default Path", fmt.Sprintf("userset %d menu RCLONE_PATH", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🚩 Rclone Flags", fmt.Sprintf("userset %d menu RCLONE_FLAGS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back mirror", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "gdrive":
		gdid := st.Get("GDRIVE_ID")
		if gdid == "" {
			gdid = "None"
		}
		index := st.Get("INDEX_URL")
		if index == "" {
			index = "None"
		}

		text := fmt.Sprintf(`⚜ <b><u>GDrive Tools Settings</u></b>
│
┠ <b>GDrive ID:</b> <code>%s</code>
┠ <b>Index URL:</b> <code>%s</code>
┠ <b>Token.pickle:</b> <b>%s</b>
┖ <b>Categories:</b> <code>%s</code>`,
			gdid,
			index,
			formatBool(st.Get("TOKEN_PICKLE") != "", "Exists ✅", "Not Exists ❌"),
			formatOpt(st.Get("DRIVE_CAT"), "None"),
		)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🆔 Gdrive ID", fmt.Sprintf("userset %d menu GDRIVE_ID", userID)),
				tg.Button.Data("🔗 Index URL", fmt.Sprintf("userset %d menu INDEX_URL", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔑 Token.pickle", fmt.Sprintf("userset %d menu TOKEN_PICKLE", userID)),
				tg.Button.Data("📁 Drive Categories", fmt.Sprintf("userset %d menu DRIVE_CAT", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back mirror", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "uphoster":
		uph := st.Get("UPHOSTER_SERVICE")
		if uph == "" {
			uph = "gofile"
		}

		text := fmt.Sprintf(`⚜ <b><u>Uphoster Settings</u></b>
│
┖ <b>Current Destination:</b> <code>%s</code>`, uph)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🔀 Change Destination ➔", fmt.Sprintf("userset %d uphoster_destinations", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("📁 Gofile Tools", fmt.Sprintf("userset %d gofile", userID)),
				tg.Button.Data("⚡️ BuzzHeavier", fmt.Sprintf("userset %d buzzheavier", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("💧 PixelDrain", fmt.Sprintf("userset %d pixeldrain", userID)),
				tg.Button.Data("📦 DevUploads", fmt.Sprintf("userset %d devuploads", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("⚔️ VikingFile", fmt.Sprintf("userset %d vikingfile", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "gofile":
		text := fmt.Sprintf(`⚜ <b><u>Gofile Settings</u></b>
│
┠ <b>Token:</b> <code>%s</code>
┖ <b>Folder ID:</b> <code>%s</code>`,
			formatOpt(st.Get("GOFILE_TOKEN"), "None"),
			formatOpt(st.Get("GOFILE_FOLDER_ID"), "Root"),
		)
		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🔑 Gofile Token", fmt.Sprintf("userset %d menu GOFILE_TOKEN", userID)),
				tg.Button.Data("📂 Folder ID", fmt.Sprintf("userset %d menu GOFILE_FOLDER_ID", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back uphoster", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "buzzheavier":
		text := fmt.Sprintf(`⚜ <b><u>BuzzHeavier Settings</u></b>
│
┠ <b>Token:</b> <code>%s</code>
┖ <b>Folder ID:</b> <code>%s</code>`,
			formatOpt(st.Get("BUZZHEAVIER_TOKEN"), "None"),
			formatOpt(st.Get("BUZZHEAVIER_FOLDER_ID"), "Root"),
		)
		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🔑 BuzzHeavier Token", fmt.Sprintf("userset %d menu BUZZHEAVIER_TOKEN", userID)),
				tg.Button.Data("📂 Folder ID", fmt.Sprintf("userset %d menu BUZZHEAVIER_FOLDER_ID", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back uphoster", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "pixeldrain":
		text := fmt.Sprintf(`⚜ <b><u>PixelDrain Settings</u></b>
│
┖ <b>API Key:</b> <code>%s</code>`,
			formatOpt(st.Get("PIXELDRAIN_KEY"), "None"),
		)
		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🔑 PixelDrain Key", fmt.Sprintf("userset %d menu PIXELDRAIN_KEY", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back uphoster", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "devuploads":
		text := fmt.Sprintf(`⚜ <b><u>DevUploads Settings</u></b>
│
┠ <b>API Key:</b> <code>%s</code>
┖ <b>Folder ID:</b> <code>%s</code>`,
			formatOpt(st.Get("DEVUPLOADS_KEY"), "None"),
			formatOpt(st.Get("DEVUPLOADS_FOLDER"), "Root"),
		)
		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🔑 DevUploads Key", fmt.Sprintf("userset %d menu DEVUPLOADS_KEY", userID)),
				tg.Button.Data("📂 Folder ID", fmt.Sprintf("userset %d menu DEVUPLOADS_FOLDER", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back uphoster", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "vikingfile":
		text := fmt.Sprintf(`⚜ <b><u>VikingFile Settings</u></b>
│
┠ <b>Hash:</b> <code>%s</code>
┖ <b>Folder:</b> <code>%s</code>`,
			formatOpt(st.Get("VIKINGFILE_HASH"), "None"),
			formatOpt(st.Get("VIKINGFILE_FOLDER"), "Root"),
		)
		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🔑 VikingFile Hash", fmt.Sprintf("userset %d menu VIKINGFILE_HASH", userID)),
				tg.Button.Data("📂 Folder Path", fmt.Sprintf("userset %d menu VIKINGFILE_FOLDER", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back uphoster", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "advanced":
		text := fmt.Sprintf(`⚜ <b><u>Misc / Advanced Settings</u></b>
│
┠ <b>Excluded Extensions:</b> <code>%s</code>
┠ <b>Name Swap:</b> <code>%s</code>
┖ <b>Upload Paths:</b> <code>%s</code>`,
			formatOpt(st.Get("EXCLUDED_EXTENSIONS"), "None"),
			formatOpt(st.Get("NAME_SWAP"), "None"),
			formatOpt(st.Get("UPLOAD_PATHS"), "None"),
		)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🚫 Excluded Ext", fmt.Sprintf("userset %d menu EXCLUDED_EXTENSIONS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔄 Name Swap", fmt.Sprintf("userset %d menu NAME_SWAP", userID)),
				tg.Button.Data("🗺 Upload Paths", fmt.Sprintf("userset %d menu UPLOAD_PATHS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "clone":
		hasSess := st.Get("USER_SESSION") != ""
		text := fmt.Sprintf(`⚜ <b><u>Clone Settings</u></b>
│
┠ <b>User Session:</b> <b>%s</b>
┠ <b>Destinations:</b> <code>%s</code>
┠ <b>Content Type:</b> <b>%s</b>
┠ <b>Excluded Ext:</b> <code>%s</code>
┖ <b>Regex Filters:</b> <code>%s</code>`,
			formatBool(hasSess, "Exists 🔒", "Not Exists 🔓"),
			formatOpt(st.Get("CLONE_DUMP_CHATS"), "None"),
			formatOpt(st.Get("CLONE_CONTENT_TYPE"), "all"),
			formatOpt(st.Get("CLONE_EXCLUDED_EXTENSIONS"), "None"),
			formatOpt(st.Get("CLONE_FILTERS"), "None"),
		)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🔑 User Session", fmt.Sprintf("userset %d menu USER_SESSION", userID)),
				tg.Button.Data("📦 Dump Chats", fmt.Sprintf("userset %d menu CLONE_DUMP_CHATS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("📑 Content Type", fmt.Sprintf("userset %d menu CLONE_CONTENT_TYPE", userID)),
				tg.Button.Data("🚫 Excluded Ext", fmt.Sprintf("userset %d menu CLONE_EXCLUDED_EXTENSIONS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔍 Regex Filters", fmt.Sprintf("userset %d menu CLONE_FILTERS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "ytdlp":
		text := fmt.Sprintf(`⚜ <b><u>YT-DLP Settings</u></b>
│
┠ <b>Options:</b> <code>%s</code>
┠ <b>Cookie File:</b> <b>%s</b>
┠ <b>Description:</b> <code>%s</code>
┠ <b>Tags:</b> <code>%s</code>
┠ <b>Category ID:</b> <code>%s</code>
┖ <b>Privacy Status:</b> <code>%s</code>`,
			formatOpt(st.Get("YT_DLP_OPTIONS"), "None"),
			formatBool(st.Get("USER_COOKIE_FILE") != "", "Exists ✅", "Not Exists ❌"),
			formatOpt(st.Get("YT_DESP"), "None"),
			formatOpt(st.Get("YT_TAGS"), "None"),
			formatOpt(st.Get("YT_CATEGORY_ID"), "None"),
			formatOpt(st.Get("YT_PRIVACY_STATUS"), "None"),
		)

		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("⚙️ YT-DLP Options", fmt.Sprintf("userset %d menu YT_DLP_OPTIONS", userID)),
				tg.Button.Data("🍪 Cookie File", fmt.Sprintf("userset %d menu USER_COOKIE_FILE", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("📝 Description", fmt.Sprintf("userset %d menu YT_DESP", userID)),
				tg.Button.Data("🏷 Tags", fmt.Sprintf("userset %d menu YT_TAGS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🆔 Category ID", fmt.Sprintf("userset %d menu YT_CATEGORY_ID", userID)),
				tg.Button.Data("🔒 Privacy Status", fmt.Sprintf("userset %d menu YT_PRIVACY_STATUS", userID)),
			),
			tg.Button.Row(
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back", userID)),
				tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
			),
		)
		return text, markup

	case "mega":
		email := st.Get("MEGA_EMAIL")
		hasCreds := email != "" && st.Get("MEGA_PASSWORD") != ""
		text := fmt.Sprintf(`⚜ <b><u>Mega Tools</u></b>
│
┠ <b>Mega Email:</b> <code>%s</code>
┖ <b>Account:</b> <b>%s</b>`,
			formatOpt(email, "Not Set"),
			formatBool(hasCreds, "Configured ✅", "Not Configured ❌"),
		)

		var megaButtons []*tg.KeyboardButtonRow
		megaButtons = append(megaButtons, tg.Button.Row(
			tg.Button.Data("📧 Mega Email", fmt.Sprintf("userset %d menu MEGA_EMAIL", userID)),
		))
		if email != "" {
			megaButtons = append(megaButtons, tg.Button.Row(
				tg.Button.Data("🔑 Mega Password", fmt.Sprintf("userset %d menu MEGA_PASSWORD", userID)),
			))
		}
		if hasCreds {
			megaButtons = append(megaButtons, tg.Button.Row(
				tg.Button.Data("🗑 Remove Account", fmt.Sprintf("userset %d remove MEGA_EMAIL", userID)),
			))
		}
		megaButtons = append(megaButtons, tg.Button.Row(
			tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back mirror", userID)),
			tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
		))
		markup := tg.Button.Keyboard(megaButtons...)
		return text, markup

	case "seedr":
		email := st.Get("SEEDR_EMAIL")
		delFolder := st.Get("SEEDR_DELETE_FOLDER") == "true"
		text := fmt.Sprintf(`⚜ <b><u>Seedr Tools</u></b>
│
┠ <b>Seedr Email:</b> <code>%s</code>
┖ <b>Delete Folder:</b> <b>%s</b>`,
			formatOpt(email, "Not Set"),
			formatBool(delFolder, "ON", "OFF"),
		)

		var seedrButtons []*tg.KeyboardButtonRow
		seedrButtons = append(seedrButtons, tg.Button.Row(
			tg.Button.Data("📧 Seedr Email", fmt.Sprintf("userset %d menu SEEDR_EMAIL", userID)),
		))
		if email != "" {
			seedrButtons = append(seedrButtons, tg.Button.Row(
				tg.Button.Data("🔑 Seedr Password", fmt.Sprintf("userset %d menu SEEDR_PASSWORD", userID)),
			))
		}
		seedrButtons = append(seedrButtons, tg.Button.Row(
			tg.Button.Data(fmt.Sprintf("Delete Folder: %s", boolStr(delFolder)), fmt.Sprintf("userset %d tog SEEDR_DELETE_FOLDER %s", userID, boolStr(!delFolder))),
		))
		seedrButtons = append(seedrButtons, tg.Button.Row(
			tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back mirror", userID)),
			tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
		))
		markup := tg.Button.Keyboard(seedrButtons...)
		return text, markup
	}

	return GetUserSettingsFull(userID, "main")
}

// GetOptionMenu renders the dedicated configuration panel for an individual option
func GetOptionMenu(option string, userID int64) (string, tg.ReplyMarkup) {
	info, exists := OptionCatalog[option]
	if !exists {
		info = OptionInfo{
			Title:       option,
			Description: "Configuration setting.",
			Prompt:      "Send new value.",
			ParentPage:  "main",
		}
	}

	var st *db.UserSettings
	if db.GlobalStore != nil {
		st, _ = db.GlobalStore.GetUserSettings(userID)
	}
	if st == nil {
		st = &db.UserSettings{UserID: userID, ExtraData: make(map[string]string)}
	}

	currVal := st.Get(option)
	if currVal == "" {
		switch option {
		case "LEECH_PREFIX":
			currVal = st.Prefix
		case "LEECH_SUFFIX":
			currVal = st.Suffix
		case "LEECH_CAPTION":
			currVal = st.Caption
		case "LEECH_SPLIT_SIZE":
			if st.SplitSize > 0 {
				currVal = utils.FormatBytes(st.SplitSize)
			}
		case "LEECH_DUMP_CHATS":
			if st.DumpChat != 0 {
				currVal = fmt.Sprintf("%d", st.DumpChat)
			}
		case "THUMBNAIL":
			if st.ThumbPath != "" {
				currVal = "Active"
			}
		}
	}
	if currVal == "" {
		currVal = "Not Set"
	}

	text := fmt.Sprintf(`⚜ <b><u>%s</u></b>
│
┠ <b>Description:</b> <i>%s</i>
┖ <b>Current Value:</b> <code>%s</code>`, info.Title, info.Description, html.EscapeString(currVal))

	var rows []*tg.KeyboardButtonRow
	setLabel := "✏️ Set"
	if currVal != "Not Set" {
		setLabel = "✏️ Change"
	}

	if currVal != "Not Set" {
		rows = append(rows, tg.Button.Row(
			tg.Button.Data(setLabel, fmt.Sprintf("userset %d set %s", userID, option)),
			tg.Button.Data("🔄 Reset", fmt.Sprintf("userset %d reset %s", userID, option)),
		))
	} else {
		rows = append(rows, tg.Button.Row(
			tg.Button.Data(setLabel, fmt.Sprintf("userset %d set %s", userID, option)),
		))
	}

	if option == "THUMBNAIL" && st.ThumbPath != "" {
		rows = append(rows, tg.Button.Row(
			tg.Button.Data("👁 View Thumb", fmt.Sprintf("userset %d view THUMBNAIL", userID)),
			tg.Button.Data("🗑 Delete Thumb", fmt.Sprintf("userset %d remove THUMBNAIL", userID)),
		))
	}

	rows = append(rows, tg.Button.Row(
		tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back %s", userID, info.ParentPage)),
		tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", userID)),
	))

	return text, tg.Button.Keyboard(rows...)
}

// HandleUsersetCallback handles callback actions adhering strictly to the WZML-X syntax
func HandleUsersetCallback(cb *tg.CallbackQuery, botClient *tg.Client) error {
	data := strings.Fields(cb.DataString())
	if len(data) < 3 {
		return nil
	}

	var targetUID int64
	fmt.Sscanf(data[1], "%d", &targetUID)
	if targetUID != 0 && targetUID != cb.SenderID && !auth.IsSudo(cb.SenderID) {
		_, _ = cb.Answer("Not Yours!", &tg.CallbackOptions{Alert: true})
		return nil
	}

	action := data[2]

	switch action {
	case "close":
		_, _ = cb.Answer("", &tg.CallbackOptions{})
		_, _ = cb.Delete()
		return nil

	case "back":
		backTo := "main"
		if len(data) >= 4 {
			backTo = data[3]
		}
		text, markup := GetUserSettingsFull(targetUID, backTo)
		_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
		_, _ = cb.Answer("", &tg.CallbackOptions{})
		return nil

	case "general", "mirror", "leech", "thumb", "ffset", "uphoster", "advanced", "clone", "ytdlp", "mega", "seedr", "rclone", "gdrive", "gofile", "buzzheavier", "pixeldrain", "devuploads", "vikingfile":
		text, markup := GetUserSettingsFull(targetUID, action)
		_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
		_, _ = cb.Answer("", &tg.CallbackOptions{})
		return nil

	case "menu":
		if len(data) >= 4 {
			opt := data[3]
			if len(data) >= 5 && data[4] == "stop" {
				ClearPendingInput(targetUID)
				_, _ = cb.Answer("Prompt cancelled.", &tg.CallbackOptions{})
			}
			text, markup := GetOptionMenu(opt, targetUID)
			_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
			_, _ = cb.Answer("", &tg.CallbackOptions{})
		}
		return nil

	case "set":
		if len(data) >= 4 {
			opt := data[3]
			info, exists := OptionCatalog[opt]
			if !exists {
				info = OptionInfo{Title: opt, Prompt: "Send value."}
			}
			SetPendingInput(targetUID, opt, info.ParentPage)

			promptText := fmt.Sprintf(`⚜ <b><u>Set %s</u></b>
│
%s`, info.Title, info.Prompt)

			markup := tg.Button.Keyboard(
				tg.Button.Row(
					tg.Button.Data("🛑 Cancel / Stop", fmt.Sprintf("userset %d menu %s stop", targetUID, opt)),
				),
				tg.Button.Row(
					tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d menu %s", targetUID, opt)),
					tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", targetUID)),
				),
			)

			_, _ = cb.Edit(promptText, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
			_, _ = cb.Answer("", &tg.CallbackOptions{})
		}
		return nil

	case "reset":
		if len(data) >= 4 && db.GlobalStore != nil {
			opt := data[3]
			st, _ := db.GlobalStore.GetUserSettings(targetUID)
			if st != nil {
				st.Delete(opt)
				switch opt {
				case "LEECH_PREFIX":
					st.Prefix = ""
				case "LEECH_SUFFIX":
					st.Suffix = ""
				case "LEECH_CAPTION":
					st.Caption = ""
				case "LEECH_SPLIT_SIZE":
					st.SplitSize = 0
				case "LEECH_DUMP_CHATS":
					st.DumpChat = 0
				}
				_ = db.GlobalStore.SaveUserSettings(st)
			}
			_, _ = cb.Answer("Reset successful!", &tg.CallbackOptions{})
			text, markup := GetOptionMenu(opt, targetUID)
			_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
		}
		return nil

	case "remove":
		if len(data) >= 4 && db.GlobalStore != nil {
			opt := data[3]
			st, _ := db.GlobalStore.GetUserSettings(targetUID)
			if st != nil {
				if opt == "THUMBNAIL" {
					if st.ThumbPath != "" {
						_ = os.Remove(st.ThumbPath)
					}
					st.ThumbPath = ""
				} else {
					st.Delete(opt)
				}
				_ = db.GlobalStore.SaveUserSettings(st)
			}
			_, _ = cb.Answer("Removed successfully!", &tg.CallbackOptions{Alert: true})
			text, markup := GetOptionMenu(opt, targetUID)
			_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
		}
		return nil

	case "tog":
		if len(data) >= 5 && db.GlobalStore != nil {
			key := data[3]
			val := data[4] == "t"
			st, _ := db.GlobalStore.GetUserSettings(targetUID)
			if st == nil {
				st = &db.UserSettings{UserID: targetUID, ExtraData: make(map[string]string)}
			}

			if key == "AS_DOCUMENT" {
				st.AsDoc = val
			} else {
				st.Set(key, boolStr(val))
			}
			_ = db.GlobalStore.SaveUserSettings(st)

			backTo := "leech"
			if key == "STOP_DUPLICATE" {
				backTo = "gdrive"
			} else if key == "drive_cat_mode" {
				backTo = "mirror"
			} else if key == "USER_TOKENS" {
				backTo = "general"
			} else if key == "AUTO_THUMBNAIL" {
				backTo = "thumb"
			} else if key == "SEEDR_DELETE_FOLDER" {
				backTo = "seedr"
			}

			text, markup := GetUserSettingsFull(targetUID, backTo)
			_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
			_, _ = cb.Answer("", &tg.CallbackOptions{})
		}
		return nil

	case "view":
		if len(data) >= 4 && data[3] == "THUMBNAIL" && db.GlobalStore != nil {
			st, _ := db.GlobalStore.GetUserSettings(targetUID)
			if st != nil && st.ThumbPath != "" {
				if _, err := os.Stat(st.ThumbPath); err == nil {
					_, _ = cb.Answer("Sending thumbnail...", &tg.CallbackOptions{})
					_, _ = botClient.SendMedia(cb.SenderID, st.ThumbPath, &tg.MediaOptions{
						Caption: "🖼 <b>Your Current Custom Thumbnail</b>",
					})
					return nil
				}
			}
			_, _ = cb.Answer("No custom thumbnail found. Send a photo in private chat to set one!", &tg.CallbackOptions{Alert: true})
		}
		return nil

	case "confirm_reset_all":
		confirmText := `⚠️ <b><u>Confirm Reset All Settings</u></b>
│
┖ <i>Are you sure you want to reset all your custom user settings back to system defaults?</i>`
		markup := tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("🔴 Yes, Reset All", fmt.Sprintf("userset %d do_reset_all", targetUID)),
				tg.Button.Data("🔙 Back", fmt.Sprintf("userset %d back", targetUID)),
			),
		)
		_, _ = cb.Edit(confirmText, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
		_, _ = cb.Answer("", &tg.CallbackOptions{})
		return nil

	case "do_reset_all":
		if db.GlobalStore != nil {
			st, _ := db.GlobalStore.GetUserSettings(targetUID)
			if st != nil && st.ThumbPath != "" {
				_ = os.Remove(st.ThumbPath)
			}
			cleanSettings := &db.UserSettings{
				UserID:    targetUID,
				ExtraData: make(map[string]string),
			}
			_ = db.GlobalStore.SaveUserSettings(cleanSettings)
		}
		_, _ = cb.Answer("All settings have been reset to defaults!", &tg.CallbackOptions{Alert: true})
		text, markup := GetUserSettingsFull(targetUID, "main")
		_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
		return nil

	case "gd", "rc":
		if db.GlobalStore != nil {
			st, _ := db.GlobalStore.GetUserSettings(targetUID)
			if st == nil {
				st = &db.UserSettings{UserID: targetUID, ExtraData: make(map[string]string)}
			}
			next := "rc"
			if action == "rc" {
				next = "gd"
			}
			st.Set("DEFAULT_UPLOAD", next)
			_ = db.GlobalStore.SaveUserSettings(st)
			text, markup := GetUserSettingsFull(targetUID, "general")
			_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
			_, _ = cb.Answer("", &tg.CallbackOptions{})
		}
		return nil
	}

	return nil
}

// Helpers
func formatOpt(val, fallback string) string {
	if val == "" {
		return fmt.Sprintf("<i>%s</i>", fallback)
	}
	return fmt.Sprintf("<code>%s</code>", html.EscapeString(val))
}

func formatBool(val bool, trueLabel, falseLabel string) string {
	if val {
		return trueLabel
	}
	return falseLabel
}

func formatSplit(size int64) string {
	if size <= 0 {
		return "2 GB (Standard)"
	}
	return utils.FormatBytes(size)
}

func formatThumbStatus(path string) string {
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			return "✅ <i>Custom Active</i>"
		}
	}
	return "❌ <i>Not Set</i>"
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// ProcessPendingUserInput processes an incoming user message if an interactive setting prompt is active
func ProcessPendingUserInput(m *tg.NewMessage, pending *PendingUserState, botClient *tg.Client) error {
	senderID := m.SenderID()
	ClearPendingInput(senderID)

	text := strings.TrimSpace(m.Text())
	if text == "/cancel" || text == "/stop" {
		_, err := m.Reply("🛑 Input cancelled.")
		return err
	}

	if db.GlobalStore == nil {
		_, err := m.Reply("Database not initialized.")
		return err
	}

	st, _ := db.GlobalStore.GetUserSettings(senderID)
	if st == nil {
		st = &db.UserSettings{UserID: senderID, ExtraData: make(map[string]string)}
	}

	displayVal := text

	switch pending.Option {
	case "THUMBNAIL":
		if m.Photo() != nil {
			thumbDir := filepath.Join("data", "thumbnails")
			_ = os.MkdirAll(thumbDir, 0755)
			tPath := filepath.Join(thumbDir, fmt.Sprintf("%d.jpg", senderID))
			if _, err := m.Download(&tg.DownloadOptions{FileName: tPath}); err == nil {
				st.ThumbPath = tPath
				displayVal = "Photo saved as custom thumbnail"
			}
		} else if strings.HasPrefix(text, "http") {
			st.ThumbPath = text
			displayVal = text
		}
	case "LEECH_PREFIX":
		st.Prefix = text
	case "LEECH_SUFFIX":
		st.Suffix = text
	case "LEECH_CAPTION":
		st.Caption = text
	case "LEECH_SPLIT_SIZE":
		parsed := utils.ParseSize(text)
		if parsed > 0 {
			st.SplitSize = parsed
			displayVal = utils.FormatBytes(parsed)
		} else {
			displayVal = "Invalid size, default retained"
		}
	case "LEECH_DUMP_CHATS":
		var dump int64
		if _, err := fmt.Sscanf(text, "%d", &dump); err == nil {
			st.DumpChat = dump
		}
		st.Set("LEECH_DUMP_CHATS", text)
	case "RCLONE_CONFIG":
		if m.Document() != nil {
			rcDir := "rclone"
			_ = os.MkdirAll(rcDir, 0755)
			rcPath := filepath.Join(rcDir, fmt.Sprintf("%d.conf", senderID))
			_, _ = m.Download(&tg.DownloadOptions{FileName: rcPath})
			st.Set("RCLONE_CONFIG", rcPath)
			displayVal = "rclone.conf saved"
		}
	case "TOKEN_PICKLE":
		if m.Document() != nil {
			tokDir := "tokens"
			_ = os.MkdirAll(tokDir, 0755)
			tokPath := filepath.Join(tokDir, fmt.Sprintf("%d.pickle", senderID))
			_, _ = m.Download(&tg.DownloadOptions{FileName: tokPath})
			st.Set("TOKEN_PICKLE", tokPath)
			displayVal = "token.pickle saved"
		}
	case "USER_COOKIE_FILE":
		if m.Document() != nil {
			cDir := filepath.Join("cookies", fmt.Sprintf("%d", senderID))
			_ = os.MkdirAll(cDir, 0755)
			cPath := filepath.Join(cDir, "cookies.txt")
			_, _ = m.Download(&tg.DownloadOptions{FileName: cPath})
			st.Set("USER_COOKIE_FILE", cPath)
			displayVal = "cookies.txt saved"
		}
	default:
		st.Set(pending.Option, text)
	}

	_ = db.GlobalStore.SaveUserSettings(st)

	title := pending.Option
	if info, ok := OptionCatalog[pending.Option]; ok {
		title = info.Title
	}

	respMsg := fmt.Sprintf(`⚜ <b><u>%s Saved!</u></b>
│
┠ <b>Option:</b> %s
┠ <b>Value:</b> <code>%s</code>
┖ <b>Status:</b> ✅ Updated successfully!`, title, pending.Option, html.EscapeString(displayVal))

	markup := tg.Button.Keyboard(
		tg.Button.Row(
			tg.Button.Data("⚙️ Back to Settings", fmt.Sprintf("userset %d %s", senderID, pending.Category)),
			tg.Button.Data("❌ Close", fmt.Sprintf("userset %d close", senderID)),
		),
	)

	_, err := m.Reply(respMsg, &tg.SendOptions{
		ParseMode:   "html",
		ReplyMarkup: markup,
	})
	return err
}
