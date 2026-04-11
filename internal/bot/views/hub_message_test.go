package views

import (
	"strings"
	"testing"
	"time"

	"poker-bot/internal/domain"
)

func makeGame(id, buyIn int64, status domain.GameStatus) *domain.Game {
	return &domain.Game{
		ID:        id,
		ChatID:    100,
		CreatorID: 1,
		BuyIn:     buyIn,
		Status:    status,
		CreatedAt: time.Date(2026, 4, 11, 18, 30, 0, 0, time.UTC),
	}
}

func makeParticipant(playerID int64, rebuy int, confirmed bool) domain.Participant {
	return domain.Participant{
		GameID:           1,
		PlayerID:         playerID,
		RebuyCount:       rebuy,
		ResultsConfirmed: confirmed,
	}
}

func makePlayer(id int64, name string) *domain.Player {
	return &domain.Player{TelegramID: id, DisplayName: name}
}

func TestRenderHub_ContainsAllFields(t *testing.T) {
	game := makeGame(7, 1000, domain.GameStatusActive)
	participants := []domain.Participant{
		makeParticipant(1, 0, false),
		makeParticipant(2, 1, false),
		makeParticipant(3, 0, false),
	}
	players := map[int64]*domain.Player{
		1: makePlayer(1, "Алиса"),
		2: makePlayer(2, "Боб"),
		3: makePlayer(3, "Вася"),
	}

	result := RenderHub(game, participants, players)

	checks := []string{
		"Игра #7",
		"активна",
		"1000 ₽",
		"Алиса",
		"Боб",
		"Вася",
		"Игроки (3)",
		"💰 Банк",
		"⏱ Начало",
		"18:30",
	}
	for _, want := range checks {
		if !strings.Contains(result, want) {
			t.Errorf("RenderHub output missing %q\ngot:\n%s", want, result)
		}
	}
}

func TestRenderHub_BankCalculation(t *testing.T) {
	// 3 players, buy_in=1000, rebuy_count=[0,1,2] → bank = 1000+2000+3000 = 6000
	game := makeGame(1, 1000, domain.GameStatusActive)
	participants := []domain.Participant{
		makeParticipant(1, 0, false),
		makeParticipant(2, 1, false),
		makeParticipant(3, 2, false),
	}
	players := map[int64]*domain.Player{
		1: makePlayer(1, "A"),
		2: makePlayer(2, "B"),
		3: makePlayer(3, "C"),
	}

	result := RenderHub(game, participants, players)

	if !strings.Contains(result, "6000 ₽") {
		t.Errorf("expected bank 6000 ₽, got:\n%s", result)
	}
}

func TestRenderHub_CollectingResults_ShowsStatusAndIcons(t *testing.T) {
	game := makeGame(3, 500, domain.GameStatusCollectingResults)
	participants := []domain.Participant{
		makeParticipant(1, 0, true),
		makeParticipant(2, 0, false),
	}
	players := map[int64]*domain.Player{
		1: makePlayer(1, "Петя"),
		2: makePlayer(2, "Маша"),
	}

	result := RenderHub(game, participants, players)

	if !strings.Contains(result, "сбор результатов") {
		t.Errorf("expected 'сбор результатов', got:\n%s", result)
	}
	if !strings.Contains(result, "✅") {
		t.Errorf("expected ✅ for confirmed participant, got:\n%s", result)
	}
	if !strings.Contains(result, "⏳") {
		t.Errorf("expected ⏳ for unconfirmed participant, got:\n%s", result)
	}
}

func TestRenderHub_RebuyDisplay(t *testing.T) {
	game := makeGame(1, 1000, domain.GameStatusActive)
	participants := []domain.Participant{
		makeParticipant(1, 3, false),
	}
	players := map[int64]*domain.Player{1: makePlayer(1, "Игрок")}

	result := RenderHub(game, participants, players)

	if !strings.Contains(result, "×3 докуп") {
		t.Errorf("expected rebuy indicator ×3 докуп, got:\n%s", result)
	}
}

func TestRenderHub_UnknownPlayerFallback(t *testing.T) {
	game := makeGame(1, 1000, domain.GameStatusActive)
	participants := []domain.Participant{makeParticipant(42, 0, false)}
	players := map[int64]*domain.Player{} // no player info

	result := RenderHub(game, participants, players)

	if !strings.Contains(result, "Игрок #42") {
		t.Errorf("expected fallback name 'Игрок #42', got:\n%s", result)
	}
}
