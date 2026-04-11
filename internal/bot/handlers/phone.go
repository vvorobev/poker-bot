package handlers

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/bot/keyboards"
	"poker-bot/internal/fsm"
	"poker-bot/internal/service"
)

// PhoneHandler handles phone number collection during onboarding.
type PhoneHandler struct {
	players  *service.PlayerService
	fsmStore *fsm.Store
}

// NewPhoneHandler creates a PhoneHandler.
func NewPhoneHandler(players *service.PlayerService, fsmStore *fsm.Store) *PhoneHandler {
	return &PhoneHandler{players: players, fsmStore: fsmStore}
}

// HandleContact handles contact sharing (the "📱 Поделиться контактом" button).
// Telegram sends the phone number in message.Contact.
func (h *PhoneHandler) HandleContact(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Contact == nil {
		return
	}
	msg := update.Message
	if msg.Chat.Type != models.ChatTypePrivate {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID

	phone := normalizePhone(msg.Contact.PhoneNumber)
	if !service.ValidatePhone(phone) {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      "Не удалось распознать номер телефона. Введи его вручную в формате <code>+7XXXXXXXXXX</code>:",
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			slog.Error("phone: HandleContact send failed", "chatID", chatID, "err", err)
		}
		return
	}

	h.savePhoneAndAskBank(ctx, b, userID, chatID, phone)
}

// HandlePhoneText handles manual phone number input when FSM is in StateAwaitingPhone.
func (h *PhoneHandler) HandlePhoneText(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}
	msg := update.Message
	if msg.Chat.Type != models.ChatTypePrivate {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID

	sess, ok := h.fsmStore.Get(userID)
	if !ok || sess.State != fsm.StateAwaitingPhone {
		return
	}

	phone := strings.TrimSpace(msg.Text)
	if !service.ValidatePhone(phone) {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      "Неверный формат. Введи номер в формате <code>+7XXXXXXXXXX</code>:",
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			slog.Error("phone: HandlePhoneText send failed", "chatID", chatID, "err", err)
		}
		return
	}

	h.savePhoneAndAskBank(ctx, b, userID, chatID, phone)
}

// savePhoneAndAskBank stores the phone in FSM and transitions to StateAwaitingBank.
func (h *PhoneHandler) savePhoneAndAskBank(ctx context.Context, b *bot.Bot, userID, chatID int64, phone string) {
	sess, ok := h.fsmStore.Get(userID)
	if !ok {
		sess = &fsm.Session{State: fsm.StateIdle, Data: make(map[string]any)}
	}
	sess.State = fsm.StateAwaitingBank
	sess.Data["phone"] = phone
	h.fsmStore.Set(userID, sess)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Телефон принят! Теперь выбери свой банк для переводов:",
		ReplyMarkup: keyboards.BankKeyboard(),
	})
	if err != nil {
		slog.Error("phone: savePhoneAndAskBank send failed", "chatID", chatID, "err", err)
	}
}

// normalizePhone normalizes a Telegram contact phone to +7XXXXXXXXXX format.
// Telegram may provide the number without a leading '+'.
func normalizePhone(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "+") {
		raw = "+" + raw
	}
	return raw
}
