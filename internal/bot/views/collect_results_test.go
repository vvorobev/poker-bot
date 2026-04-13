package views

import (
	"strings"
	"testing"

	"poker-bot/internal/domain"
)

func TestRenderChipsInput(t *testing.T) {
	game := &domain.Game{ID: 5, BuyIn: 1000}
	p := &domain.Participant{RebuyCount: 2}

	text := RenderChipsInput(game, p)

	if !strings.Contains(text, "Игра #5") {
		t.Error("expected game ID in text")
	}
	if !strings.Contains(text, "Бай-ин: <b>1000 ₽</b>") {
		t.Error("expected buy-in in text")
	}
	if !strings.Contains(text, "Докупов: <b>2</b>") {
		t.Error("expected rebuy count in text")
	}
	// Total: 1000 * (1+2) = 3000
	if !strings.Contains(text, "Всего вложено: <b>3000 ₽</b>") {
		t.Error("expected total invested in text")
	}
}

func TestRenderChipsInput_ZeroRebuys(t *testing.T) {
	game := &domain.Game{ID: 1, BuyIn: 500}
	p := &domain.Participant{RebuyCount: 0}

	text := RenderChipsInput(game, p)

	if !strings.Contains(text, "Всего вложено: <b>500 ₽</b>") {
		t.Error("total invested should equal buy-in when no rebuys")
	}
}

func TestRenderChipsConfirm_Profit(t *testing.T) {
	game := &domain.Game{ID: 3, BuyIn: 1000}
	p := &domain.Participant{RebuyCount: 0}

	text := RenderChipsConfirm(game, p, 1500)

	if !strings.Contains(text, "Осталось: <b>1500 ₽</b>") {
		t.Error("expected final chips in text")
	}
	if !strings.Contains(text, "Результат: <b>+500 ₽</b>") {
		t.Error("expected positive result")
	}
}

func TestRenderChipsConfirm_Loss(t *testing.T) {
	game := &domain.Game{ID: 3, BuyIn: 1000}
	p := &domain.Participant{RebuyCount: 1} // invested 2000

	text := RenderChipsConfirm(game, p, 1500)

	if !strings.Contains(text, "Результат: <b>-500 ₽</b>") {
		t.Error("expected negative result")
	}
}

func TestRenderChipsConfirm_BreakEven(t *testing.T) {
	game := &domain.Game{ID: 3, BuyIn: 1000}
	p := &domain.Participant{RebuyCount: 0}

	text := RenderChipsConfirm(game, p, 1000)

	if !strings.Contains(text, "Результат: <b>0 ₽</b>") {
		t.Error("expected zero result")
	}
}
