package domain

import "errors"

var (
	ErrNotFound                = errors.New("not found")
	ErrAlreadyJoined           = errors.New("already joined")
	ErrNotParticipant          = errors.New("not a participant")
	ErrBankMismatch            = errors.New("bank mismatch")
	ErrGameAlreadyActive       = errors.New("game already active")
	ErrGameNotActive           = errors.New("game not active")
	ErrResultsAlreadyConfirmed = errors.New("results already confirmed")
)
