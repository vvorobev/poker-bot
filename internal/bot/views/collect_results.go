package views

import (
	"fmt"
	"strings"

	"poker-bot/internal/domain"
)

// RenderChipsInput renders the personal chip collection message for a participant.
func RenderChipsInput(game *domain.Game, p *domain.Participant) string {
	var b strings.Builder
	totalInvested := game.BuyIn * int64(1+p.RebuyCount)
	fmt.Fprintf(&b, "🎲 <b>Сбор результатов — Игра #%d</b>\n\n", game.ID)
	fmt.Fprintf(&b, "Бай-ин: <b>%d ₽</b>\n", game.BuyIn)
	fmt.Fprintf(&b, "Докупов: <b>%d</b>\n", p.RebuyCount)
	fmt.Fprintf(&b, "Всего вложено: <b>%d ₽</b>\n\n", totalInvested)
	fmt.Fprintf(&b, "Нажми кнопку ниже, чтобы ввести финальный остаток:")
	return b.String()
}

// RenderChipsConfirm renders the confirmation preview after a player enters their chip count.
func RenderChipsConfirm(game *domain.Game, p *domain.Participant, finalChips int64) string {
	var b strings.Builder
	totalInvested := game.BuyIn * int64(1+p.RebuyCount)
	result := finalChips - totalInvested
	var resultStr string
	if result > 0 {
		resultStr = fmt.Sprintf("+%d ₽", result)
	} else {
		resultStr = fmt.Sprintf("%d ₽", result)
	}
	fmt.Fprintf(&b, "🎲 <b>Подтверждение — Игра #%d</b>\n\n", game.ID)
	fmt.Fprintf(&b, "Докупов: <b>%d</b>\n", p.RebuyCount)
	fmt.Fprintf(&b, "Осталось: <b>%d ₽</b>\n", finalChips)
	fmt.Fprintf(&b, "Результат: <b>%s</b>", resultStr)
	return b.String()
}
