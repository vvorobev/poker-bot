package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"poker-bot/internal/domain"
)

// GameService handles game lifecycle: creation, joining, rebuys.
type GameService struct {
	games        GameRepository
	participants ParticipantRepository
	tx           TxManager
}

// NewGameService creates a GameService.
func NewGameService(games GameRepository, participants ParticipantRepository, tx TxManager) *GameService {
	return &GameService{
		games:        games,
		participants: participants,
		tx:           tx,
	}
}

// NewGame creates a new game in chatID with creatorID as the first participant.
// buyIn must be between 100 and 100_000 (inclusive).
// Returns ErrGameAlreadyActive if an active game already exists in chatID.
func (s *GameService) NewGame(ctx context.Context, chatID, creatorID, buyIn int64) (*domain.Game, error) {
	if buyIn < 100 || buyIn > 100_000 {
		return nil, fmt.Errorf("buyIn must be between 100 and 100000, got %d", buyIn)
	}

	var game *domain.Game
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		// Check for existing active game
		_, err := s.games.GetActiveByChatID(ctx, chatID)
		if err == nil {
			return domain.ErrGameAlreadyActive
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("GameService.NewGame check active: %w", err)
		}

		// Create the game
		g := &domain.Game{
			ChatID:    chatID,
			CreatorID: creatorID,
			BuyIn:     buyIn,
			Status:    domain.GameStatusActive,
			CreatedAt: time.Now().UTC(),
		}
		id, err := s.games.Create(ctx, g)
		if err != nil {
			return fmt.Errorf("GameService.NewGame create: %w", err)
		}
		g.ID = id

		// Add creator as first participant
		p := &domain.Participant{
			GameID:   id,
			PlayerID: creatorID,
			JoinedAt: time.Now().UTC(),
		}
		if err := s.participants.Join(ctx, p); err != nil {
			return fmt.Errorf("GameService.NewGame join creator: %w", err)
		}

		game = g
		return nil
	})
	if err != nil {
		return nil, err
	}
	return game, nil
}

// GetActiveGame returns the active game for a chat, or domain.ErrNotFound.
func (s *GameService) GetActiveGame(ctx context.Context, chatID int64) (*domain.Game, error) {
	return s.games.GetActiveByChatID(ctx, chatID)
}

// Join adds playerID to the active game gameID.
// Returns ErrAlreadyJoined, ErrNotFound, or ErrGameNotActive as appropriate.
func (s *GameService) Join(ctx context.Context, gameID, playerID int64) (*domain.Game, []domain.Participant, error) {
	var (
		game         *domain.Game
		participants []domain.Participant
	)
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		g, err := s.games.GetByID(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status != domain.GameStatusActive {
			return domain.ErrGameNotActive
		}

		p := &domain.Participant{
			GameID:   gameID,
			PlayerID: playerID,
			JoinedAt: time.Now().UTC(),
		}
		if err := s.participants.Join(ctx, p); err != nil {
			return err
		}

		list, err := s.participants.ListByGame(ctx, gameID)
		if err != nil {
			return fmt.Errorf("GameService.Join list: %w", err)
		}
		game = g
		participants = list
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return game, participants, nil
}

// Rebuy increments rebuy_count for playerID in gameID.
// Returns ErrNotParticipant if playerID is not in the game, ErrGameNotActive if game is not active.
func (s *GameService) Rebuy(ctx context.Context, gameID, playerID int64) (*domain.Game, []domain.Participant, error) {
	var (
		game         *domain.Game
		participants []domain.Participant
	)
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		g, err := s.games.GetByID(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status != domain.GameStatusActive {
			return domain.ErrGameNotActive
		}

		_, err = s.participants.GetByGameAndPlayer(ctx, gameID, playerID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotParticipant
		}
		if err != nil {
			return err
		}

		if err := s.participants.IncrementRebuy(ctx, gameID, playerID); err != nil {
			return fmt.Errorf("GameService.Rebuy increment: %w", err)
		}

		list, err := s.participants.ListByGame(ctx, gameID)
		if err != nil {
			return fmt.Errorf("GameService.Rebuy list: %w", err)
		}
		game = g
		participants = list
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return game, participants, nil
}

// CancelRebuy decrements rebuy_count for playerID in gameID (minimum 0).
// Returns ErrNotParticipant if playerID is not in the game, ErrGameNotActive if game is not active.
func (s *GameService) CancelRebuy(ctx context.Context, gameID, playerID int64) (*domain.Game, []domain.Participant, error) {
	var (
		game         *domain.Game
		participants []domain.Participant
	)
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		g, err := s.games.GetByID(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status != domain.GameStatusActive {
			return domain.ErrGameNotActive
		}

		_, err = s.participants.GetByGameAndPlayer(ctx, gameID, playerID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotParticipant
		}
		if err != nil {
			return err
		}

		if err := s.participants.DecrementRebuy(ctx, gameID, playerID); err != nil {
			return fmt.Errorf("GameService.CancelRebuy decrement: %w", err)
		}

		list, err := s.participants.ListByGame(ctx, gameID)
		if err != nil {
			return fmt.Errorf("GameService.CancelRebuy list: %w", err)
		}
		game = g
		participants = list
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return game, participants, nil
}
