package domain

type Settlement struct {
	ID           int64
	GameID       int64
	FromPlayerID int64
	ToPlayerID   int64
	Amount       int64
}

type Transfer struct {
	FromPlayerID int64
	ToPlayerID   int64
	Amount       int64
}
