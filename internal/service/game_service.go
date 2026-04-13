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
	settlements  SettlementRepository
	tx           TxManager
}

// NewGameService creates a GameService.
func NewGameService(games GameRepository, participants ParticipantRepository, settlements SettlementRepository, tx TxManager) *GameService {
	return &GameService{
		games:        games,
		participants: participants,
		settlements:  settlements,
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

// SubmitResult saves final_chips and marks results_confirmed=true for playerID in gameID.
// Returns ErrNotParticipant if playerID is not in the game.
// Returns ErrGameNotActive if game is not in collecting_results status.
// Returns the updated Participant and full participants list.
func (s *GameService) SubmitResult(ctx context.Context, gameID, playerID, finalChips int64) (*domain.Participant, []domain.Participant, error) {
	var (
		p    *domain.Participant
		list []domain.Participant
	)
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		g, err := s.games.GetByID(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status != domain.GameStatusCollectingResults {
			return domain.ErrGameNotActive
		}

		existing, err := s.participants.GetByGameAndPlayer(ctx, gameID, playerID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotParticipant
		}
		if err != nil {
			return err
		}
		if existing.ResultsConfirmed {
			// Already confirmed — return current state without error.
			p = existing
			all, err := s.participants.ListByGame(ctx, gameID)
			if err != nil {
				return fmt.Errorf("SubmitResult list: %w", err)
			}
			list = all
			return nil
		}

		if err := s.participants.SetFinalChips(ctx, gameID, playerID, finalChips); err != nil {
			return fmt.Errorf("SubmitResult SetFinalChips: %w", err)
		}
		if err := s.participants.SetResultsConfirmed(ctx, gameID, playerID); err != nil {
			return fmt.Errorf("SubmitResult SetResultsConfirmed: %w", err)
		}

		updated, err := s.participants.GetByGameAndPlayer(ctx, gameID, playerID)
		if err != nil {
			return fmt.Errorf("SubmitResult get updated: %w", err)
		}
		p = updated

		all, err := s.participants.ListByGame(ctx, gameID)
		if err != nil {
			return fmt.Errorf("SubmitResult list: %w", err)
		}
		list = all
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return p, list, nil
}

// SetHubMessageID stores the Telegram message ID of the hub message for gameID.
func (s *GameService) SetHubMessageID(ctx context.Context, gameID, msgID int64) error {
	return s.games.SetHubMessageID(ctx, gameID, msgID)
}

// FinalizeGame saves settlements, marks game as Finished with finished_at = now().
// Caller must have already validated the bank and computed transfers.
// Returns the updated game.
func (s *GameService) FinalizeGame(ctx context.Context, gameID int64, transfers []domain.Transfer) (*domain.Game, error) {
	var game *domain.Game
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		g, err := s.games.GetByID(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status != domain.GameStatusCollectingResults {
			return domain.ErrGameNotActive
		}

		if err := s.settlements.SaveAll(ctx, gameID, transfers); err != nil {
			return fmt.Errorf("FinalizeGame SaveAll: %w", err)
		}
		if err := s.games.UpdateStatus(ctx, gameID, domain.GameStatusFinished); err != nil {
			return fmt.Errorf("FinalizeGame UpdateStatus: %w", err)
		}
		now := time.Now().UTC()
		if err := s.games.SetFinishedAt(ctx, gameID, now); err != nil {
			return fmt.Errorf("FinalizeGame SetFinishedAt: %w", err)
		}
		g.Status = domain.GameStatusFinished
		g.FinishedAt = &now
		game = g
		return nil
	})
	return game, err
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
