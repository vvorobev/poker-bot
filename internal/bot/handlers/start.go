package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/domain"
	"poker-bot/internal/fsm"
	"poker-bot/internal/service"
)

// ManualPhoneButtonText is the text sent when the user presses "Ввести номер вручную".
const ManualPhoneButtonText = "✏️ Ввести номер вручную"

// StartHandler handles the /start command and the onboarding reply keyboard.
type StartHandler struct {
	players  *service.PlayerService
	fsmStore *fsm.Store
}

// NewStartHandler creates a StartHandler.
func NewStartHandler(players *service.PlayerService, fsmStore *fsm.Store) *StartHandler {
	return &StartHandler{players: players, fsmStore: fsmStore}
}

// Handle processes the /start command. It only acts in private chats.
func (h *StartHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	if msg.Chat.Type != models.ChatTypePrivate {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID

	player, err := h.players.GetPlayer(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		slog.Error("start: GetPlayer failed", "userID", userID, "err", err)
		return
	}

	if player != nil {
		h.sendProfile(ctx, b, chatID, player)
		return
	}

	h.sendWelcome(ctx, b, chatID)
}

// sendProfile shows the player's current profile with an "Ок" inline button.
func (h *StartHandler) sendProfile(ctx context.Context, b *bot.Bot, chatID int64, p *domain.Player) {
	text := fmt.Sprintf(
		"👤 Твой профиль:\n\n<b>Имя:</b> %s\n<b>Телефон:</b> <code>%s</code>\n<b>Банк:</b> %s",
		p.DisplayName, p.PhoneNumber, p.BankName,
	)
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Ок", CallbackData: "start:ok"}},
		},
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: kb,
	})
	if err != nil {
		slog.Error("start: sendProfile failed", "chatID", chatID, "err", err)
	}
}

// sendWelcome shows the onboarding welcome message with the contact-sharing keyboard.
func (h *StartHandler) sendWelcome(ctx context.Context, b *bot.Bot, chatID int64) {
	text := "👋 Привет! Для участия в играх нужно зарегистрироваться.\n\n" +
		"Поделись своим номером телефона или введи его вручную."

	kb := &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{
				{Text: "📱 Поделиться контактом", RequestContact: true},
				{Text: ManualPhoneButtonText},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: kb,
	})
	if err != nil {
		slog.Error("start: sendWelcome failed", "chatID", chatID, "err", err)
	}
}

// HandleManualPhone handles the "✏️ Ввести номер вручную" reply keyboard button.
// It transitions the FSM to StateAwaitingPhone and removes the reply keyboard.
func (h *StartHandler) HandleManualPhone(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	if msg.Chat.Type != models.ChatTypePrivate {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID

	sess, ok := h.fsmStore.Get(userID)
	if !ok {
		sess = &fsm.Session{State: fsm.StateIdle, Data: make(map[string]any)}
	}
	sess.State = fsm.StateAwaitingPhone
	h.fsmStore.Set(userID, sess)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Введи номер телефона в формате <code>+7XXXXXXXXXX</code>:",
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.ReplyKeyboardRemove{RemoveKeyboard: true},
	})
	if err != nil {
		slog.Error("start: HandleManualPhone send failed", "chatID", chatID, "err", err)
	}
}

// HandleStartOK answers the "Ок" callback query silently.
func HandleStartOK(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	if err != nil {
		slog.Error("start: answerCallbackQuery failed", "err", err)
	}
}
