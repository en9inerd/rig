package notify

import (
	"context"
	"fmt"
	"io"

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

func (t *Telegram) Send(ctx context.Context, msg Message) error {
	body := telegramRequest{
		ChatID:                msg.ChatID,
		Text:                  msg.Content,
		ParseMode:             msg.Options.ParseMode,
		DisableWebPagePreview: msg.Options.DisableWebPagePreview,
	}

	resp, err := t.client.Post(ctx, "/sendMessage", body)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
