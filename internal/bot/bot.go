package bot

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/net/proxy"

	"poker-bot/internal/bot/handlers"
	"poker-bot/internal/bot/middleware"
	"poker-bot/internal/fsm"
	"poker-bot/internal/service"
)

var botCommands = []models.BotCommand{
	{Command: "start", Description: "Регистрация / профиль"},
	{Command: "newgame", Description: "Создать новую игру"},
	{Command: "game", Description: "Текущая игра"},
	{Command: "name", Description: "Изменить отображаемое имя"},
	{Command: "edit", Description: "Изменить введённые данные"},
	{Command: "cancel", Description: "Отменить текущее действие"},
	{Command: "help", Description: "Список команд"},
}

// Deps holds the dependencies injected into the bot's handlers.
type Deps struct {
	AllowedChatID int64
	Players       *service.PlayerService
	Games         *service.GameService
	FSM           *fsm.Store
	Settlements   *service.SettlementService
	ProxyURL      string
}

func proxyHTTPClient(proxyURL string) (*http.Client, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(u, proxy.Direct)
		if err != nil {
			return nil, err
		}
		return &http.Client{Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		}}, nil
	default:
		return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(u)}}, nil
	}
}

// New creates and configures a Telegram bot instance.
func New(token string, deps Deps) (*bot.Bot, error) {
	auth := middleware.NewAuth(deps.AllowedChatID)

	opts := []bot.Option{
		bot.WithErrorsHandler(func(err error) {
			slog.Error("telegram bot error", "err", err)
		}),
		bot.WithMiddlewares(auth.Middleware),
	}

	if deps.ProxyURL != "" {
		httpClient, err := proxyHTTPClient(deps.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("proxy: %w", err)
		}
		opts = append(opts, bot.WithHTTPClient(30*time.Second, httpClient))
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/ping", bot.MatchTypeExact, pingHandler)

	startH := handlers.NewStartHandler(deps.Players, deps.FSM)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, startH.Handle)
	b.RegisterHandler(bot.HandlerTypeMessageText, handlers.ManualPhoneButtonText, bot.MatchTypeExact, startH.HandleManualPhone)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "start:ok", bot.MatchTypeExact, handlers.HandleStartOK)

	phoneH := handlers.NewPhoneHandler(deps.Players, deps.FSM)
	// Contact sharing button handler.
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.Contact != nil &&
			update.Message.Chat.Type == models.ChatTypePrivate
	}, phoneH.HandleContact)
	// Manual phone text input handler (active when FSM is in StateAwaitingPhone).
	fsmStore := deps.FSM
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		if update.Message == nil || update.Message.Text == "" {
			return false
		}
		if update.Message.Chat.Type != models.ChatTypePrivate {
			return false
		}
		sess, ok := fsmStore.Get(update.Message.From.ID)
		return ok && sess.State == fsm.StateAwaitingPhone
	}, phoneH.HandlePhoneText)

	hubH := handlers.NewHubCallbackHandler(deps.Players, deps.Games, deps.FSM)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "join:")
	}, hubH.HandleJoin)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "rebuy:")
	}, hubH.HandleRebuy)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "cancel_rebuy:")
	}, hubH.HandleCancelRebuy)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "finish:")
	}, hubH.HandleFinish)

	newGameH := handlers.NewNewGameHandler(deps.Players, deps.Games, deps.FSM, deps.AllowedChatID)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/newgame", bot.MatchTypeExact, newGameH.Handle)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "buyin:")
	}, newGameH.HandleBuyInCallback)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		if update.Message == nil || update.Message.Text == "" {
			return false
		}
		sess, ok := fsmStore.Get(update.Message.From.ID)
		return ok && sess.State == fsm.StateAwaitingBuyIn
	}, newGameH.HandleBuyInText)

	collectH := handlers.NewCollectResultsHandler(deps.Players, deps.Games, deps.FSM, deps.Settlements)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "collect_rebuy_plus:")
	}, collectH.HandleRebuyPlus)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "collect_rebuy_minus:")
	}, collectH.HandleRebuyMinus)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "chips_mode:")
	}, collectH.HandleChipsMode)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		if update.Message == nil || update.Message.Text == "" {
			return false
		}
		if update.Message.Chat.Type != models.ChatTypePrivate {
			return false
		}
		sess, ok := fsmStore.Get(update.Message.From.ID)
		return ok && sess.State == fsm.StateAwaitingChipsInput
	}, collectH.HandleChipsText)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "confirm_result:")
	}, collectH.HandleConfirmResult)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "edit_result:")
	}, collectH.HandleEditResult)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/edit", bot.MatchTypeExact, collectH.HandleEditCommand)

	gameH := handlers.NewGameCommandHandler(deps.Players, deps.Games, deps.AllowedChatID)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/game", bot.MatchTypeExact, gameH.Handle)

	nameH := handlers.NewNameHandler(deps.Players)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/name", bot.MatchTypePrefix, nameH.Handle)

	cancelH := handlers.NewCancelHandler(deps.FSM)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/cancel", bot.MatchTypeExact, cancelH.Handle)

	helpH := handlers.NewHelpHandler()
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, helpH.Handle)

	bankH := handlers.NewBankHandler(deps.Players, deps.FSM)
	// Bank selection callback handler (bank:<name>).
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil &&
			strings.HasPrefix(update.CallbackQuery.Data, "bank:")
	}, bankH.HandleBankCallback)
	// Custom bank name text input (FSM state=AwaitingBank, bank_custom=true).
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		if update.Message == nil || update.Message.Text == "" {
			return false
		}
		if update.Message.Chat.Type != models.ChatTypePrivate {
			return false
		}
		sess, ok := fsmStore.Get(update.Message.From.ID)
		if !ok || sess.State != fsm.StateAwaitingBank {
			return false
		}
		customFlag, _ := sess.Data["bank_custom"].(bool)
		return customFlag
	}, bankH.HandleBankText)

	// Fallback handlers — must be registered last (first-match semantics).
	fallbackH := handlers.NewFallbackHandler(deps.FSM)
	b.RegisterHandlerMatchFunc(handlers.MatchUnknownCommand, fallbackH.HandleUnknownCommand)
	b.RegisterHandlerMatchFunc(fallbackH.MatchPlainTextFallback(fsmStore), fallbackH.HandlePlainText)

	if _, err := b.SetMyCommands(context.Background(), &bot.SetMyCommandsParams{
		Commands: botCommands,
	}); err != nil {
		slog.Error("failed to set bot commands", "err", err)
	}

	return b, nil
}

func pingHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "pong",
	})
	if err != nil {
		slog.Error("failed to send pong", "err", err)
	}
}
