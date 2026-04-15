# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# poker-bot project rules

Use caveman communication mode (full level) for all responses in this project.

## What this is

Telegram bot for poker night settlement. Players register (phone + bank), join games via hub message in group chat, enter final chip counts, bot calculates minimal transfers to settle debts.

## Commands

```bash
# Run
go run ./cmd/bot/

# Build
go build -o poker-bot ./cmd/bot/

# Test all
go test ./...

# Test single package
go test ./internal/service/...

# Deploy to prod (requires SSH host "tgn" and .env.prod)
./deploy.sh
```

## Config (env / .env file)

| Var | Required | Default |
|-----|----------|---------|
| `TELEGRAM_BOT_TOKEN` | yes | — |
| `ALLOWED_CHAT_ID` | yes | — |
| `DB_PATH` | no | `./poker.db` |
| `LOG_PATH` | no | `./bot.log` |
| `ADMIN_USER_IDS` | no | — |
| `PROXY_URL` | no | — (socks5/http proxy for Telegram API) |

## Architecture

```
cmd/bot/main.go          — wiring: config → DB → repos → services → bot
internal/config/         — env loading
internal/domain/         — plain structs: Player, Game, Participant, Settlement, Transfer
internal/storage/        — SQLite via database/sql; repos + TxManager; migrations embedded
internal/service/        — business logic (PlayerService, GameService, SettlementService)
internal/fsm/            — in-memory per-user state machine (Store + Session)
internal/bot/
  bot.go                 — handler registration (first-match, order matters)
  handlers/              — one file per command/flow
  views/                 — message text builders (pure functions)
  keyboards/             — inline keyboard builders
  hub/                   — updater that edits the group hub message
  middleware/            — auth: drop updates not from AllowedChatID
internal/logging/        — slog setup
```

### Key flows

1. **Registration** — `/start` in private chat → FSM: idle → awaiting_phone → awaiting_bank → idle. Player stored with phone + bank.
2. **New game** — `/newgame` in group → asks buy-in → creates Game row → posts hub message with Join/Rebuy/Finish buttons. Hub message updated on every participant change.
3. **Collecting results** — Finish button → status = `collecting_results` → each player gets DM with chip input form → confirm → when all confirmed, SettlementService computes minimal transfers → results posted to group.
4. **Settlement calc** — `SettlementService.Calculate` in `internal/service/settlement_service.go`: net position per player, greedy creditor-debtor matching.

### FSM states

`StateIdle` → `StateAwaitingPhone` → `StateAwaitingBank` → back to idle  
`StateAwaitingBuyIn` (newgame flow)  
`StateAwaitingChipsInput` (collect results flow)

### Handler registration order matters

`bot.go` uses first-match semantics. FSM-gated handlers (MatchFunc checking session state) are registered before fallback handlers. Fallback handlers **must** be last.

### DB

SQLite, single file. Schema in `internal/storage/migrations/001_init.sql`. Migrations run at startup via `storage.RunMigrations`. All monetary values stored as integers (chips/kopecks).
