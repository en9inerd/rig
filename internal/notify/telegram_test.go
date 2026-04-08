package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/en9inerd/go-pkgs/httpclient"
)

func TestTelegram_Send_Success(t *testing.T) {
	var received telegramRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sendMessage" {
			t.Errorf("path = %q, want /sendMessage", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := &Telegram{
		client: httpclient.NewWithConfig(httpclient.Config{
			BaseURL: srv.URL,
		}),
	}

	err := tg.Send(context.Background(), Message{
		ChatID:  "123",
		Content: "hello",
		Options: MessageOptions{
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if received.ChatID != "123" {
		t.Errorf("ChatID = %q", received.ChatID)
	}
	if received.Text != "hello" {
		t.Errorf("Text = %q", received.Text)
	}
	if received.ParseMode != "HTML" {
		t.Errorf("ParseMode = %q", received.ParseMode)
	}
	if !received.DisableWebPagePreview {
		t.Error("DisableWebPagePreview = false")
	}
}

func TestTelegram_Send_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	}))
	defer srv.Close()

	tg := &Telegram{
		client: httpclient.NewWithConfig(httpclient.Config{
			BaseURL: srv.URL,
		}),
	}

	err := tg.Send(context.Background(), Message{
		ChatID:  "bad",
		Content: "test",
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestTelegram_Send_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := &Telegram{
		client: httpclient.NewWithConfig(httpclient.Config{
			BaseURL: srv.URL,
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := tg.Send(ctx, Message{ChatID: "123", Content: "test"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestTelegram_Send_RetriesOn429(t *testing.T) {
	attempts := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"description":"Too Many Requests: retry after 1","parameters":{"retry_after":1}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := &Telegram{
		client: httpclient.NewWithConfig(httpclient.Config{
			BaseURL: srv.URL,
		}),
	}

	err := tg.Send(context.Background(), Message{ChatID: "123", Content: "test"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestTelegram_Send_429RespectsContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"ok":false,"parameters":{"retry_after":60}}`))
	}))
	defer srv.Close()

	tg := &Telegram{
		client: httpclient.NewWithConfig(httpclient.Config{
			BaseURL: srv.URL,
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Cancel quickly so we don't wait 60 seconds.
		cancel()
	}()

	err := tg.Send(ctx, Message{ChatID: "123", Content: "test"})
	if err == nil {
		t.Fatal("expected error for cancelled context during retry wait")
	}
}

func TestTelegram_Send_OmitsEmptyParseMode(t *testing.T) {
	var rawBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&rawBody); err != nil {
			t.Fatalf("decode: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := &Telegram{
		client: httpclient.NewWithConfig(httpclient.Config{
			BaseURL: srv.URL,
		}),
	}

	_ = tg.Send(context.Background(), Message{
		ChatID:  "123",
		Content: "plain text",
	})

	if _, exists := rawBody["parse_mode"]; exists {
		t.Error("parse_mode should be omitted when empty")
	}
}
