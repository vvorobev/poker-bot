package service

import (
	"sort"

	"poker-bot/internal/domain"
)

type SettlementService struct{}

func NewSettlementService() *SettlementService {
	return &SettlementService{}
}

// Compute calculates the minimum set of transfers to settle all debts.
// Algorithm: compute each player's balance, split into debtors and creditors,
// sort by |balance| descending, then greedily match largest debtor with largest creditor.
func (s *SettlementService) Compute(participants []domain.Participant, buyIn int64) []domain.Transfer {
	type balance struct {
		playerID int64
		amount   int64 // positive = creditor, negative = debtor
	}

	balances := make([]balance, 0, len(participants))
	for _, p := range participants {
		var chips int64
		if p.FinalChips != nil {
			chips = *p.FinalChips
		}
		invested := buyIn * int64(1+p.RebuyCount)
		bal := chips - invested
		if bal != 0 {
			balances = append(balances, balance{playerID: p.PlayerID, amount: bal})
		}
	}

	debtors := make([]balance, 0)
	creditors := make([]balance, 0)
	for _, b := range balances {
		if b.amount < 0 {
			debtors = append(debtors, balance{playerID: b.playerID, amount: -b.amount}) // store as positive
		} else {
			creditors = append(creditors, b)
		}
	}

	sort.Slice(debtors, func(i, j int) bool { return debtors[i].amount > debtors[j].amount })
	sort.Slice(creditors, func(i, j int) bool { return creditors[i].amount > creditors[j].amount })

	var transfers []domain.Transfer
	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		debtor := &debtors[i]
		creditor := &creditors[j]

		amount := debtor.amount
		if creditor.amount < amount {
			amount = creditor.amount
		}

		transfers = append(transfers, domain.Transfer{
			FromPlayerID: debtor.playerID,
			ToPlayerID:   creditor.playerID,
			Amount:       amount,
		})

		debtor.amount -= amount
		creditor.amount -= amount

		if debtor.amount == 0 {
			i++
		}
		if creditor.amount == 0 {
			j++
		}
	}

	return transfers
}
