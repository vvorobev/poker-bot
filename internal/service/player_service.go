package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"poker-bot/internal/domain"
)

var phoneRe = regexp.MustCompile(`^\+7\d{10}$`)

// PlayerService handles player registration and profile management.
type PlayerService struct {
	players PlayerRepository
}

// NewPlayerService creates a PlayerService backed by the given repository.
func NewPlayerService(players PlayerRepository) *PlayerService {
	return &PlayerService{players: players}
}

// ValidatePhone reports whether phone is a valid Russian mobile number
// in the form +7XXXXXXXXXX (plus-sign, digit 7, then exactly 10 digits).
func ValidatePhone(phone string) bool {
	return phoneRe.MatchString(phone)
}

// RegisterPlayer creates or updates the player's profile in the database.
func (s *PlayerService) RegisterPlayer(ctx context.Context, telegramID int64, username, displayName, phone, bank string) error {
	p := &domain.Player{
		TelegramID:       telegramID,
		TelegramUsername: username,
		DisplayName:      displayName,
		PhoneNumber:      phone,
		BankName:         bank,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := s.players.Upsert(ctx, p); err != nil {
		return fmt.Errorf("PlayerService.RegisterPlayer: %w", err)
	}
	return nil
}

// GetPlayer fetches a player by Telegram ID.
// Returns domain.ErrNotFound when the player has not registered yet.
func (s *PlayerService) GetPlayer(ctx context.Context, telegramID int64) (*domain.Player, error) {
	return s.players.GetByTelegramID(ctx, telegramID)
}

// IsRegistered reports whether the given Telegram user has completed registration.
func (s *PlayerService) IsRegistered(ctx context.Context, telegramID int64) bool {
	_, err := s.players.GetByTelegramID(ctx, telegramID)
	return err == nil
}

// UpdateDisplayName changes the player's display name.
// Returns domain.ErrNotFound if the player is not registered.
func (s *PlayerService) UpdateDisplayName(ctx context.Context, telegramID int64, name string) error {
	p, err := s.players.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return err
	}
	p.DisplayName = name
	if err := s.players.Upsert(ctx, p); err != nil {
		return fmt.Errorf("PlayerService.UpdateDisplayName: %w", err)
	}
	return nil
}
