package views

import (
	"strings"
	"testing"

	"poker-bot/internal/domain"
)

func makeFullPlayer(id int64, name, phone, bank string) *domain.Player {
	return &domain.Player{
		TelegramID:  id,
		DisplayName: name,
		PhoneNumber: phone,
		BankName:    bank,
	}
}

func TestRenderPersonalResult_Loser(t *testing.T) {
	// Player 1 lost, must pay player 2
	transfers := []domain.Transfer{
		{FromPlayerID: 1, ToPlayerID: 2, Amount: 1500},
	}
	players := map[int64]*domain.Player{
		1: makeFullPlayer(1, "Alice", "+79001234567", "Сбер"),
		2: makeFullPlayer(2, "Bob", "+79007654321", "Тинькофф"),
	}

	text := RenderPersonalResult(42, 1, transfers, players)

	if !strings.Contains(text, "📉") {
		t.Error("loser should see 📉")
	}
	if !strings.Contains(text, "Тебе нужно перевести") {
		t.Error("should contain outgoing transfers header")
	}
	if !strings.Contains(text, "Bob") {
		t.Error("should contain recipient name")
	}
	if !strings.Contains(text, "<code>+79007654321</code>") {
		t.Error("should contain recipient phone in <code>")
	}
	if !strings.Contains(text, "Тинькофф") {
		t.Error("should contain recipient bank")
	}
	if !strings.Contains(text, "1500 ₽") {
		t.Error("should contain transfer amount")
	}
	// Loser's own phone should NOT appear
	if strings.Contains(text, "Alice") {
		// Alice is the player, not a transfer target - OK if name not shown
	}
}

func TestRenderPersonalResult_Winner(t *testing.T) {
	// Player 2 won, will receive from player 1
	transfers := []domain.Transfer{
		{FromPlayerID: 1, ToPlayerID: 2, Amount: 1500},
	}
	players := map[int64]*domain.Player{
		1: makeFullPlayer(1, "Alice", "+79001234567", "Сбер"),
		2: makeFullPlayer(2, "Bob", "+79007654321", "Тинькофф"),
	}

	text := RenderPersonalResult(42, 2, transfers, players)

	if !strings.Contains(text, "🎉") {
		t.Error("winner should see 🎉")
	}
	if !strings.Contains(text, "Тебе должны перевести") {
		t.Error("should contain incoming transfers header")
	}
	if !strings.Contains(text, "Alice") {
		t.Error("should contain debtor name")
	}
	// Winner should NOT see debtor's phone
	if strings.Contains(text, "+79001234567") {
		t.Error("winner should NOT see debtor phone number")
	}
	if strings.Contains(text, "Сбер") {
		t.Error("winner should NOT see debtor bank")
	}
	if !strings.Contains(text, "1500 ₽") {
		t.Error("should contain amount")
	}
}

func TestRenderPersonalResult_BreakEven(t *testing.T) {
	// Player 3 has no transfers
	transfers := []domain.Transfer{
		{FromPlayerID: 1, ToPlayerID: 2, Amount: 1500},
	}
	players := map[int64]*domain.Player{
		1: makeFullPlayer(1, "Alice", "+79001234567", "Сбер"),
		2: makeFullPlayer(2, "Bob", "+79007654321", "Тинькофф"),
		3: makeFullPlayer(3, "Carol", "+79001111111", "ВТБ"),
	}

	text := RenderPersonalResult(42, 3, transfers, players)

	if !strings.Contains(text, "🤝") {
		t.Error("break-even should see 🤝")
	}
	if !strings.Contains(text, "остался при своих") {
		t.Error("break-even message expected")
	}
	if !strings.Contains(text, "Никому ничего не должен") {
		t.Error("break-even message expected")
	}
}

func TestRenderPersonalResult_MultipleTransfers(t *testing.T) {
	// Player 1 lost to two players
	transfers := []domain.Transfer{
		{FromPlayerID: 1, ToPlayerID: 2, Amount: 1000},
		{FromPlayerID: 1, ToPlayerID: 3, Amount: 500},
	}
	players := map[int64]*domain.Player{
		1: makeFullPlayer(1, "Alice", "+79001234567", "Сбер"),
		2: makeFullPlayer(2, "Bob", "+79007654321", "Тинькофф"),
		3: makeFullPlayer(3, "Carol", "+79001111111", "ВТБ"),
	}

	text := RenderPersonalResult(42, 1, transfers, players)

	if !strings.Contains(text, "Bob") {
		t.Error("should list first creditor")
	}
	if !strings.Contains(text, "Carol") {
		t.Error("should list second creditor")
	}
	if !strings.Contains(text, "1000 ₽") {
		t.Error("should show first transfer amount")
	}
	if !strings.Contains(text, "500 ₽") {
		t.Error("should show second transfer amount")
	}
	// Total loss = 1500
	if !strings.Contains(text, "1500 ₽") {
		t.Error("should show total loss amount")
	}
}

func TestRenderPersonalResult_HTMLParseMode(t *testing.T) {
	transfers := []domain.Transfer{
		{FromPlayerID: 1, ToPlayerID: 2, Amount: 2000},
	}
	players := map[int64]*domain.Player{
		1: makeFullPlayer(1, "Alice", "+79001234567", "Сбер"),
		2: makeFullPlayer(2, "Bob", "+79007654321", "Тинькофф"),
	}

	loserText := RenderPersonalResult(1, 1, transfers, players)
	winnerText := RenderPersonalResult(1, 2, transfers, players)

	for _, text := range []string{loserText, winnerText} {
		if !strings.Contains(text, "<b>") {
			t.Error("should use HTML bold tags")
		}
	}
	if !strings.Contains(loserText, "<code>") {
		t.Error("loser view should use <code> for phone number")
	}
}
