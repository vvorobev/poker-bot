package bot

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/bot/middleware"
)

// New creates and configures a Telegram bot instance.
// allowedChatID is the group chat ID that the bot is restricted to.
func New(token string, allowedChatID int64) (*bot.Bot, error) {
	auth := middleware.NewAuth(allowedChatID)

	opts := []bot.Option{
		bot.WithErrorsHandler(func(err error) {
			slog.Error("telegram bot error", "err", err)
		}),
		bot.WithMiddlewares(auth.Middleware),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/ping", bot.MatchTypeExact, pingHandler)

	return b, nil
}

func pingHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "pong",
	})
	if err != nil {
		slog.Error("failed to send pong", "err", err)
	}
}
