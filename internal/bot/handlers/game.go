package handlers

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/bot/keyboards"
	"poker-bot/internal/bot/views"
	"poker-bot/internal/domain"
	"poker-bot/internal/service"
)

// GameHandler handles the /game command in private chat.
type GameHandler struct {
	players       *service.PlayerService
	games         *service.GameService
	allowedChatID int64
}

// NewGameCommandHandler creates a GameHandler.
func NewGameCommandHandler(players *service.PlayerService, games *service.GameService, allowedChatID int64) *GameHandler {
	return &GameHandler{players: players, games: games, allowedChatID: allowedChatID}
}

// Handle processes the /game command sent in a private chat.
func (h *GameHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	userID := msg.From.ID
	chatID := msg.Chat.ID

	if !h.players.IsRegistered(ctx, userID) {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Сначала зарегистрируйся через /start",
		})
		return
	}

	game, err := h.games.GetActiveGame(ctx, h.allowedChatID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Активных игр нет. Создай игру командой /newgame",
			})
			return
		}
		slog.Error("game: GetActiveGame failed", "err", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Ошибка при получении игры. Попробуй ещё раз",
		})
		return
	}

	participants, err := h.games.GetParticipants(ctx, game.ID)
	if err != nil {
		slog.Error("game: GetParticipants failed", "gameID", game.ID, "err", err)
		participants = nil
	}

	playerIDs := make(map[int64]struct{}, len(participants)+1)
	playerIDs[game.CreatorID] = struct{}{}
	for _, p := range participants {
		playerIDs[p.PlayerID] = struct{}{}
	}
	playerMap := make(map[int64]*domain.Player, len(playerIDs))
	for id := range playerIDs {
		if p, pErr := h.players.GetPlayer(ctx, id); pErr == nil {
			playerMap[id] = p
		}
	}

	hubText := views.RenderHub(game, participants, playerMap)
	_, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        hubText,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboards.HubKeyboard(game.ID),
	})
	if sendErr != nil {
		slog.Error("game: send hub failed", "userID", userID, "err", sendErr)
	}
}
