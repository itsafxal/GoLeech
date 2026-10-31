# Security Policy

## Reporting Security Vulnerabilities

If you discover a security vulnerability within **GoLeech**, please DO NOT create a public GitHub issue.

Please report vulnerabilities privately to the project maintainers or via GitHub Security Advisories.

## Security Best Practices in GoLeech
1. **No Third-Party Session Generators:** Never generate sessions via third-party Telegram bots. Always use the built-in `cmd/login` CLI tool that connects directly to Telegram's MTProto servers.
2. **ZipSlip Protection:** GoLeech sanitizes all archive extraction paths to protect against directory traversal attacks (`../`).
3. **Secret Masking:** Sensitive session strings, bot tokens, and passwords are never output in standard application logs.
4. **Web UI Protection:** Secure your web dashboard and streaming endpoints by configuring a strong `WEB_ACCESS_PASSWORD` in `.env`.

