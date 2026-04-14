package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	telebot "poker-bot/internal/bot"
	"poker-bot/internal/config"
	"poker-bot/internal/fsm"
	"poker-bot/internal/logging"
	"poker-bot/internal/service"
	"poker-bot/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// logging not set up yet, use stderr
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	logging.Setup(cfg.LogPath)

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}

	if err := storage.RunMigrations(db); err != nil {
		slog.Error("failed to run migrations", "err", err)
		db.Close()
		os.Exit(1)
	}

	playerRepo := storage.NewPlayerRepo(db)
	gameRepo := storage.NewGameRepo(db)
	participantRepo := storage.NewParticipantRepo(db)
	settlementRepo := storage.NewSettlementRepo(db)
	txManager := storage.NewTxManager(db)

	playerSvc := service.NewPlayerService(playerRepo)
	gameSvc := service.NewGameService(gameRepo, participantRepo, settlementRepo, txManager)
	settlementSvc := service.NewSettlementService()
	fsmStore := fsm.NewStore()

	deps := telebot.Deps{
		AllowedChatID: cfg.AllowedChatID,
		Players:       playerSvc,
		Games:         gameSvc,
		FSM:           fsmStore,
		Settlements:   settlementSvc,
		ProxyURL:      cfg.ProxyURL,
	}

	b, err := telebot.New(cfg.BotToken, deps)
	if err != nil {
		slog.Error("failed to create bot", "err", err)
		db.Close()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("bot started", "db", cfg.DBPath)

	// Start long polling — blocks until ctx is cancelled.
	b.Start(ctx)

	stop()

	fsmStore.Stop()

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = shutdownCtx

	if err := db.Close(); err != nil {
		slog.Error("error closing database", "err", err)
	}

	slog.Info("bot stopped gracefully")
}
