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
	"poker-bot/internal/bot/views"
	"poker-bot/internal/domain"
	"poker-bot/internal/fsm"
	"poker-bot/internal/service"
)

// CollectResultsHandler manages the personal chip collection flow during collecting_results.
type CollectResultsHandler struct {
	players  *service.PlayerService
	games    *service.GameService
	fsmStore *fsm.Store
}

// NewCollectResultsHandler creates a CollectResultsHandler.
func NewCollectResultsHandler(players *service.PlayerService, games *service.GameService, fsmStore *fsm.Store) *CollectResultsHandler {
	return &CollectResultsHandler{players: players, games: games, fsmStore: fsmStore}
}

// SendCollectionMessage sends the interactive chip collection message to a player.
func (h *CollectResultsHandler) SendCollectionMessage(ctx context.Context, b *bot.Bot, playerID int64, game *domain.Game, p *domain.Participant) {
	text := views.RenderChipsInput(game, p)
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      playerID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboards.ChipsCollectionKeyboard(game.ID),
	})
	if err != nil {
		slog.Warn("collect: SendCollectionMessage failed", "playerID", playerID, "gameID", game.ID, "err", err)
	}
}

// HandleRebuyPlus processes "collect_rebuy_plus:gameID" callback.
func (h *CollectResultsHandler) HandleRebuyPlus(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.handleRebuyAdjust(ctx, b, update, +1)
}

// HandleRebuyMinus processes "collect_rebuy_minus:gameID" callback.
func (h *CollectResultsHandler) HandleRebuyMinus(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.handleRebuyAdjust(ctx, b, update, -1)
}

func (h *CollectResultsHandler) handleRebuyAdjust(ctx context.Context, b *bot.Bot, update *models.Update, delta int) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}
	userID := cb.From.ID
	gameID := parseGameIDFromCallback(cb.Data)

	p, err := h.games.AdjustRebuyInCollection(ctx, gameID, userID, delta)
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
				Text:            "Сбор результатов уже завершён",
				ShowAlert:       true,
			})
		default:
			slog.Error("collect: AdjustRebuy failed", "gameID", gameID, "userID", userID, "err", err)
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Ошибка. Попробуй ещё раз",
				ShowAlert:       true,
			})
		}
		return
	}

	game, err := h.games.GetGameByID(ctx, gameID)
	if err != nil {
		slog.Error("collect: GetGameByID failed", "gameID", gameID, "err", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})

	text := views.RenderChipsInput(game, p)
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      cb.Message.Message.Chat.ID,
		MessageID:   cb.Message.Message.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboards.ChipsCollectionKeyboard(gameID),
	})
	if err != nil {
		slog.Error("collect: EditMessageText (rebuy) failed", "err", err)
	}
}

// HandleChipsMode processes "chips_mode:chips:gameID" or "chips_mode:rubles:gameID".
// Sets FSM to StateAwaitingChipsInput and edits message to prompt for input.
func (h *CollectResultsHandler) HandleChipsMode(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}
	userID := cb.From.ID

	// Format: "chips_mode:<mode>:<gameID>"
	parts := strings.SplitN(cb.Data, ":", 3)
	if len(parts) != 3 {
		return
	}
	mode := parts[1]
	gameID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}

	sess, ok := h.fsmStore.Get(userID)
	if !ok || sess.Data == nil {
		sess = &fsm.Session{Data: make(map[string]any)}
	}
	sess.State = fsm.StateAwaitingChipsInput
	sess.Data["collect_game_id"] = gameID
	sess.Data["collect_mode"] = mode
	sess.Data["collect_msg_id"] = int64(cb.Message.Message.ID)
	sess.Data["collect_chat_id"] = cb.Message.Message.Chat.ID
	h.fsmStore.Set(userID, sess)

	modeLabel := "фишках"
	if mode == "rubles" {
		modeLabel = "рублях"
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})

	promptText := "✏️ Введи свой остаток в " + modeLabel + " (целое число >= 0):"
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    cb.Message.Message.Chat.ID,
		MessageID: cb.Message.Message.ID,
		Text:      promptText,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		slog.Error("collect: EditMessageText (mode prompt) failed", "err", err)
	}
}

