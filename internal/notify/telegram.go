package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/en9inerd/go-pkgs/httpclient"
)

type Telegram struct {
	client *httpclient.Client
}

func NewTelegram(token string) *Telegram {
	return &Telegram{
		client: httpclient.NewWithConfig(httpclient.Config{
			BaseURL: "https://api.telegram.org/bot" + token,
		}),
	}
}

type telegramRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
}

type telegramError struct {
	Parameters struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

func (t *Telegram) Send(ctx context.Context, msg Message) error {
	body := telegramRequest{
		ChatID:                msg.ChatID,
		Text:                  msg.Content,
		ParseMode:             msg.Options.ParseMode,
		DisableWebPagePreview: msg.Options.DisableWebPagePreview,
	}

	for {
		resp, err := t.client.Post(ctx, "/sendMessage", body)
		if err != nil {
			return fmt.Errorf("send telegram message: %w", err)
		}

		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			resp.Body.Close()
			return nil
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			var tgErr telegramError
			if err := json.Unmarshal(respBody, &tgErr); err == nil && tgErr.Parameters.RetryAfter > 0 {
				delay := time.Duration(tgErr.Parameters.RetryAfter) * time.Second
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
		}

		return fmt.Errorf("telegram API status %d: %s", resp.StatusCode, string(respBody))
	}
}
