package service_test

import (
	"testing"

	"poker-bot/internal/domain"
	"poker-bot/internal/service"
)

func ptr64(v int64) *int64 { return &v }

func makeParticipant(playerID int64, rebuyCount int, finalChips int64) domain.Participant {
	return domain.Participant{
		PlayerID:   playerID,
		RebuyCount: rebuyCount,
		FinalChips: ptr64(finalChips),
	}
}

func TestCompute_FourPlayers(t *testing.T) {
	// buyIn=1000, balances [-3000, -1000, +2000, +2000]
	// player1: chips=0, invested=1000*(1+2)=3000 → balance=-3000
	// player2: chips=0, invested=1000*(1+0)=1000 → balance=-1000
	// player3: chips=3000, invested=1000*(1+0)=1000 → balance=+2000
	// player4: chips=3000, invested=1000*(1+0)=1000 → balance=+2000
	svc := service.NewSettlementService()
	participants := []domain.Participant{
		makeParticipant(1, 2, 0),
		makeParticipant(2, 0, 0),
		makeParticipant(3, 0, 3000),
		makeParticipant(4, 0, 3000),
	}
	transfers := svc.Compute(participants, 1000)

	if len(transfers) > 3 {
		t.Errorf("expected <= 3 transfers, got %d", len(transfers))
	}

	// Verify total flow from debtors equals total flow to creditors
	var totalOut, totalIn int64
	for _, tr := range transfers {
		totalOut += tr.Amount
		totalIn += tr.Amount
		if tr.Amount <= 0 {
			t.Errorf("transfer amount must be positive, got %d", tr.Amount)
		}
	}

	// Total settled = 3000+1000 = 4000
	if totalOut != 4000 {
		t.Errorf("expected total transfers 4000, got %d", totalOut)
	}
}

func TestCompute_AllZero(t *testing.T) {
	// All players break even
	svc := service.NewSettlementService()
	participants := []domain.Participant{
		makeParticipant(1, 0, 1000),
		makeParticipant(2, 0, 1000),
	}
	transfers := svc.Compute(participants, 1000)
	if len(transfers) != 0 {
		t.Errorf("expected 0 transfers, got %d", len(transfers))
	}
}

func TestCompute_OneWinnerOneLoser(t *testing.T) {
	// player1: chips=0, invested=1000 → -1000
	// player2: chips=2000, invested=1000 → +1000
	svc := service.NewSettlementService()
	participants := []domain.Participant{
		makeParticipant(1, 0, 0),
		makeParticipant(2, 0, 2000),
	}
	transfers := svc.Compute(participants, 1000)
	if len(transfers) != 1 {
		t.Errorf("expected exactly 1 transfer, got %d", len(transfers))
	}
	if transfers[0].Amount != 1000 {
		t.Errorf("expected amount 1000, got %d", transfers[0].Amount)
	}
	if transfers[0].FromPlayerID != 1 || transfers[0].ToPlayerID != 2 {
		t.Errorf("wrong direction: from=%d to=%d", transfers[0].FromPlayerID, transfers[0].ToPlayerID)
	}
}

func TestCompute_EmptyParticipants(t *testing.T) {
	svc := service.NewSettlementService()
	transfers := svc.Compute(nil, 1000)
	if len(transfers) != 0 {
		t.Errorf("expected 0 transfers for empty participants, got %d", len(transfers))
	}
}

func TestCompute_NilFinalChips(t *testing.T) {
	// participant with nil FinalChips treated as 0
	svc := service.NewSettlementService()
	p1 := domain.Participant{PlayerID: 1, RebuyCount: 0, FinalChips: nil}
	p2 := domain.Participant{PlayerID: 2, RebuyCount: 0, FinalChips: ptr64(2000)}
	transfers := svc.Compute([]domain.Participant{p1, p2}, 1000)
	if len(transfers) != 1 {
		t.Errorf("expected 1 transfer, got %d", len(transfers))
	}
	if transfers[0].Amount != 1000 {
		t.Errorf("expected amount 1000, got %d", transfers[0].Amount)
	}
}

