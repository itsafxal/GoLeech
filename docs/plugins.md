# GoLeech Plugin & Extension System

GoLeech provides a native, type-safe Go plugin architecture. Unlike Python plugin systems that require runtime `pip install` and dynamic `importlib` calls that risk crashing the entire process, GoLeech plugins run with error isolation, strong types, and lifecycle hooks.

---

## 1. Plugin Interface

Any module implementing `plugin.Plugin` can be registered into GoLeech:

```go
package myplugin

import (
    "context"
    "github.com/GoLeech/GoLeech/internal/plugin"
    tg "github.com/amarnathcjd/gogram/telegram"
)

type MyCustomPlugin struct{}

func (p *MyCustomPlugin) Name() string        { return "custom_plugin" }
func (p *MyCustomPlugin) Description() string { return "My awesome custom feature" }
func (p *MyCustomPlugin) Version() string     { return "1.0.0" }
func (p *MyCustomPlugin) Close() error        { return nil }

func (p *MyCustomPlugin) Init(ctx context.Context, reg *plugin.Registry) error {
    // Register custom Telegram command
    reg.RegisterCommand("mycommand", func(m *tg.NewMessage) error {
        _, err := m.Reply("Hello from MyCustomPlugin!", &tg.SendOptions{})
        return err
    })

    // Register lifecycle event hook
    reg.RegisterHook(plugin.EventPreTask, func(ctx context.Context, t *task.Task) error {
        // Run pre-task inspection or logging
        return nil
    })

    return nil
}
```

---

## 2. Supported Lifecycle Hooks

- `EventPreTask`: Triggered before any download begins. Can be used for custom quota enforcement, URL rewriting, or metadata checking.
- `EventPostTask`: Triggered after download and upload complete. Can be used for custom webhook alerts or database archiving.
- `EventOnStart`: Executed when GoLeech starts up.
- `EventOnStop`: Executed during graceful shutdown.

---

## 3. Built-in Extensions
- **`mediainfo`**: Detailed audio/video stream inspection and codec verification.

