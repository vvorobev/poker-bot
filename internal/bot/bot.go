package bot

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/bot/handlers"
	"poker-bot/internal/bot/middleware"
	"poker-bot/internal/fsm"
	"poker-bot/internal/service"
)

// Deps holds the dependencies injected into the bot's handlers.
type Deps struct {
	AllowedChatID int64
	Players       *service.PlayerService
	FSM           *fsm.Store
}

// New creates and configures a Telegram bot instance.
func New(token string, deps Deps) (*bot.Bot, error) {
	auth := middleware.NewAuth(deps.AllowedChatID)

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

	startH := handlers.NewStartHandler(deps.Players, deps.FSM)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, startH.Handle)
	b.RegisterHandler(bot.HandlerTypeMessageText, handlers.ManualPhoneButtonText, bot.MatchTypeExact, startH.HandleManualPhone)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "start:ok", bot.MatchTypeExact, handlers.HandleStartOK)

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
