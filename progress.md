# Progress Log

Этот файл ведётся агентами автоматически. Каждый агент добавляет запись после завершения задачи.

## Формат записи
```
### [TASK-XXX] Название задачи
**Дата:** YYYY-MM-DD
**Статус:** done
**Summary:** что было сделано, какие файлы изменены, какие решения приняты
```

---

### [TASK-001] Инициализация Go-проекта
**Дата:** 2026-04-10
**Статус:** done
**Summary:** Создан go.mod (module poker-bot, go 1.22). Добавлены stub Go-файлы во все пакеты: cmd/bot/main.go, internal/config, domain, storage, service, bot, fsm, logging. Создан .gitignore (.env, *.db, *.log, /bot). Создан .env.example с полями TELEGRAM_BOT_TOKEN, DB_PATH, LOG_PATH, ALLOWED_CHAT_ID, ADMIN_USER_IDS. Папки уже существовали. Все acceptance criteria выполнены.
**Следующий шаг:** TASK-002 (internal/config), TASK-003 (logging), TASK-004 (storage) — все critical, зависят только от TASK-001.

---

### [TASK-002] Пакет internal/config
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Код уже был реализован в предыдущей итерации (файл internal/config/config.go). Верифицировано: go vet проходит, Load() возвращает ошибки при отсутствии TELEGRAM_BOT_TOKEN/ALLOWED_CHAT_ID, DB_PATH defaults to ./poker.db. Статус обновлён до done.

---

