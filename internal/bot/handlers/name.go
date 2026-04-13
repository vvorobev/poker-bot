package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/domain"
	"poker-bot/internal/service"
)

const maxDisplayNameLen = 50

// NameHandler handles the /name command.
type NameHandler struct {
	players *service.PlayerService
}

// NewNameHandler creates a NameHandler.
func NewNameHandler(players *service.PlayerService) *NameHandler {
	return &NameHandler{players: players}
}

// Handle processes the /name command. Private chats only.
func (h *NameHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	if msg.Chat.Type != models.ChatTypePrivate {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID

	// Extract argument after "/name"
	text := strings.TrimSpace(msg.Text)
	arg := strings.TrimSpace(strings.TrimPrefix(text, "/name"))

	player, err := h.players.GetPlayer(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		slog.Error("name: GetPlayer failed", "userID", userID, "err", err)
		return
	}

	if errors.Is(err, domain.ErrNotFound) || player == nil {
		h.send(ctx, b, chatID, "Сначала зарегистрируйся через /start")
		return
	}

	// No argument — show current name
	if arg == "" {
		h.send(ctx, b, chatID, fmt.Sprintf("Твоё текущее имя: <b>%s</b>\n\nЧтобы изменить: /name Новое имя", player.DisplayName), models.ParseModeHTML)
		return
	}

	if len([]rune(arg)) > maxDisplayNameLen {
		h.send(ctx, b, chatID, fmt.Sprintf("Имя слишком длинное. Максимум %d символов.", maxDisplayNameLen))
		return
	}

	if err := h.players.UpdateDisplayName(ctx, userID, arg); err != nil {
		slog.Error("name: UpdateDisplayName failed", "userID", userID, "err", err)
		h.send(ctx, b, chatID, "Ошибка обновления имени. Попробуй ещё раз.")
		return
	}

	h.send(ctx, b, chatID, fmt.Sprintf("Имя изменено на: <b>%s</b>", arg), models.ParseModeHTML)
}

func (h *NameHandler) send(ctx context.Context, b *bot.Bot, chatID int64, text string, parseMode ...models.ParseMode) {
	params := &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	}
	if len(parseMode) > 0 {
		params.ParseMode = parseMode[0]
	}
	_, err := b.SendMessage(ctx, params)
	if err != nil {
		slog.Warn("name: SendMessage failed", "chatID", chatID, "err", err)
	}
}
