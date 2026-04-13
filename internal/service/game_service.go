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

// FinishGame transitions game gameID to CollectingResults status.
// actorID must be a participant; otherwise ErrNotParticipant is returned.
// Returns ErrGameNotActive if the game is not currently active.
// On success returns the updated game and full participant list.
func (s *GameService) FinishGame(ctx context.Context, gameID, actorID int64) (*domain.Game, []domain.Participant, error) {
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

		_, err = s.participants.GetByGameAndPlayer(ctx, gameID, actorID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotParticipant
		}
		if err != nil {
			return err
		}

		if err := s.games.UpdateStatus(ctx, gameID, domain.GameStatusCollectingResults); err != nil {
			return fmt.Errorf("GameService.FinishGame update status: %w", err)
		}

		list, err := s.participants.ListByGame(ctx, gameID)
		if err != nil {
			return fmt.Errorf("GameService.FinishGame list: %w", err)
		}
		g.Status = domain.GameStatusCollectingResults
		game = g
		participants = list
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return game, participants, nil
}

// GetParticipants returns the participant list for gameID.
func (s *GameService) GetParticipants(ctx context.Context, gameID int64) ([]domain.Participant, error) {
	return s.participants.ListByGame(ctx, gameID)
}

// GetGameByID returns a game by its ID.
func (s *GameService) GetGameByID(ctx context.Context, gameID int64) (*domain.Game, error) {
	return s.games.GetByID(ctx, gameID)
}

// GetParticipant returns a single participant for gameID+playerID.
func (s *GameService) GetParticipant(ctx context.Context, gameID, playerID int64) (*domain.Participant, error) {
	return s.participants.GetByGameAndPlayer(ctx, gameID, playerID)
}

// AdjustRebuyInCollection increments (delta=+1) or decrements (delta=-1) rebuy_count
// for playerID in gameID while the game is in collecting_results status.
// Returns the updated Participant.
func (s *GameService) AdjustRebuyInCollection(ctx context.Context, gameID, playerID int64, delta int) (*domain.Participant, error) {
	var p *domain.Participant
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		g, err := s.games.GetByID(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status != domain.GameStatusCollectingResults {
			return domain.ErrGameNotActive
		}

		_, err = s.participants.GetByGameAndPlayer(ctx, gameID, playerID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotParticipant
		}
		if err != nil {
			return err
		}

		if delta > 0 {
			if err := s.participants.IncrementRebuy(ctx, gameID, playerID); err != nil {
				return fmt.Errorf("AdjustRebuyInCollection increment: %w", err)
			}
		} else if delta < 0 {
			if err := s.participants.DecrementRebuy(ctx, gameID, playerID); err != nil {
				return fmt.Errorf("AdjustRebuyInCollection decrement: %w", err)
			}
		}

		updated, err := s.participants.GetByGameAndPlayer(ctx, gameID, playerID)
		if err != nil {
			return fmt.Errorf("AdjustRebuyInCollection get updated: %w", err)
		}
		p = updated
		return nil
	})
	return p, err
}

// SetHubMessageID stores the Telegram message ID of the hub message for gameID.
func (s *GameService) SetHubMessageID(ctx context.Context, gameID, msgID int64) error {
	return s.games.SetHubMessageID(ctx, gameID, msgID)
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
