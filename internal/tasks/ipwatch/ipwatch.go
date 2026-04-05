package ipwatch

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/en9inerd/go-pkgs/httpclient"
	"github.com/en9inerd/rig/internal/notify"
	"github.com/en9inerd/rig/internal/storage"
)

const (
	bucket  = "ipwatch"
	ipv4Key = "current_ipv4"
	ipv6Key = "current_ipv6"
)

type Task struct {
	notifier notify.Notifier
	logger   *slog.Logger
	cfg      Config
	client   *httpclient.Client
	store    *storage.Store
	lastIPv4 string
	lastIPv6 string
}

func New(notifier notify.Notifier, logger *slog.Logger, cfg Config, store *storage.Store) *Task {
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
	t.lastIPv4, _ = t.store.Get(bucket, ipv4Key)
	t.lastIPv6, _ = t.store.Get(bucket, ipv6Key)

	// If no persisted state, do an initial fetch to seed the known IPs
	// without notifying — matching the Node-RED behavior where the inject
	// node has once=false (first check after the interval, not on deploy).
	if t.lastIPv4 == "" {
		if ip, err := t.fetchIPv4(ctx); err == nil {
			t.lastIPv4 = ip
			if err := t.store.Set(bucket, ipv4Key, ip); err != nil {
				t.logger.Error("failed to save initial IPv4 state", "error", err)
			}
			t.logger.Info("seeded initial IPv4", "ip", ip)
		} else {
			t.logger.Error("failed to fetch initial IPv4", "error", err)
		}
	}
	if t.lastIPv6 == "" {
		if ip, err := t.fetchIPv6(ctx); err == nil {
			t.lastIPv6 = ip
			if err := t.store.Set(bucket, ipv6Key, ip); err != nil {
				t.logger.Error("failed to save initial IPv6 state", "error", err)
			}
			t.logger.Info("seeded initial IPv6", "ip", ip)
		} else {
			t.logger.Warn("IPv6 not available", "error", err)
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
	var changed []string

	if ipv4, err := t.fetchIPv4(ctx); err != nil {
		t.logger.Error("IPv4 check failed", "error", err)
	} else if ipv4 != t.lastIPv4 {
		t.logger.Info("IPv4 changed", "old", t.lastIPv4, "new", ipv4)
		t.lastIPv4 = ipv4
		if err := t.store.Set(bucket, ipv4Key, ipv4); err != nil {
			t.logger.Error("failed to save IPv4 state", "error", err)
		}
		changed = append(changed, fmt.Sprintf("IPv4: %s", ipv4))
	}

	if ipv6, err := t.fetchIPv6(ctx); err != nil {
		t.logger.Warn("IPv6 check failed", "error", err)
	} else if ipv6 != t.lastIPv6 {
		t.logger.Info("IPv6 changed", "old", t.lastIPv6, "new", ipv6)
		t.lastIPv6 = ipv6
		if err := t.store.Set(bucket, ipv6Key, ipv6); err != nil {
			t.logger.Error("failed to save IPv6 state", "error", err)
		}
		changed = append(changed, fmt.Sprintf("IPv6: %s", ipv6))
	}

	if len(changed) == 0 {
		return nil
	}

	content := "New Public IP\n" + strings.Join(changed, "\n")
	return t.notifier.Send(ctx, notify.Message{
		ChatID:  t.cfg.ChatID,
		Content: content,
		Options: notify.MessageOptions{
			DisableWebPagePreview: true,
		},
	})
}

type ipifyResponse struct {
	IP string `json:"ip"`
}

func (t *Task) fetchIPv4(ctx context.Context) (string, error) {
	var result ipifyResponse
	if err := t.client.GetJSON(ctx, "https://api.ipify.org?format=json", &result); err != nil {
		return "", fmt.Errorf("fetch IPv4: %w", err)
	}
	return result.IP, nil
}

func (t *Task) fetchIPv6(ctx context.Context) (string, error) {
	var result ipifyResponse
	if err := t.client.GetJSON(ctx, "https://api6.ipify.org?format=json", &result); err != nil {
		return "", fmt.Errorf("fetch IPv6: %w", err)
	}
	return result.IP, nil
}