// HandleChipsText processes numeric text input when FSM is StateAwaitingChipsInput.
func (h *CollectResultsHandler) HandleChipsText(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message
	if msg == nil {
		return
	}
	userID := msg.From.ID

	sess, ok := h.fsmStore.Get(userID)
	if !ok || sess.State != fsm.StateAwaitingChipsInput {
		return
	}

	gameID, _ := sess.Data["collect_game_id"].(int64)
	msgID, _ := sess.Data["collect_msg_id"].(int64)
	chatID, _ := sess.Data["collect_chat_id"].(int64)

	chips, parseErr := strconv.ParseInt(strings.TrimSpace(msg.Text), 10, 64)
	if parseErr != nil || chips < 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "⚠️ Введи целое число >= 0",
		})
		return
	}

	game, err := h.games.GetGameByID(ctx, gameID)
	if err != nil {
		slog.Error("collect: GetGameByID after chip input", "gameID", gameID, "err", err)
		return
	}
	p, err := h.games.GetParticipant(ctx, gameID, userID)
	if err != nil {
		slog.Error("collect: GetParticipant after chip input", "gameID", gameID, "userID", userID, "err", err)
		return
	}

	// Store chips value for TASK-031 confirm handler.
	sess.State = fsm.StateIdle
	sess.Data["confirm_chips"] = chips
	sess.Data["confirm_game_id"] = gameID
	h.fsmStore.Set(userID, sess)

	confirmText := views.RenderChipsConfirm(game, p, chips)
	if msgID != 0 && chatID != 0 {
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   int(msgID),
			Text:        confirmText,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboards.ResultConfirmKeyboard(gameID),
		})
		if err != nil {
			slog.Warn("collect: EditMessageText (confirm) failed, sending new", "err", err)
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      msg.Chat.ID,
				Text:        confirmText,
				ParseMode:   models.ParseModeHTML,
				ReplyMarkup: keyboards.ResultConfirmKeyboard(gameID),
			})
		}
	} else {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      msg.Chat.ID,
			Text:        confirmText,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboards.ResultConfirmKeyboard(gameID),
		})
	}
}

// HandleConfirmResult processes "confirm_result:gameID" callback.
// Saves final chips and marks results_confirmed. Updates hub in group.
func (h *CollectResultsHandler) HandleConfirmResult(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}
	userID := cb.From.ID
	gameID := parseGameIDFromCallback(cb.Data)

	// Retrieve chips from FSM.
	sess, ok := h.fsmStore.Get(userID)
	if !ok || sess.Data == nil {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			Text:            "Нет данных для подтверждения. Введи остаток снова.",
			ShowAlert:       true,
		})
		return
	}

	confirmGameID, _ := sess.Data["confirm_game_id"].(int64)
	chips, chipsOk := sess.Data["confirm_chips"].(int64)
	if !chipsOk || confirmGameID != gameID {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			Text:            "Нет данных для подтверждения. Введи остаток снова.",
			ShowAlert:       true,
		})
		return
	}

	participant, participants, err := h.games.SubmitResult(ctx, gameID, userID, chips)
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
				Text:            "Сбор результатов уже завершён",
				ShowAlert:       true,
			})
		default:
			slog.Error("collect: SubmitResult failed", "gameID", gameID, "userID", userID, "err", err)
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Ошибка сохранения. Попробуй ещё раз.",
				ShowAlert:       true,
			})
		}
		return
	}

	// Already confirmed case — SubmitResult returns current state with no error.
	if participant.ResultsConfirmed {
		// Check if it was already confirmed before this call by comparing FSM state.
		// SubmitResult is idempotent — just notify if already confirmed.
	}

	// Clear confirm data from FSM.
	delete(sess.Data, "confirm_chips")
	delete(sess.Data, "confirm_game_id")
	h.fsmStore.Set(userID, sess)

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "✅ Данные подтверждены!",
	})

	// Edit the message to show confirmed state.
	game, err := h.games.GetGameByID(ctx, gameID)
	if err != nil {
		slog.Error("collect: GetGameByID after confirm", "gameID", gameID, "err", err)
		return
	}

	confirmedText := fmt.Sprintf("✅ Данные подтверждены!\n\nИгра #%d\nТвой остаток: <b>%d</b>", gameID, chips)
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    cb.Message.Message.Chat.ID,
		MessageID: cb.Message.Message.ID,
		Text:      confirmedText,
		ParseMode: models.ParseModeHTML,
	})

	// Update hub in group chat.
	h.updateHubAfterConfirm(ctx, b, game, participants)
}

// HandleEditResult processes "edit_result:gameID" callback.
// Returns user to the chip collection message.
func (h *CollectResultsHandler) HandleEditResult(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}
	userID := cb.From.ID
	gameID := parseGameIDFromCallback(cb.Data)

	// Clear confirm data from FSM.
	sess, _ := h.fsmStore.Get(userID)
	if sess != nil && sess.Data != nil {
		delete(sess.Data, "confirm_chips")
		delete(sess.Data, "confirm_game_id")
		h.fsmStore.Set(userID, sess)
	}

	game, err := h.games.GetGameByID(ctx, gameID)
	if err != nil {
		slog.Error("collect: GetGameByID on edit", "gameID", gameID, "err", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})
		return
	}
	p, err := h.games.GetParticipant(ctx, gameID, userID)
	if err != nil {
		slog.Error("collect: GetParticipant on edit", "gameID", gameID, "userID", userID, "err", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})

	text := views.RenderChipsInput(game, p)
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      cb.Message.Message.Chat.ID,
		MessageID:   cb.Message.Message.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboards.ChipsCollectionKeyboard(gameID),
	})
	if err != nil {
		slog.Error("collect: EditMessageText (edit_result) failed", "err", err)
	}
}

// updateHubAfterConfirm edits the hub message in the group chat after a participant confirms results.
func (h *CollectResultsHandler) updateHubAfterConfirm(ctx context.Context, b *bot.Bot, game *domain.Game, participants []domain.Participant) {
	if game.HubMessageID == 0 {
		return
	}

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
		slog.Error("collect: hub EditMessageText failed", "gameID", game.ID, "err", err)
	}
}
