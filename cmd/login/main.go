package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	tg "github.com/amarnathcjd/gogram/telegram"
	"golang.org/x/term"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=====================================================")
	fmt.Println("  GoLeech - Native Gogram MTProto Session Generator  ")
	fmt.Println("=====================================================")
	fmt.Println("NOTE: Never use unverified 3rd-party bots (like @GeneratorSessionRobot)")
	fmt.Println("to generate Telegram sessions. This tool securely connects directly")
	fmt.Println("to Telegram's MTProto servers using Gogram.")
	fmt.Println("=====================================================")

	fmt.Print("\nEnter Telegram App ID (e.g. 1234567): ")
	appIDStr, _ := reader.ReadString('\n')
	appIDInt, err := strconv.Atoi(strings.TrimSpace(appIDStr))
	if err != nil || appIDInt == 0 {
		fmt.Println("❌ Error: Invalid App ID.")
		return
	}

	fmt.Print("Enter Telegram App Hash: ")
	appHash, _ := reader.ReadString('\n')
	appHash = strings.TrimSpace(appHash)
	if appHash == "" {
		fmt.Println("❌ Error: App Hash cannot be empty.")
		return
	}

	fmt.Print("Enter Phone Number (with international code, e.g. +1234567890): ")
	phone, _ := reader.ReadString('\n')
	phone = strings.TrimSpace(phone)
	if phone == "" {
		fmt.Println("❌ Error: Phone number cannot be empty.")
		return
	}

	opts := &tg.LoginOptions{
		CodeCallback: func() (string, error) {
			fmt.Print("\n💬 Enter the Telegram OTP code received: ")
			code, _ := reader.ReadString('\n')
			return strings.TrimSpace(code), nil
		},
		PasswordCallback: func() (string, error) {
			fmt.Print("\n🔐 Two-Factor Authentication (2FA) Password: ")
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(bytePassword)), nil
		},
	}

	fmt.Println("\n⏳ Connecting directly to Telegram MTProto servers...")
	client, err := tg.QuickPhone(int32(appIDInt), appHash, phone, opts)
	if err != nil {
		fmt.Printf("\n❌ Authentication failed: %v\n", err)
		return
	}

	sessionString := client.ExportSession()
	fmt.Println("\n=====================================================")
	fmt.Println("✅ Successfully Authenticated with Telegram MTProto!")
	fmt.Println("=====================================================")
	fmt.Println("\nYour Gogram USER_SESSION_STRING:")
	fmt.Println("-----------------------------------------------------")
	fmt.Println(sessionString)
	fmt.Println("-----------------------------------------------------")
	fmt.Println("🔒 Copy this string and set it as USER_SESSION_STRING in your .env")
	fmt.Println("Keep this secret! Anyone with this string can access your Telegram account.")
}
