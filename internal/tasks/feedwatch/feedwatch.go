package feedwatch

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/en9inerd/go-pkgs/httpclient"
	"github.com/en9inerd/rig/internal/config"
	"github.com/en9inerd/rig/internal/notify"
	"github.com/en9inerd/rig/internal/storage"
)

const bucket = "feedwatch"

type Task struct {
	notifier notify.Notifier
	logger   *slog.Logger
	cfg      config.FeedConfig
	client   *httpclient.Client
	store    *storage.Store
}

func New(notifier notify.Notifier, logger *slog.Logger, cfg config.FeedConfig, store *storage.Store) *Task {
	return &Task{
		notifier: notifier,
		logger:   logger.With("task", "feedwatch"),
		cfg:      cfg,
		client: httpclient.NewWithConfig(httpclient.Config{
			Timeout: 30 * time.Second,
		}),
		store: store,
	}
}

func (t *Task) Name() string { return "feedwatch" }

func (t *Task) Start(ctx context.Context) error {
	if t.store.Len(bucket) == 0 {
		// First run: seed the store with all current entries so we
		// only notify on NEW posts — matching Node-RED's ignorefirst behavior.
		if err := t.seedPublished(ctx); err != nil {
			t.logger.Error("failed to seed published set", "error", err)
		}
	} else {
		if err := t.check(ctx); err != nil {
			t.logger.Error("initial check failed", "error", err)
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

func (t *Task) seedPublished(ctx context.Context) error {
	feed, err := t.fetchFeed(ctx)
	if err != nil {
		return err
	}

	pairs := make(map[string]string, len(feed.Entries))
	for _, entry := range feed.Entries {
		if link := entry.URL(); link != "" {
			pairs[link] = "seed"
		}
	}

	if err := t.store.SetBatch(bucket, pairs); err != nil {
		return err
	}

	t.logger.Info("seeded published set on first run", "count", len(pairs))
	return nil
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	Link    []atomLink `xml:"link"`
	Content string     `xml:"content"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func (e *atomEntry) URL() string {
	for _, l := range e.Link {
		if l.Rel == "alternate" || l.Rel == "" {
			return l.Href
		}
	}
	if len(e.Link) > 0 {
		return e.Link[0].Href
	}
	return ""
}

func extractParagraphs(s string) []string {
	var paragraphs []string
	lower := strings.ToLower(s)

	for {
		i := strings.Index(lower, "<p")
		if i == -1 {
			break
		}

		// Must be <p> or <p ...>, not <pre>, <param>, <picture>, etc.
		if i+2 >= len(lower) {
			break
		}
		if c := lower[i+2]; c != '>' && c != ' ' && c != '\t' && c != '\n' {
			lower = lower[i+2:]
			s = s[i+2:]
			continue
		}

		gt := strings.Index(lower[i:], ">")
		if gt == -1 {
			break
		}
		start := i + gt + 1

		end := strings.Index(lower[start:], "</p>")
		if end == -1 {
			break
		}

		if text := strings.TrimSpace(s[start : start+end]); text != "" {
			paragraphs = append(paragraphs, strings.ReplaceAll(text, "&apos;", "'"))
		}

		lower = lower[start+end+4:]
		s = s[start+end+4:]
	}

	return paragraphs
}

func (t *Task) check(ctx context.Context) error {
	feed, err := t.fetchFeed(ctx)
	if err != nil {
		return err
	}

	for _, entry := range feed.Entries {
		link := entry.URL()
		if link == "" {
			continue
		}

		if t.store.Has(bucket, link) {
			continue
		}

		paragraphs := extractParagraphs(entry.Content)
		summary := strings.Join(paragraphs, "\n\n")
		content := fmt.Sprintf("<b>%s</b>\n\n%s\n\n<b><a href=\"%s\">READ MORE</a></b>", html.EscapeString(entry.Title), summary, html.EscapeString(link))

		if err := t.notifier.Send(ctx, notify.Message{
			ChatID:  t.cfg.ChatID,
			Content: content,
			Options: notify.MessageOptions{
				ParseMode:             "HTML",
				DisableWebPagePreview: true,
			},
		}); err != nil {
			t.logger.Error("failed to send entry", "title", entry.Title, "error", err)
			continue
		}

		if err := t.store.Set(bucket, link, entry.Title); err != nil {
			t.logger.Error("failed to persist entry", "title", entry.Title, "error", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return nil
}

func (t *Task) fetchFeed(ctx context.Context) (*atomFeed, error) {
	resp, err := t.client.Get(ctx, t.cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read feed body: %w", err)
	}

	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse feed XML: %w", err)
	}

	return &feed, nil
}
