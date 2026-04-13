package views

import (
	"strings"
	"testing"
	"time"

	"poker-bot/internal/domain"
)

func makeFinishedGame(id int64, buyIn int64, createdAt time.Time, finishedAt *time.Time) *domain.Game {
	return &domain.Game{
		ID:         id,
		BuyIn:      buyIn,
		CreatedAt:  createdAt,
		FinishedAt: finishedAt,
	}
}

func chipsPtr(v int64) *int64 { return &v }

func makeParticipantWithChips(playerID int64, rebuyCount int, finalChips int64) domain.Participant {
	return domain.Participant{
		PlayerID:   playerID,
		RebuyCount: rebuyCount,
		FinalChips: chipsPtr(finalChips),
	}
}

func makeNamedPlayers(ids ...int64) map[int64]*domain.Player {
	m := make(map[int64]*domain.Player)
	names := []string{"Alice", "Bob", "Carol", "Dave", "Eve", "Frank"}
	for i, id := range ids {
		name := names[i%len(names)]
		m[id] = &domain.Player{TelegramID: id, DisplayName: name}
	}
	return m
}

func TestRenderGameSummary_Medals(t *testing.T) {
	// 6 participants: 3 positive, 3 negative
	created := time.Now()
	fin := created.Add(2 * time.Hour)
	game := makeFinishedGame(1, 1000, created, &fin)

	// buy_in=1000, invested per player = 1000*(1+0)=1000
	// chips: 3000, 2000, 1500, 800, 500, 200
	// balances: +2000, +1000, +500, -200, -500, -800
	participants := []domain.Participant{
		makeParticipantWithChips(1, 0, 3000),
		makeParticipantWithChips(2, 0, 2000),
		makeParticipantWithChips(3, 0, 1500),
		makeParticipantWithChips(4, 0, 800),
		makeParticipantWithChips(5, 0, 500),
		makeParticipantWithChips(6, 0, 200),
	}

	players := makeNamedPlayers(1, 2, 3, 4, 5, 6)
	// Names: Alice(1), Bob(2), Carol(3), Dave(4), Eve(5), Frank(6)

	text := RenderGameSummary(game, participants, nil, players)

	// Medals for first 3 positive
	if !strings.Contains(text, "🥇") {
		t.Error("should have gold medal for 1st place")
	}
	if !strings.Contains(text, "🥈") {
		t.Error("should have silver medal for 2nd place")
	}
	if !strings.Contains(text, "🥉") {
		t.Error("should have bronze medal for 3rd place")
	}

	// Losers marked with ❌
	occurrences := strings.Count(text, "❌")
	if occurrences != 3 {
		t.Errorf("expected 3 ❌ for losers, got %d", occurrences)
	}

	// Alice (+2000) should appear before Bob (+1000) in text
	alicePos := strings.Index(text, "Alice")
	bobPos := strings.Index(text, "Bob")
	if alicePos >= bobPos {
		t.Error("Alice (+2000) should appear before Bob (+1000)")
	}
}

func TestRenderGameSummary_Duration(t *testing.T) {
	created := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	fin := time.Date(2026, 1, 1, 13, 20, 0, 0, time.UTC) // 3h 20m
	game := makeFinishedGame(2, 1000, created, &fin)

	participants := []domain.Participant{
		makeParticipantWithChips(1, 0, 1000),
		makeParticipantWithChips(2, 0, 1000),
	}
	players := makeNamedPlayers(1, 2)

	text := RenderGameSummary(game, participants, nil, players)

	if !strings.Contains(text, "3ч") {
		t.Error("duration should contain '3ч'")
	}
	if !strings.Contains(text, "20мин") {
		t.Error("duration should contain '20мин'")
	}
}

func TestRenderGameSummary_Transfers(t *testing.T) {
	created := time.Now()
	fin := created.Add(time.Hour)
	game := makeFinishedGame(3, 1000, created, &fin)

	participants := []domain.Participant{
		makeParticipantWithChips(1, 0, 500),
		makeParticipantWithChips(2, 0, 1500),
	}
	players := map[int64]*domain.Player{
		1: {TelegramID: 1, DisplayName: "Alice"},
		2: {TelegramID: 2, DisplayName: "Bob"},
	}
	transfers := []domain.Transfer{
		{FromPlayerID: 1, ToPlayerID: 2, Amount: 500},
	}

	text := RenderGameSummary(game, participants, transfers, players)

	if !strings.Contains(text, "💸") {
		t.Error("should have transfers section emoji")
	}
	if !strings.Contains(text, "Переводы") {
		t.Error("should have 'Переводы' section")
	}
	if !strings.Contains(text, "Alice") {
		t.Error("should contain sender name")
	}
	if !strings.Contains(text, "Bob") {
		t.Error("should contain receiver name")
	}
	if !strings.Contains(text, "500 ₽") {
		t.Error("should contain transfer amount")
	}
	// Arrow between names
	if !strings.Contains(text, "→") {
		t.Error("should contain → between sender and receiver")
	}
}

func TestRenderGameSummary_HTMLParseMode(t *testing.T) {
	created := time.Now()
	fin := created.Add(time.Hour)
	game := makeFinishedGame(4, 1000, created, &fin)

	participants := []domain.Participant{
		makeParticipantWithChips(1, 0, 2000),
		makeParticipantWithChips(2, 0, 0),
	}
	players := makeNamedPlayers(1, 2)

	text := RenderGameSummary(game, participants, nil, players)

	if !strings.Contains(text, "<b>") {
		t.Error("should use HTML bold tags")
	}
}

func TestRenderGameSummary_Header(t *testing.T) {
	created := time.Now()
	fin := created.Add(time.Hour)
	game := makeFinishedGame(99, 500, created, &fin)

	participants := []domain.Participant{
		makeParticipantWithChips(1, 1, 1000), // invested 500*(1+1)=1000
		makeParticipantWithChips(2, 0, 1000), // invested 500
	}
	players := makeNamedPlayers(1, 2)

	text := RenderGameSummary(game, participants, nil, players)

	if !strings.Contains(text, "🎰") {
		t.Error("header should have 🎰")
	}
	if !strings.Contains(text, "#99") {
		t.Error("should show game ID 99")
	}
	// Bank = 500*(1+1) + 500*(1+0) = 1000+500 = 1500
	if !strings.Contains(text, "1500 ₽") {
		t.Error("bank should be 1500")
	}
	if !strings.Contains(text, "2") { // Игроков: 2
		t.Error("should show player count")
	}
}
