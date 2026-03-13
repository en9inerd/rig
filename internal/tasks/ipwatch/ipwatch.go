package ipwatch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/en9inerd/go-pkgs/httpclient"
	"github.com/en9inerd/rig/internal/config"
	"github.com/en9inerd/rig/internal/notify"
	"github.com/en9inerd/rig/internal/storage"
)

const (
	bucket = "ipwatch"
	ipKey  = "current_ip"
)

type Task struct {
	notifier notify.Notifier
	logger   *slog.Logger
	cfg      config.IPConfig
	client   *httpclient.Client
	store    *storage.Store
	lastIP   string
}

func New(notifier notify.Notifier, logger *slog.Logger, cfg config.IPConfig, store *storage.Store) *Task {
	return &Task{
		notifier: notifier,
		logger:   logger.With("task", "ipwatch"),
		cfg:      cfg,
		client: httpclient.NewWithConfig(httpclient.Config{
			Timeout: 10 * time.Second,
		}),
		store: store,
	}
}

func (t *Task) Name() string { return "ipwatch" }

func (t *Task) Start(ctx context.Context) error {
	t.lastIP, _ = t.store.Get(bucket, ipKey)

	// If no persisted state, do an initial fetch to seed the known IP
	// without notifying — matching the Node-RED behavior where the inject
	// node has once=false (first check after the interval, not on deploy).
	if t.lastIP == "" {
		if ip, err := t.fetchIP(ctx); err == nil {
			t.lastIP = ip
			if err := t.store.Set(bucket, ipKey, ip); err != nil {
				t.logger.Error("failed to save initial state", "error", err)
			}
			t.logger.Info("seeded initial IP", "ip", ip)
		} else {
			t.logger.Error("failed to fetch initial IP", "error", err)
		}
	}

	ticker := time.NewTicker(t.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := t.check(ctx); err != nil {
				t.logger.Error("check failed", "error", err)
			}
		}
	}
}

func (t *Task) check(ctx context.Context) error {
	ip, err := t.fetchIP(ctx)
	if err != nil {
		return err
	}

	if ip == t.lastIP {
		return nil
	}

	t.logger.Info("IP changed", "old", t.lastIP, "new", ip)
	t.lastIP = ip

	if err := t.store.Set(bucket, ipKey, ip); err != nil {
		t.logger.Error("failed to save state", "error", err)
	}

	return t.notifier.Send(ctx, notify.Message{
		ChatID:  t.cfg.ChatID,
		Content: fmt.Sprintf("New Public IP: %s", ip),
		Options: notify.MessageOptions{
			DisableWebPagePreview: true,
		},
	})
}

type ipifyResponse struct {
	IP string `json:"ip"`
}

func (t *Task) fetchIP(ctx context.Context) (string, error) {
	var result ipifyResponse
	if err := t.client.GetJSON(ctx, "https://api64.ipify.org?format=json", &result); err != nil {
		return "", fmt.Errorf("fetch IP: %w", err)
	}
	return result.IP, nil
}
