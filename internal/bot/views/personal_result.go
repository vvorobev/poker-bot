package views

import (
	"fmt"
	"strings"

	"poker-bot/internal/domain"
)

// RenderPersonalResult renders a personal settlement message for a player.
// transfers is the full list of transfers for the game.
// players maps PlayerID to Player for name/phone/bank resolution.
func RenderPersonalResult(gameID int64, playerID int64, transfers []domain.Transfer, players map[int64]*domain.Player) string {
	// Compute balance and relevant transfers
	var balance int64
	var outgoing []domain.Transfer // player owes
	var incoming []domain.Transfer // player is owed

	for _, t := range transfers {
		if t.FromPlayerID == playerID {
			balance -= t.Amount
			outgoing = append(outgoing, t)
		}
		if t.ToPlayerID == playerID {
			balance += t.Amount
			incoming = append(incoming, t)
		}
	}

	if balance == 0 {
		return "🤝 Ты остался при своих. Никому ничего не должен"
	}

	var b strings.Builder

	if balance < 0 {
		fmt.Fprintf(&b, "📉 <b>Результат — Игра #%d</b>\n\n", gameID)
		fmt.Fprintf(&b, "Ты проиграл: <b>%d ₽</b>\n\n", -balance)
		b.WriteString("Тебе нужно перевести:\n")
		for _, t := range outgoing {
			name := resolveDisplayName(t.ToPlayerID, players)
			phone := ""
			bank := ""
			if p, ok := players[t.ToPlayerID]; ok && p != nil {
				phone = p.PhoneNumber
				bank = p.BankName
			}
			fmt.Fprintf(&b, "• %s — <b>%d ₽</b>\n", name, t.Amount)
			if phone != "" {
				fmt.Fprintf(&b, "  <code>%s</code>", phone)
				if bank != "" {
					fmt.Fprintf(&b, " (%s)", bank)
				}
				b.WriteString("\n")
			}
		}
	} else {
		fmt.Fprintf(&b, "🎉 <b>Результат — Игра #%d</b>\n\n", gameID)
		fmt.Fprintf(&b, "Ты выиграл: <b>+%d ₽</b>\n\n", balance)
		b.WriteString("Тебе должны перевести:\n")
		for _, t := range incoming {
			name := resolveDisplayName(t.FromPlayerID, players)
			fmt.Fprintf(&b, "• %s — <b>%d ₽</b>\n", name, t.Amount)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}