### [TASK-009] Domain types: Player, Game, Participant, Settlement, errors
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Созданы файлы:
- internal/domain/player.go — struct Player (telegram_id, username, display_name, phone, bank, timestamps)
- internal/domain/game.go — struct Game, struct Participant, type GameStatus string с константами Active/CollectingResults/Finished
- internal/domain/settlement.go — struct Settlement, struct Transfer{FromPlayerID, ToPlayerID, Amount}
- internal/domain/errors.go — 7 sentinel errors через errors.New()
Удалён stub-файл domain.go. Пакет не импортирует internal/*. go build и go vet проходят.
**Следующий шаг:** TASK-003 (logging, lumberjack.v2), TASK-004 (storage, modernc.org/sqlite) — оба critical, зависят только от TASK-001. После них разблокируются TASK-005, TASK-006 → TASK-007 (telegram bot).

---

### [TASK-003] Пакет internal/logging
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Реализована функция Setup(logPath string) в internal/logging/logging.go. Используется lumberjack.v2 (gopkg.in/natefinch/lumberjack.v2 v2.2.1) для ротации логов: 10 MB, 5 бэкапов, без ограничения по возрасту. Формат JSON через slog.NewJSONHandler. При logPath="" логи идут в stdout. Добавлена зависимость в go.mod. go vet и тесты проходят.
**Следующий шаг:** TASK-004 (storage, modernc.org/sqlite) — последний critical, блокирующий TASK-006 → цепочку до telegram bot.

---

### [TASK-004] Пакет internal/storage: SQLite Open + migrations runner
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Созданы файлы:
- internal/storage/sqlite.go — Open(dbPath) через modernc.org/sqlite, применяет PRAGMA journal_mode=WAL и foreign_keys=ON
- internal/storage/migrations.go — RunMigrations(db) с go:embed для migrations/*.sql, запускает файлы в алфавитном порядке, идемпотентен
- internal/storage/storage_test.go — тесты: WAL mode (на file DB), :memory: открытие, RunMigrations x2, FK enforcement
Все 4 теста проходят. WAL проверяется на file-based DB (в :memory: SQLite не поддерживает WAL — возвращает "memory").

---

### [TASK-005] SQL-миграция 001_init.sql: 4 таблицы
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан internal/storage/migrations/001_init.sql с таблицами players, games, game_participants, settlements. Все FK с ON DELETE RESTRICT, UNIQUE(game_id, player_id) в game_participants. Идемпотентность через CREATE TABLE IF NOT EXISTS. Верифицировано тестами: все 4 таблицы создаются, FK нарушения возвращают ошибку.
**Следующий шаг:** TASK-006 (graceful shutdown в main.go) — все зависимости (002, 003, 004) теперь done. Затем TASK-007 (telegram bot) → TASK-008 (middleware) → TASK-010 (repositories).

---

### [TASK-006] Graceful shutdown в main.go
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Реализован cmd/bot/main.go с полным graceful shutdown:
- config.Load() → logging.Setup() → storage.Open() → storage.RunMigrations()
- signal.NotifyContext с os.Interrupt и syscall.SIGTERM
- 5-секундный таймаут на завершение через context.WithTimeout
- db.Close() после остановки polling
- slog.Info("bot stopped gracefully") в конце
go vet и go build проходят без ошибок. Тесты в storage пакете проходят (cached).
**Следующий шаг:** TASK-007 (Telegram bot, go-telegram/bot, long polling, /ping) — разблокирован. TASK-010 (TxManager + PlayerRepository) также разблокирован параллельно.

---

### [TASK-011] GameRepository и ParticipantRepository
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Созданы файлы:
- internal/service/interfaces.go — добавлены интерфейсы GameRepository (Create, GetByID, GetActiveByChatID, UpdateStatus, SetHubMessageID, SetFinishedAt) и ParticipantRepository (Join, IncrementRebuy, DecrementRebuy, ListByGame, SetFinalChips, SetResultsConfirmed, GetByGameAndPlayer)
- internal/storage/game_repo.go — GameRepo и ParticipantRepo; DecrementRebuy использует MAX(0, rebuy_count-1) в SQL; Join определяет ErrAlreadyJoined по "UNIQUE constraint failed" в тексте ошибки
- internal/storage/game_repo_test.go — 11 тестов покрывают все acceptance criteria: GetActiveByChatID, ErrAlreadyJoined, DecrementRebuy floor=0 и т.д.
- internal/storage/tx.go — расширен интерфейс querier: добавлен QueryContext для поддержки ListByGame
go vet чист, go test ./... проходит (13 тестов в storage).
**Следующий шаг:** TASK-012 (SettlementRepository: SaveAll, ListByGame) — разблокирован.

---

### [TASK-007] Базовый Telegram бот
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Добавлена зависимость github.com/go-telegram/bot v1.20.0. Реализован internal/bot/bot.go: функция New(token) создаёт бота с обработчиком ошибок через slog.Error и регистрирует /ping → "pong". Обновлён cmd/bot/main.go: вызов telebot.New(cfg.BotToken) и b.Start(ctx) для long polling; b.Start блокирует до отмены ctx, после чего graceful shutdown как прежде. go vet чист, go test ./... проходит.
**Следующий шаг:** TASK-008 (middleware контроля доступа по ALLOWED_CHAT_ID) — разблокирован. Также параллельно доступны TASK-012 (SettlementRepository) и TASK-013 (FSM), TASK-014 (PlayerService).

---

### [TASK-008] Middleware контроля доступа
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан пакет internal/bot/middleware/ с файлами:
- auth.go — Auth middleware: group chat → проверяет chat_id == ALLOWED_CHAT_ID; private chat → вызывает getChatMember(ALLOWED_CHAT_ID, userID), кэширует результат в memberCache с TTL 10 минут. Неавторизованным отправляет "Этот бот работает только в закрытой группе. Обратись к владельцу." Middleware применяется глобально через WithMiddlewares.
- auth_test.go — 5 unit-тестов: cache TTL/expiry, extractChatInfo для message/private/callback/empty update.
Обновлён internal/bot/bot.go: New() теперь принимает allowedChatID int64 и применяет auth middleware. Обновлён cmd/bot/main.go: передаёт cfg.AllowedChatID. go vet и go test ./... проходят.
**Следующий шаг:** TASK-013 (FSM), TASK-014 (PlayerService), TASK-019 (GameService.NewGame), TASK-021 (View hub), TASK-022 (Keyboards) — все разблокированы, можно работать параллельно. TASK-015 (/start handler) теперь разблокирован (зависит от 008, 013, 014).

---

### [TASK-014] PlayerService: валидация телефона, онбординг, профиль
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан internal/service/player_service.go:
- `ValidatePhone(phone string) bool` — regexp `^\+7\d{10}$`, покрыты valid/invalid кейсы
- `RegisterPlayer(ctx, id, username, displayName, phone, bank)` — делегирует в PlayerRepository.Upsert
- `GetPlayer(ctx, id)` — делегирует в PlayerRepository.GetByTelegramID
- `IsRegistered(ctx, id) bool` — обёртка над GetPlayer
- `UpdateDisplayName(ctx, id, name)` — GetByTelegramID + Upsert
Создан internal/service/player_service_test.go (6 тестов, пакет service_test, использует реальную SQLite :memory:).
go vet чист, go test ./... проходит.
**Следующий шаг:** TASK-013 (FSM) — разблокирован ранее; вместе с TASK-014 разблокирует TASK-015 (/start handler). Параллельно: TASK-019 (GameService), TASK-021 (View hub), TASK-022 (Keyboards), TASK-024 (GameService join/rebuy), TASK-034 (SettlementService.Compute) — все critical и разблокированы.

---

### [TASK-013] FSM: in-memory хранилище сессий
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Реализован пакет internal/fsm:
- fsm.go — Store с методами Get/Set/Clear, thread-safe (sync.RWMutex). Session содержит State, Data map[string]any, UpdatedAt. TTL=30 мин, фоновая горутина (evictExpired) чистит устаревшие каждые 5 минут. Stop() закрывает горутину.
- states.go — константы StateIdle, StateAwaitingPhone, StateAwaitingBank, StateAwaitingChipsInput, StateAwaitingBuyIn
- fsm_test.go — 5 тестов: Set/Get, Clear, GetMissing, UpdatedAt, concurrent access
go test -race проходит без data race.
**Следующий шаг:** TASK-015 (/start handler) — теперь разблокирован (зависит от TASK-008 done, TASK-013 done, TASK-014 done). Параллельно: TASK-019 (GameService.NewGame), TASK-021 (View RenderHub), TASK-022 (Keyboards), TASK-024 (GameService Join/Rebuy) — все critical и разблокированы.

---

### [TASK-019 + TASK-024] GameService: NewGame, GetActiveGame, Join, Rebuy, CancelRebuy
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан internal/service/game_service.go:
- `NewGame(ctx, chatID, creatorID, buyIn)` — валидация buyIn (100..100_000), проверка ErrGameAlreadyActive через GetActiveByChatID, создание игры в транзакции, добавление создателя как первого участника через ParticipantRepo.Join
- `GetActiveGame(ctx, chatID)` — делегирует в GameRepository.GetActiveByChatID
- `Join(ctx, gameID, playerID)` — проверяет статус active, добавляет участника, возвращает (Game, []Participant)
- `Rebuy(ctx, gameID, playerID)` — проверяет статус + ErrNotParticipant, инкрементирует rebuy_count
- `CancelRebuy(ctx, gameID, playerID)` — декрементирует rebuy_count (floor=0 обеспечивается SQL)
Все методы используют TxManager.RunInTx. Создан game_service_test.go (9 тестов). go vet чист, go test ./... проходит.
**Следующий шаг:** TASK-015 (/start handler), TASK-020 (/newgame handler), TASK-021 (View RenderHub), TASK-022 (Keyboards) — все critical и разблокированы.

---

### [TASK-034] SettlementService.Compute: жадный алгоритм минимизации переводов
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан internal/service/settlement_service.go:
- `Compute(participants []domain.Participant, buyIn int64) []domain.Transfer` — жадный алгоритм минимизации переводов
- Вычисляет баланс каждого участника: FinalChips - buyIn*(1+rebuyCount)
- Разбивает на debtors (balance<0) и creditors (balance>0), игроки с balance=0 исключаются
- Сортирует оба списка по убыванию |balance|, жадно сопоставляет наибольшего должника с наибольшим кредитором
- nil FinalChips трактуется как 0 фишек
- Создан settlement_service_test.go (6 тестов: 4 игрока с балансами, все в нуле, 1 winner/1 loser, пустые участники, nil FinalChips, upper bound на кол-во переводов)
go vet чист, go test ./... проходит.
**Следующий шаг:** TASK-033 (SettlementService.Validate) — разблокирован (зависит от TASK-028), TASK-037 (View group summary) — разблокирован (зависит от TASK-034). Параллельно: TASK-021 (RenderHub), TASK-022 (Keyboards), TASK-028 (GameService.FinishGame), TASK-015 (/start handler).

---

### [TASK-021] View: рендеринг сообщения-хаба игры
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан internal/bot/views/hub_message.go:
- `RenderHub(game *domain.Game, participants []domain.Participant, players map[int64]*domain.Player) string` — рендерит HTML-текст хаба
- Статус: "активна" / "сбор результатов" / "завершена"
- Банк = Σ(buy_in × (1 + rebuy_count)) — протестировано: 3 игрока с rebuy=[0,1,2] → 6000 ₽
- При collecting_results/finished — ⏳/✅ вместо bullet points
- Докупы отображаются как "(×N докуп)"
- Fallback для неизвестных игроков: "Игрок #N"
- HTML parse mode, суммы в <b>
- Добавлен players map[int64]*domain.Player как третий параметр (не указан в task spec, но необходим для отображения имён)
- Создан hub_message_test.go: 5 тестов, все проходят
**Следующий шаг:** TASK-022 (Keyboards: HubKeyboard, BankKeyboard, BuyInKeyboard, ChipsInputKeyboard) — разблокирован. Вместе с TASK-021 unblocks TASK-023 (публикация хаба).

---

### [TASK-022] Keyboards: inline-клавиатуры
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан пакет internal/bot/keyboards/:
- keyboards.go — 4 функции: HubKeyboard(gameID int64), BankKeyboard(), BuyInKeyboard(), ChipsInputKeyboard()
- HubKeyboard: 4 кнопки в 2 строках с callback_data формата "action:game_id" (join, rebuy, cancel_rebuy, finish)
- BankKeyboard: 8 кнопок (Тинькофф, Сбербанк, Альфа-Банк, ВТБ, Райффайзен, Озон Банк, Яндекс Банк, Другой) с callback "bank:<name>"
- BuyInKeyboard: 1 кнопка "1000 ₽ (по умолчанию)" с callback "buyin:1000"
- ChipsInputKeyboard: 2 кнопки "Ввести в фишках" / "Ввести в рублях"
- keyboards_test.go — 6 тестов, все проходят
go vet чист, go test ./... проходит.
**Следующий шаг:** TASK-023 (публикация хаба) и TASK-026 (hub updater) теперь разблокированы. Параллельно: TASK-015 (/start handler), TASK-020 (/newgame handler), TASK-028 (GameService.FinishGame).

---

### [TASK-028] GameService.FinishGame
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Добавлен метод `FinishGame(ctx, gameID, actorID)` в `internal/service/game_service.go`:
- Проверяет статус игры (возвращает `ErrGameNotActive` если не `active`)
- Проверяет участие actorID (возвращает `ErrNotParticipant` если не участник)
- Переводит игру в `CollectingResults` через `GameRepository.UpdateStatus`
- Возвращает обновлённую игру и полный список участников для рассылки личных сообщений
- Всё выполняется в транзакции через `TxManager.RunInTx`
Добавлены 3 теста (TestFinishGame_Success, TestFinishGame_ErrGameNotActive, TestFinishGame_ErrNotParticipant). go vet и go test ./... проходят.
**Следующий шаг:** TASK-033 (SettlementService.Validate) и TASK-029 (callback «finish») теперь разблокированы. Параллельно: TASK-012 (SettlementRepository), TASK-015 (/start handler), TASK-020 (/newgame handler).

---

### [TASK-015] Хендлер /start: онбординг и профиль игрока
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Реализован TASK-015:
- internal/bot/handlers/start.go — StartHandler с двумя методами:
  - `Handle` (/start): только private chat; зарегистрированным → профиль (имя/телефон/банк) + inline "Ок"; незарегистрированным → приветствие + ReplyKeyboard ("📱 Поделиться контактом" с RequestContact=true, "✏️ Ввести номер вручную")
  - `HandleManualPhone` (обработчик текста "✏️ Ввести номер вручную"): устанавливает FSM = StateAwaitingPhone, убирает ReplyKeyboard, просит ввести номер
  - `HandleStartOK`: answerCallbackQuery для кнопки "Ок"
- internal/bot/bot.go — введён Deps struct (AllowedChatID, Players, FSM); зарегистрированы все три хендлера
- cmd/bot/main.go — wiring: PlayerRepo → PlayerService → fsm.Store → Deps → telebot.New; fsmStore.Stop() при shutdown
go build, go vet, go test ./... — чисто.
**Следующий шаг:** TASK-016 (онбординг: получение и валидация номера телефона — contact handler + text handler для StateAwaitingPhone) — теперь разблокирован.

---

### [TASK-016] Онбординг: телефон (contact + ручной ввод)
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан internal/bot/handlers/phone.go с PhoneHandler:
- `HandleContact` — принимает message.Contact, нормализует номер через `normalizePhone` (добавляет "+" если отсутствует), валидирует через `service.ValidatePhone`; при невалидном → просит ввести вручную; при валидном → сохраняет в FSM Data["phone"], переводит в StateAwaitingBank, показывает BankKeyboard
- `HandlePhoneText` — текстовый хендлер для FSM StateAwaitingPhone; валидирует введённый номер; при ошибке → просит повторить в формате +7XXXXXXXXXX; при успехе → аналогично HandleContact
- `normalizePhone` — добавляет "+" к номеру если не начинается с "+"
Зарегистрированы в bot.go через `RegisterHandlerMatchFunc`: contact-хендлер по `update.Message.Contact != nil`, text-хендлер по FSM-состоянию `StateAwaitingPhone`.
Добавлен phone_test.go (тест normalizePhone). go vet и go test ./... проходят.
**Следующий шаг:** TASK-017 (выбор банка + финальное сохранение профиля) — теперь разблокирован.

---

### [TASK-017] Онбординг: выбор банка и финальное сохранение профиля
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан internal/bot/handlers/bank.go с BankHandler:
- `HandleBankCallback` — обрабатывает callback `bank:<name>`. При выборе "Другой" устанавливает `bank_custom=true` в FSM и запрашивает текстовый ввод. При любом другом банке — вызывает finishRegistration.
- `HandleBankText` — текстовый хендлер для FSM StateAwaitingBank + bank_custom=true; принимает название банка, вызывает finishRegistration.
- `finishRegistration` — извлекает phone из FSM, собирает displayName из Telegram first+last name, вызывает PlayerService.RegisterPlayer, очищает FSM, отправляет подтверждение с именем/телефоном/банком.
Зарегистрированы в bot.go: callback-хендлер через HasPrefix("bank:"), text-хендлер через MatchFunc по состоянию FSM.
go vet и go test ./... проходят.
**Следующий шаг:** TASK-018 (/name команда, priority high) или TASK-020 (/newgame handler, priority critical) — оба разблокированы.

---

### [TASK-010] TxManager и PlayerRepository
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Созданы файлы:
- internal/service/interfaces.go — интерфейсы TxManager{RunInTx} и PlayerRepository{GetByTelegramID, Upsert}
- internal/storage/tx.go — TxManager через context-injection: *sql.Tx хранится в ctx по приватному ключу txKey{}; extractDB() возвращает tx из ctx или db; RunInTx делает commit/rollback с recover для паник
- internal/storage/player_repo.go — PlayerRepo: GetByTelegramID возвращает domain.ErrNotFound при sql.ErrNoRows; Upsert использует INSERT ... ON CONFLICT DO UPDATE; оба метода используют extractDB()
- internal/storage/player_repo_test.go — 5 тестов: Upsert+Get, ErrNotFound, обновление через Upsert, commit транзакции, rollback транзакции
go vet чист, go test ./... проходит.
**Следующий шаг:** TASK-011 (GameRepository + ParticipantRepository) — разблокирован.

