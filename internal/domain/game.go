package domain

import "time"

type GameStatus string

const (
	GameStatusActive             GameStatus = "active"
	GameStatusCollectingResults  GameStatus = "collecting_results"
	GameStatusFinished           GameStatus = "finished"
)

type Game struct {
	ID           int64
	ChatID       int64
	CreatorID    int64
	BuyIn        int64
	HubMessageID int64
	Status       GameStatus
	CreatedAt    time.Time
	FinishedAt   *time.Time
}

type Participant struct {
	ID               int64
	GameID           int64
	PlayerID         int64
	RebuyCount       int
	FinalChips       *int64
	ResultsConfirmed bool
	JoinedAt         time.Time
}
