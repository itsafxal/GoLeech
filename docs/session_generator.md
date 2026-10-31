# Session Management & Direct MTProto Authentication

## 1. Why Third-Party Session Bots Must Be Avoided
Many Telegram bots in the Python ecosystem instruct users to use third-party generator bots (such as `@GeneratorSessionRobot` or web session apps). For GoLeech, this is **both incompatible and unsafe**:

1. **Protocol & Serialization Incompatibility**: Third-party bots serialize sessions specifically formatted for Pyrogram or Telethon (Python struct pack format). Gogram is a 100% native Go MTProto implementation that uses Gogram's internal binary and JSON/Base64 MTProto session schema. A Pyrogram session string cannot be deserialized by Gogram.
2. **Severe Security Vulnerability**: Authenticating via third-party bots sends your Telegram phone number, login OTP code, and Two-Factor Authentication (2FA) password to an unverified third-party server. This exposes your personal account, private chats, and cloud storage to hijacking.

---

## 2. GoLeech Direct Session Generator

GoLeech includes a built-in, secure CLI utility (`cmd/login`) that communicates directly with Telegram's official MTProto servers from your own machine or VPS:

```bash
# Run the interactive generator directly
go run ./cmd/login
```

Or run the compiled binary:
```bash
./bin/goleech-login
```

### Interactive Login Workflow:
1. Prompts for your Telegram **App ID** and **App Hash** (obtainable from [my.telegram.org](https://my.telegram.org)).
2. Prompts for your **Phone Number** (in international format, e.g. `+1234567890`).
3. Telegram delivers an official login code to your Telegram app or SMS.
4. If Two-Factor Authentication (2FA cloud password) is enabled, the CLI securely reads your password with terminal masking (hidden keystrokes).
5. The client authenticates via Gogram MTProto and outputs your native `USER_SESSION_STRING`.

---

## 3. Configuring the Session in GoLeech

Copy the generated session string into your `.env` file:

```env
USER_SESSION_STRING="your_generated_gogram_session_string_here"
```

### Benefits of User Session:
- **4GB File Uploads**: If your user account has Telegram Premium, GoLeech automatically enables 4GB leech split limits instead of the standard bot 2GB limit.
- **Direct Link Extraction**: Access restricted channels and media where your user account is a member.
- **Zero Log Exposure**: GoLeech never prints or exposes session strings in logs.

