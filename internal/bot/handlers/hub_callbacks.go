package handlers

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/bot/keyboards"
	"poker-bot/internal/bot/views"
	"poker-bot/internal/domain"
	"poker-bot/internal/service"
)

// HubCallbackHandler handles join/rebuy/cancel_rebuy callback queries from the game hub.
type HubCallbackHandler struct {
	players *service.PlayerService
	games   *service.GameService
}

// NewHubCallbackHandler creates a HubCallbackHandler.
func NewHubCallbackHandler(players *service.PlayerService, games *service.GameService) *HubCallbackHandler {
	return &HubCallbackHandler{players: players, games: games}
}

// HandleJoin processes the "join:N" callback query.
func (h *HubCallbackHandler) HandleJoin(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}
	userID := cb.From.ID
	gameID := parseGameIDFromCallback(cb.Data)

	if !h.players.IsRegistered(ctx, userID) {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			Text:            "Сначала нажми /start в личном чате",
			ShowAlert:       true,
		})
		return
	}

	game, participants, err := h.games.Join(ctx, gameID, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAlreadyJoined):
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Ты уже в игре",
				ShowAlert:       true,
			})
		case errors.Is(err, domain.ErrGameNotActive):
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Эта игра уже завершена",
				ShowAlert:       true,
			})
		default:
			slog.Error("hub: Join failed", "gameID", gameID, "userID", userID, "err", err)
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Ошибка. Попробуй ещё раз",
				ShowAlert:       true,
			})
		}
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "Ты присоединился!",
	})
	h.updateHub(ctx, b, game, participants)
}

// HandleRebuy processes the "rebuy:N" callback query.
func (h *HubCallbackHandler) HandleRebuy(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}
	userID := cb.From.ID
	gameID := parseGameIDFromCallback(cb.Data)

	game, participants, err := h.games.Rebuy(ctx, gameID, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotParticipant):
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Ты не участник этой игры",
				ShowAlert:       true,
			})
		case errors.Is(err, domain.ErrGameNotActive):
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Эта игра уже завершена",
				ShowAlert:       true,
			})
		default:
			slog.Error("hub: Rebuy failed", "gameID", gameID, "userID", userID, "err", err)
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Ошибка. Попробуй ещё раз",
				ShowAlert:       true,
			})
		}
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "Докуп записан!",
	})
	h.updateHub(ctx, b, game, participants)
}

// HandleCancelRebuy processes the "cancel_rebuy:N" callback query.
func (h *HubCallbackHandler) HandleCancelRebuy(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}
	userID := cb.From.ID
	gameID := parseGameIDFromCallback(cb.Data)

	game, participants, err := h.games.CancelRebuy(ctx, gameID, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotParticipant):
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Ты не участник этой игры",
				ShowAlert:       true,
			})
		case errors.Is(err, domain.ErrGameNotActive):
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Эта игра уже завершена",
				ShowAlert:       true,
			})
		default:
			slog.Error("hub: CancelRebuy failed", "gameID", gameID, "userID", userID, "err", err)
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Ошибка. Попробуй ещё раз",
				ShowAlert:       true,
			})
		}
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "Докуп отменён",
	})
	h.updateHub(ctx, b, game, participants)
}

// updateHub edits the hub message in the group chat to reflect the current game state.
func (h *HubCallbackHandler) updateHub(ctx context.Context, b *bot.Bot, game *domain.Game, participants []domain.Participant) {
	if game.HubMessageID == 0 {
		return
	}

	// Build players map for display name resolution.
	playerIDs := make(map[int64]struct{}, len(participants)+1)
	playerIDs[game.CreatorID] = struct{}{}
	for _, p := range participants {
		playerIDs[p.PlayerID] = struct{}{}
	}
	playerMap := make(map[int64]*domain.Player, len(playerIDs))
	for id := range playerIDs {
		if p, err := h.players.GetPlayer(ctx, id); err == nil {
			playerMap[id] = p
		}
	}

	hubText := views.RenderHub(game, participants, playerMap)

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      game.ChatID,
		MessageID:   int(game.HubMessageID),
		Text:        hubText,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboards.HubKeyboard(game.ID),
	})
	if err != nil {
		slog.Error("hub: EditMessageText failed", "gameID", game.ID, "err", err)
	}
}

// parseGameIDFromCallback extracts the numeric game ID from callback data like "join:42".
func parseGameIDFromCallback(data string) int64 {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return 0
	}
	id, _ := strconv.ParseInt(parts[1], 10, 64)
	return id
}
