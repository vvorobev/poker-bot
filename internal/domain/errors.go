package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound                = errors.New("not found")
	ErrAlreadyJoined           = errors.New("already joined")
	ErrNotParticipant          = errors.New("not a participant")
	ErrGameAlreadyActive       = errors.New("game already active")
	ErrGameNotActive           = errors.New("game not active")
	ErrResultsAlreadyConfirmed = errors.New("results already confirmed")
)

// BankMismatchError is returned when Σfinal_chips != expected bank.
type BankMismatchError struct {
	Expected int64
	Actual   int64
	Diff     int64
}

func (e *BankMismatchError) Error() string {
	return fmt.Sprintf("bank mismatch: expected %d, actual %d, diff %d", e.Expected, e.Actual, e.Diff)
}

// ErrBankMismatch is a sentinel for errors.Is checks.
var ErrBankMismatch = errors.New("bank mismatch")
