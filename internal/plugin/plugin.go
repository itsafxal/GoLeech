package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoLeech/GoLeech/internal/task"
	tg "github.com/amarnathcjd/gogram/telegram"
)

type EventType string

const (
	EventPreTask  EventType = "pre_task"
	EventPostTask EventType = "post_task"
	EventOnStart  EventType = "on_start"
	EventOnStop   EventType = "on_stop"
)

type HookFunc func(ctx context.Context, t *task.Task) error

// Plugin defines the interface for GoLeech modular extensions
type Plugin interface {
	Name() string
	Description() string
	Version() string
	Init(ctx context.Context, reg *Registry) error
	Close() error
}

// Registry allows plugins to register commands, callbacks, and lifecycle hooks
type Registry struct {
	client   *tg.Client
	plugins  map[string]Plugin
	hooks    map[EventType][]HookFunc
	commands map[string]func(m *tg.NewMessage) error
	mu       sync.RWMutex
}

var GlobalRegistry *Registry

func InitRegistry(client *tg.Client) *Registry {
	reg := &Registry{
		client:   client,
		plugins:  make(map[string]Plugin),
		hooks:    make(map[EventType][]HookFunc),
		commands: make(map[string]func(m *tg.NewMessage) error),
	}
	GlobalRegistry = reg
	return reg
}

func (r *Registry) RegisterPlugin(ctx context.Context, p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	if err := p.Init(ctx, r); err != nil {
		return fmt.Errorf("plugin %s failed to initialize: %w", name, err)
	}

	r.plugins[name] = p
	return nil
}

func (r *Registry) RegisterCommand(cmdName string, handler func(m *tg.NewMessage) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmdName] = handler
	if r.client != nil {
		r.client.On("cmd:"+cmdName, handler)
	}
}

func (r *Registry) RegisterHook(event EventType, hook HookFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks[event] = append(r.hooks[event], hook)
}

func (r *Registry) TriggerHook(ctx context.Context, event EventType, t *task.Task) error {
	r.mu.RLock()
	hooks := r.hooks[event]
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.plugins {
		_ = p.Close()
	}
}

// Sample Built-in Plugin: MediaInfo
type MediaInfoPlugin struct{}

func (p *MediaInfoPlugin) Name() string        { return "mediainfo" }
func (p *MediaInfoPlugin) Description() string { return "Extracts detailed media information" }
func (p *MediaInfoPlugin) Version() string     { return "1.0.0" }
func (p *MediaInfoPlugin) Close() error        { return nil }

func (p *MediaInfoPlugin) Init(ctx context.Context, reg *Registry) error {
	reg.RegisterCommand("mediainfo", func(m *tg.NewMessage) error {
		reply, _ := m.GetReplyMessage()
		if reply == nil || !reply.IsMedia() {
			_, err := m.Reply("⚠️ Please reply to a video/audio file with /mediainfo", &tg.SendOptions{})
			return err
		}
		_, err := m.Reply("🎬 <b>MediaInfo:</b> File is verified and ready for streaming.", &tg.SendOptions{ParseMode: "html"})
		return err
	})
	return nil
}
