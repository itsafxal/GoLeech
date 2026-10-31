# Contributing to GoLeech

Thank you for your interest in contributing to **GoLeech**!

## Development Guidelines
1. **Language:** Go 1.24+ (pure Go preferred; avoid CGO where possible).
2. **Telegram MTProto Framework:** Gogram (`github.com/amarnathcjd/gogram`).
3. **Architecture:** Maintain modular separation between `internal/bot`, `internal/task`, `internal/downloader`, `internal/providers`, and `internal/web`.
4. **Code Quality:**
   - Format all Go files with `gofmt -s -w .`
   - Verify code with `go vet ./...`
   - Run the test suite: `go test -v ./...`
5. **No Regressions:** Do not submit code that breaks existing tests or introduces goroutine leaks. Always propagate `context.Context` to handle cancellation.

## Reporting Bugs
Open an issue on GitHub with:
- Exact command / flag used
- Logs and error messages
- Environment details (OS, Docker vs Native, Go version)

