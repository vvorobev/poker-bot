package handlers

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/fsm"
)

const helpText = `📖 <b>Доступные команды:</b>

/start — регистрация / профиль
/newgame — создать новую игру
/game — текущая игра
/name — изменить отображаемое имя
/edit — изменить введённые данные
/cancel — отменить текущее действие
/number — Изменить номер телефона и банк
/help — список команд`

// HelpHandler handles /help command.
type HelpHandler struct{}

func NewHelpHandler() *HelpHandler { return &HelpHandler{} }

func (h *HelpHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      helpText,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		slog.Error("help: send failed", "err", err)
	}
}

// FallbackHandler handles unknown commands and random text.
type FallbackHandler struct {
	fsmStore *fsm.Store
}

func NewFallbackHandler(fsmStore *fsm.Store) *FallbackHandler {
	return &FallbackHandler{fsmStore: fsmStore}
}

// HandleUnknownCommand responds to unknown /commands in private chat.
func (h *FallbackHandler) HandleUnknownCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Неизвестная команда. Используй /help для списка команд.",
	})
	if err != nil {
		slog.Error("fallback: unknown command send failed", "err", err)
	}
}

// MatchUnknownCommand matches any unhandled /command in a private chat.
func MatchUnknownCommand(update *models.Update) bool {
	if update.Message == nil || update.Message.Text == "" {
		return false
	}
	if update.Message.Chat.Type != models.ChatTypePrivate {
		return false
	}
	return strings.HasPrefix(update.Message.Text, "/")
}

// HandlePlainText responds to plain text in private chat when no FSM state is active.
// In groups, ignores the message.
func (h *FallbackHandler) HandlePlainText(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Используй команды для управления. /help",
	})
	if err != nil {
		slog.Error("fallback: plain text send failed", "err", err)
	}
}

// MatchPlainTextFallback matches plain (non-command) text in a private chat
// when no FSM dialog is active for the user.
func (h *FallbackHandler) MatchPlainTextFallback(fsmStore *fsm.Store) func(*models.Update) bool {
	return func(update *models.Update) bool {
		if update.Message == nil || update.Message.Text == "" {
			return false
		}
		msg := update.Message
		if msg.Chat.Type != models.ChatTypePrivate {
			return false
		}
		if strings.HasPrefix(msg.Text, "/") {
			return false
		}
		// Only fire when no FSM state is active.
		sess, ok := fsmStore.Get(msg.From.ID)
		if ok && sess.State != fsm.StateIdle {
			return false
		}
		return true
	}
}