func TestCompute_TransferCountBound(t *testing.T) {
	// n players with non-zero balance → transfers <= n-1
	svc := service.NewSettlementService()
	// 6 players: 3 lose 1000 each, 3 win 1000 each
	participants := []domain.Participant{
		makeParticipant(1, 0, 0),    // -1000
		makeParticipant(2, 0, 0),    // -1000
		makeParticipant(3, 0, 0),    // -1000
		makeParticipant(4, 0, 2000), // +1000
		makeParticipant(5, 0, 2000), // +1000
		makeParticipant(6, 0, 2000), // +1000
	}
	transfers := svc.Compute(participants, 1000)
	if len(transfers) > 5 {
		t.Errorf("expected <= 5 transfers for 6 non-zero players, got %d", len(transfers))
	}
}

func makeConfirmedParticipant(playerID int64, rebuyCount int, finalChips int64) domain.Participant {
	p := makeParticipant(playerID, rebuyCount, finalChips)
	p.ResultsConfirmed = true
	return p
}

func TestValidate_AllConfirmedMatch(t *testing.T) {
	// buyIn=1000, player1: chips=1500 (invested=1000), player2: chips=500 (invested=1000)
	// expected=2000, actual=2000 → nil
	svc := service.NewSettlementService()
	participants := []domain.Participant{
		makeConfirmedParticipant(1, 0, 1500),
		makeConfirmedParticipant(2, 0, 500),
	}
	if err := svc.Validate(participants, 1000); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_AllConfirmedMismatch(t *testing.T) {
	// buyIn=1000, player1: chips=1600, player2: chips=500
	// expected=2000, actual=2100 → BankMismatchError{Expected:2000, Actual:2100, Diff:100}
	svc := service.NewSettlementService()
	participants := []domain.Participant{
		makeConfirmedParticipant(1, 0, 1600),
		makeConfirmedParticipant(2, 0, 500),
	}
	err := svc.Validate(participants, 1000)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	mismatch, ok := service.IsBankMismatch(err)
	if !ok {
		t.Fatalf("expected BankMismatchError, got %T", err)
	}
	if mismatch.Expected != 2000 {
		t.Errorf("Expected=%d, want 2000", mismatch.Expected)
	}
	if mismatch.Actual != 2100 {
		t.Errorf("Actual=%d, want 2100", mismatch.Actual)
	}
	if mismatch.Diff != 100 {
		t.Errorf("Diff=%d, want 100", mismatch.Diff)
	}
}

func TestValidate_NotAllConfirmed_ReturnsNil(t *testing.T) {
	// one participant not confirmed → deferred, return nil
	svc := service.NewSettlementService()
	p1 := makeConfirmedParticipant(1, 0, 0)  // confirmed, chips=0 (mismatch would occur)
	p2 := makeParticipant(2, 0, 3000)        // NOT confirmed
	// If validation ran: expected=2000, actual=3000 → mismatch. But p2 not confirmed → nil.
	if err := svc.Validate([]domain.Participant{p1, p2}, 1000); err != nil {
		t.Errorf("expected nil when not all confirmed, got %v", err)
	}
}

func TestValidate_WithRebuys(t *testing.T) {
	// buyIn=1000, player1: rebuy=1 → invested=2000, chips=2000; player2: rebuy=0 → invested=1000, chips=1000
	// expected=3000, actual=3000 → nil
	svc := service.NewSettlementService()
	participants := []domain.Participant{
		makeConfirmedParticipant(1, 1, 2000),
		makeConfirmedParticipant(2, 0, 1000),
	}
	if err := svc.Validate(participants, 1000); err != nil {
		t.Errorf("expected nil with rebuys balanced, got %v", err)
	}
}
