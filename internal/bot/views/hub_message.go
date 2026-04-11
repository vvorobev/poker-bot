package views

import (
	"fmt"
	"strings"

	"poker-bot/internal/domain"
)

// RenderHub renders the hub message text for a game.
// players maps PlayerID to Player for display name resolution.
func RenderHub(game *domain.Game, participants []domain.Participant, players map[int64]*domain.Player) string {
	var b strings.Builder

	statusText := statusLabel(game.Status)
	fmt.Fprintf(&b, "🎰 Игра #%d — %s\n", game.ID, statusText)
	fmt.Fprintf(&b, "Бай-ин: <b>%d ₽</b>\n", game.BuyIn)

	creatorName := resolveDisplayName(game.CreatorID, players)
	fmt.Fprintf(&b, "Создатель: %s\n", creatorName)

	fmt.Fprintf(&b, "Игроки (%d):\n", len(participants))
	for _, p := range participants {
		name := resolveDisplayName(p.PlayerID, players)
		line := formatParticipantLine(name, p, game.Status)
		fmt.Fprintf(&b, "%s\n", line)
	}

	bank := computeBank(game.BuyIn, participants)
	fmt.Fprintf(&b, "💰 Банк: <b>%d ₽</b>\n", bank)
	fmt.Fprintf(&b, "⏱ Начало: %s", game.CreatedAt.Format("15:04"))

	return b.String()
}

func statusLabel(s domain.GameStatus) string {
	switch s {
	case domain.GameStatusCollectingResults:
		return "сбор результатов"
	case domain.GameStatusFinished:
		return "завершена"
	default:
		return "активна"
	}
}

func resolveDisplayName(playerID int64, players map[int64]*domain.Player) string {
	if p, ok := players[playerID]; ok && p != nil {
		return p.DisplayName
	}
	return fmt.Sprintf("Игрок #%d", playerID)
}

func formatParticipantLine(name string, p domain.Participant, status domain.GameStatus) string {
	var prefix string
	if status == domain.GameStatusCollectingResults || status == domain.GameStatusFinished {
		if p.ResultsConfirmed {
			prefix = "✅ "
		} else {
			prefix = "⏳ "
		}
	} else {
		prefix = "• "
	}

	if p.RebuyCount > 0 {
		return fmt.Sprintf("%s%s (×%d докуп)", prefix, name, p.RebuyCount)
	}
	return prefix + name
}

func computeBank(buyIn int64, participants []domain.Participant) int64 {
	var total int64
	for _, p := range participants {
		total += buyIn * int64(1+p.RebuyCount)
	}
	return total
}
