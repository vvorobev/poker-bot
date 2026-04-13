package handlers

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/fsm"
)

// CancelHandler resets the user's FSM state on /cancel.
type CancelHandler struct {
	fsmStore *fsm.Store
}

// NewCancelHandler creates a CancelHandler.
func NewCancelHandler(fsmStore *fsm.Store) *CancelHandler {
	return &CancelHandler{fsmStore: fsmStore}
}

// Handle processes the /cancel command. Resets FSM to StateIdle.
func (h *CancelHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	userID := update.Message.From.ID

	sess, ok := h.fsmStore.Get(userID)
	if !ok || sess.State == fsm.StateIdle {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Нечего отменять",
		})
		if err != nil {
			slog.Warn("cancel: SendMessage failed", "userID", userID, "err", err)
		}
		return
	}

	h.fsmStore.Set(userID, &fsm.Session{
		State: fsm.StateIdle,
		Data:  make(map[string]any),
	})

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Отменено",
	})
	if err != nil {
		slog.Warn("cancel: SendMessage failed", "userID", userID, "err", err)
	}
}
