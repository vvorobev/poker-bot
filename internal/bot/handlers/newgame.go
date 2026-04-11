package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/bot/keyboards"
	"poker-bot/internal/domain"
	"poker-bot/internal/fsm"
	"poker-bot/internal/service"
)

// NewGameHandler handles the /newgame command and buy-in input flow.
type NewGameHandler struct {
	players       *service.PlayerService
	games         *service.GameService
	fsmStore      *fsm.Store
	allowedChatID int64
}

// NewNewGameHandler creates a NewGameHandler.
func NewNewGameHandler(players *service.PlayerService, games *service.GameService, fsmStore *fsm.Store, allowedChatID int64) *NewGameHandler {
	return &NewGameHandler{
		players:       players,
		games:         games,
		fsmStore:      fsmStore,
		allowedChatID: allowedChatID,
	}
}

// Handle processes the /newgame command.
func (h *NewGameHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	userID := msg.From.ID
	chatID := msg.Chat.ID

	// Determine which chat the game will be created in.
	// Group messages → use that chat; private messages → use the allowed group chat.
	gameChatID := chatID
	if msg.Chat.Type == models.ChatTypePrivate {
		gameChatID = h.allowedChatID
	}

	registered := h.players.IsRegistered(ctx, userID)
	if !registered {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Сначала зарегистрируйся через /start",
		})
		return
	}

	// Store target chat in FSM so the buy-in callback knows where to create the game.
	sess, ok := h.fsmStore.Get(userID)
	if !ok {
		sess = &fsm.Session{State: fsm.StateIdle, Data: make(map[string]any)}
	}
	sess.State = fsm.StateAwaitingBuyIn
	sess.Data["game_chat_id"] = gameChatID
	h.fsmStore.Set(userID, sess)

	_, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Укажи размер бай-ина для новой игры:",
		ReplyMarkup: keyboards.BuyInKeyboard(),
	})
	if sendErr != nil {
		slog.Error("newgame: send ask buy-in failed", "chatID", chatID, "err", sendErr)
	}
}

// HandleBuyInCallback processes the "buyin:XXXX" callback query.
func (h *NewGameHandler) HandleBuyInCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	cb := update.CallbackQuery
	userID := cb.From.ID

	amountStr := strings.TrimPrefix(cb.Data, "buyin:")
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			Text:            "Некорректный бай-ин",
		})
		return
	}

	replyTo := cb.Message.Message.Chat.ID

	h.createGame(ctx, b, userID, replyTo, amount, cb.ID)
}

// HandleBuyInText handles text input when FSM is in StateAwaitingBuyIn.
func (h *NewGameHandler) HandleBuyInText(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	userID := msg.From.ID
	chatID := msg.Chat.ID

	amount, err := strconv.ParseInt(strings.TrimSpace(msg.Text), 10, 64)
	if err != nil || amount <= 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        "Введи сумму бай-ина цифрами (например: 1000):",
			ReplyMarkup: keyboards.BuyInKeyboard(),
		})
		return
	}

	h.createGame(ctx, b, userID, chatID, amount, "")
}

// createGame calls GameService.NewGame and replies with the result.
// callbackQueryID is non-empty when called from a callback, used to answer the query.
func (h *NewGameHandler) createGame(ctx context.Context, b *bot.Bot, userID, chatID, amount int64, callbackQueryID string) {
	sess, ok := h.fsmStore.Get(userID)
	gameChatID := h.allowedChatID
	if ok {
		if v, ok2 := sess.Data["game_chat_id"].(int64); ok2 {
			gameChatID = v
		}
	}

	if callbackQueryID != "" {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callbackQueryID,
		})
	}

	game, err := h.games.NewGame(ctx, gameChatID, userID, amount)
	if err != nil {
		if errors.Is(err, domain.ErrGameAlreadyActive) {
			// Retrieve the active game to show its ID.
			activeGame, aerr := h.games.GetActiveGame(ctx, gameChatID)
			var text string
			if aerr == nil {
				text = fmt.Sprintf("В чате уже идёт игра #%d. Заверши её перед созданием новой.", activeGame.ID)
			} else {
				text = "В чате уже идёт игра. Заверши её перед созданием новой."
			}
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   text,
			})
			h.fsmStore.Clear(userID)
			return
		}
		slog.Error("newgame: NewGame failed", "err", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("Не удалось создать игру: %v", err),
		})
		return
	}

	h.fsmStore.Clear(userID)

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   fmt.Sprintf("✅ Игра #%d создана с бай-ином <b>%d ₽</b>.", game.ID, game.BuyIn),
		ParseMode: models.ParseModeHTML,
	})
}
