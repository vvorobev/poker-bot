package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/fsm"
	"poker-bot/internal/service"
)

// BankHandler handles bank selection during onboarding.
type BankHandler struct {
	players  *service.PlayerService
	fsmStore *fsm.Store
}

// NewBankHandler creates a BankHandler.
func NewBankHandler(players *service.PlayerService, fsmStore *fsm.Store) *BankHandler {
	return &BankHandler{players: players, fsmStore: fsmStore}
}

// HandleBankCallback handles the "bank:<name>" callback query from BankKeyboard.
// If the user selected "Другой", FSM asks for a custom bank name.
// Otherwise, it finalises registration.
func (h *BankHandler) HandleBankCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	cb := update.CallbackQuery
	userID := cb.From.ID
	chatID := cb.Message.Message.Chat.ID

	// Acknowledge the callback immediately.
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
	})

	bankName := strings.TrimPrefix(cb.Data, "bank:")

	sess, ok := h.fsmStore.Get(userID)
	if !ok || sess.State != fsm.StateAwaitingBank {
		return
	}

	if bankName == "Другой" {
		// Ask for a custom bank name.
		sess.Data["bank_custom"] = true
		h.fsmStore.Set(userID, sess)
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Введи название своего банка:",
		})
		if err != nil {
			slog.Error("bank: send custom bank prompt failed", "chatID", chatID, "err", err)
		}
		return
	}

	h.finishRegistration(ctx, b, userID, chatID, bankName, &cb.From)
}

// HandleBankText handles free-text bank name input when FSM is in StateAwaitingBank
// and the user previously selected "Другой".
func (h *BankHandler) HandleBankText(ctx context.Context, b *bot.Bot, update *models.Update) {
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
	if !ok || sess.State != fsm.StateAwaitingBank {
		return
	}
	customFlag, _ := sess.Data["bank_custom"].(bool)
	if !customFlag {
		return
	}

	bankName := strings.TrimSpace(msg.Text)
	if bankName == "" {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Название банка не может быть пустым. Введи название своего банка:",
		})
		if err != nil {
			slog.Error("bank: send empty bank error failed", "chatID", chatID, "err", err)
		}
		return
	}

	h.finishRegistration(ctx, b, userID, chatID, bankName, msg.From)
}

// Custom bank name text input (FSM state=AwaitingBank, bank_custom=true).
func (h *BankHandler) MatchBankTextCommand(update *models.Update) bool {
	if update.Message == nil || update.Message.Text == "" {
		return false
	}
	if update.Message.Chat.Type != models.ChatTypePrivate {
		return false
	}
	sess, ok := h.fsmStore.Get(update.Message.From.ID)
	if !ok || sess.State != fsm.StateAwaitingBank {
		return false
	}
	customFlag, _ := sess.Data["bank_custom"].(bool)
	return customFlag
}

// finishRegistration saves the player profile and sends a confirmation message.
func (h *BankHandler) finishRegistration(
	ctx context.Context,
	b *bot.Bot,
	userID, chatID int64,
	bankName string,
	from *models.User,
) {
	sess, ok := h.fsmStore.Get(userID)
	if !ok {
		return
	}
	phone, _ := sess.Data["phone"].(string)

	username := from.Username
	displayName := strings.TrimSpace(from.FirstName + " " + from.LastName)
	if displayName == "" {
		displayName = from.FirstName
	}

	if err := h.players.RegisterPlayer(ctx, userID, username, displayName, phone, bankName); err != nil {
		slog.Error("bank: RegisterPlayer failed", "userID", userID, "err", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Ошибка при сохранении профиля. Попробуй ещё раз.",
		})
		return
	}

	h.fsmStore.Clear(userID)

	text := fmt.Sprintf(
		"✅ Профиль сохранён!\n\n<b>Имя:</b> %s\n<b>Телефон:</b> <code>%s</code>\n<b>Банк:</b> %s",
		displayName, phone, bankName,
	)
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &models.ReplyKeyboardRemove{RemoveKeyboard: true},
	})
	if err != nil {
		slog.Error("bank: send confirmation failed", "chatID", chatID, "err", err)
	}
}
