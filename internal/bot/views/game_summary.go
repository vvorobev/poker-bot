package views

import (
	"fmt"
	"sort"
	"strings"

	"poker-bot/internal/domain"
)

// RenderGameSummary renders the final game summary for the group chat.
// participants must have FinalChips set; players maps PlayerID → Player for names.
func RenderGameSummary(game *domain.Game, participants []domain.Participant, transfers []domain.Transfer, players map[int64]*domain.Player) string {
	var b strings.Builder

	// Header
	fmt.Fprintf(&b, "🎰 <b>Игра #%d завершена</b>\n", game.ID)

	// Duration
	if game.FinishedAt != nil {
		dur := game.FinishedAt.Sub(game.CreatedAt)
		h := int(dur.Hours())
		m := int(dur.Minutes()) % 60
		fmt.Fprintf(&b, "Длительность: %dч %dмин\n", h, m)
	}

	// Bank
	var bank int64
	for _, p := range participants {
		bank += game.BuyIn * int64(1+p.RebuyCount)
	}
	fmt.Fprintf(&b, "Банк: <b>%d ₽</b>\n", bank)
	fmt.Fprintf(&b, "Игроков: %d\n", len(participants))

	// Sort participants by result descending (balance = chips - invested)
	type playerResult struct {
		p       domain.Participant
		balance int64
	}
	results := make([]playerResult, 0, len(participants))
	for _, p := range participants {
		var chips int64
		if p.FinalChips != nil {
			chips = *p.FinalChips
		}
		invested := game.BuyIn * int64(1+p.RebuyCount)
		results = append(results, playerResult{p: p, balance: chips - invested})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].balance > results[j].balance
	})

	medals := []string{"🥇", "🥈", "🥉"}
	b.WriteString("\n")
	for i, r := range results {
		name := resolveDisplayName(r.p.PlayerID, players)
		var prefix string
		if r.balance > 0 && i < len(medals) {
			prefix = medals[i]
		} else if r.balance <= 0 {
			prefix = "❌"
		} else {
			prefix = "  "
		}
		if r.balance > 0 {
			fmt.Fprintf(&b, "%s %s: <b>+%d ₽</b>\n", prefix, name, r.balance)
		} else if r.balance < 0 {
			fmt.Fprintf(&b, "%s %s: <b>%d ₽</b>\n", prefix, name, r.balance)
		} else {
			fmt.Fprintf(&b, "🤝 %s: <b>0 ₽</b>\n", name)
		}
	}

	// Transfers section
	if len(transfers) > 0 {
		b.WriteString("\n💸 <b>Переводы:</b>\n")
		for _, t := range transfers {
			from := resolveDisplayName(t.FromPlayerID, players)
			to := resolveDisplayName(t.ToPlayerID, players)
			fmt.Fprintf(&b, "• %s → %s: <b>%d ₽</b>\n", from, to, t.Amount)
		}
	} else {
		b.WriteString("\n🤝 Все остались при своих, переводов нет.")
	}

	return strings.TrimRight(b.String(), "\n")
}
