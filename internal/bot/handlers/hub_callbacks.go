package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"poker-bot/internal/bot/keyboards"
	"poker-bot/internal/bot/views"
	"poker-bot/internal/domain"
	"poker-bot/internal/fsm"
	"poker-bot/internal/service"
)

// HubCallbackHandler handles join/rebuy/cancel_rebuy/finish callback queries from the game hub.
type HubCallbackHandler struct {
	players *service.PlayerService
	games   *service.GameService
	fsmStore *fsm.Store
}

// NewHubCallbackHandler creates a HubCallbackHandler.
func NewHubCallbackHandler(players *service.PlayerService, games *service.GameService, fsmStore *fsm.Store) *HubCallbackHandler {
	return &HubCallbackHandler{players: players, games: games, fsmStore: fsmStore}
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

// HandleFinish processes the "finish:N" callback query with two-tap confirmation.
// First tap shows a confirmation alert; second tap within 30 seconds triggers FinishGame.
func (h *HubCallbackHandler) HandleFinish(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}
	userID := cb.From.ID
	gameID := parseGameIDFromCallback(cb.Data)

	now := time.Now()
	sess, hasSess := h.fsmStore.Get(userID)

	// Check for pending confirmation from a previous tap.
	if hasSess && sess.Data != nil {
		confirmGameID, _ := sess.Data["finish_confirm_game_id"].(int64)
		confirmTime, _ := sess.Data["finish_confirm_time"].(time.Time)
		if confirmGameID == gameID && now.Sub(confirmTime) <= 30*time.Second {
			// Second tap confirmed — execute finish.
			delete(sess.Data, "finish_confirm_game_id")
			delete(sess.Data, "finish_confirm_time")
			h.fsmStore.Set(userID, sess)

			game, participants, err := h.games.FinishGame(ctx, gameID, userID)
			if err != nil {
				switch {
				case errors.Is(err, domain.ErrGameNotActive):
					_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
						CallbackQueryID: cb.ID,
						Text:            "Игра уже завершается или завершена",
						ShowAlert:       true,
					})
				case errors.Is(err, domain.ErrNotParticipant):
					_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
						CallbackQueryID: cb.ID,
						Text:            "Только участники могут завершить игру",
						ShowAlert:       true,
					})
				default:
					slog.Error("hub: FinishGame failed", "gameID", gameID, "userID", userID, "err", err)
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
				Text:            "Игра завершена! Сбор результатов начат.",
			})
			h.updateHub(ctx, b, game, participants)
			h.sendCollectResultsMessages(ctx, b, game, participants)
			return
		}
	}

	// First tap (or expired confirmation) — store confirmation state and show alert.
	if !hasSess || sess.Data == nil {
		sess = &fsm.Session{
			State: fsm.StateIdle,
			Data:  make(map[string]any),
		}
	}
	sess.Data["finish_confirm_game_id"] = gameID
	sess.Data["finish_confirm_time"] = now
	h.fsmStore.Set(userID, sess)

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "Точно завершить игру? Это запустит сбор результатов у всех игроков. Нажми ещё раз для подтверждения",
		ShowAlert:       true,
	})
}

// sendCollectResultsMessages sends a personal message to each participant asking for final chip counts.
func (h *HubCallbackHandler) sendCollectResultsMessages(ctx context.Context, b *bot.Bot, game *domain.Game, participants []domain.Participant) {
	for _, p := range participants {
		text := fmt.Sprintf(
			"🏁 <b>Игра #%d завершена!</b>\n\nВведи свои финальные данные — напиши /game в этот чат.",
			game.ID,
		)
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    p.PlayerID,
			Text:      text,
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			slog.Warn("hub: failed to send collect-results message", "playerID", p.PlayerID, "gameID", game.ID, "err", err)
		}
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
