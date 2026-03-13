package notify

import "context"

type Message struct {
	ChatID  string
	Content string
	Options MessageOptions
}

type MessageOptions struct {
	ParseMode             string // "HTML", "Markdown", ""
	DisableWebPagePreview bool
}

type Notifier interface {
	Send(ctx context.Context, msg Message) error
}
