package domain

import "time"

type Player struct {
	TelegramID       int64
	TelegramUsername string
	DisplayName      string
	PhoneNumber      string
	BankName         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
