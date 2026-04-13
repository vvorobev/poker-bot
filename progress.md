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

---

### [TASK-020] Хендлер /newgame: запрос бай-ина и создание игры
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан internal/bot/handlers/newgame.go с NewGameHandler:
- `Handle` (/newgame): работает в group и private чатах; незарегистрированным → "Сначала зарегистрируйся через /start"; показывает BuyInKeyboard и переводит FSM в StateAwaitingBuyIn с сохранением game_chat_id (для private чата используется allowedChatID)
- `HandleBuyInCallback`: обрабатывает "buyin:XXXX" callback; парсит сумму; вызывает createGame
- `HandleBuyInText`: обрабатывает текстовый ввод при FSM StateAwaitingBuyIn; валидирует и вызывает createGame
- `createGame`: вызывает GameService.NewGame; при ErrGameAlreadyActive → "В чате уже идёт игра #N. Заверши её перед созданием новой."
Обновлены Deps (добавлен Games *service.GameService), bot.go (регистрация 3 хендлеров), main.go (wiring GameRepo, ParticipantRepo, TxManager, GameService).
go vet и go test ./... проходят.
**Следующий шаг:** TASK-023 (публикация хаба) — теперь разблокирован. Также разблокированы TASK-025 (callback join/rebuy/cancel_rebuy) и TASK-029 (finish callback).

---

### [TASK-023] Публикация сообщения-хаба после создания игры
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Модифицирован `internal/bot/handlers/newgame.go` — функция `createGame`:
- После успешного `GameService.NewGame` загружает участников (`GameService.GetParticipants`) и данные создателя (`PlayerService.GetPlayer`)
- Рендерит текст хаба через `views.RenderHub(game, participants, playerMap)`
- Публикует хаб в `gameChatID` (групповой чат) с `keyboards.HubKeyboard(game.ID)` и `parse_mode=HTML`
- Сохраняет `hub_message_id` через `GameService.SetHubMessageID`
- Если /newgame вызван из private чата (`chatID != gameChatID`), отправляет краткое подтверждение в private чат
Добавлены методы `GetParticipants` и `SetHubMessageID` в `internal/service/game_service.go`.
go vet и go test ./... проходят.
**Следующий шаг:** TASK-025 (callback join/rebuy/cancel_rebuy — теперь разблокирован) — critical.

---

### [TASK-025] Callback хендлеры: join, rebuy, cancel_rebuy
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Создан internal/bot/handlers/hub_callbacks.go с HubCallbackHandler:
- `HandleJoin` — обрабатывает "join:N"; проверяет регистрацию (alert "Сначала нажми /start"); вызывает GameService.Join; при ErrAlreadyJoined → "Ты уже в игре"; при ErrGameNotActive → "Эта игра уже завершена"; при успехе → answer "Ты присоединился!" + updateHub
- `HandleRebuy` — обрабатывает "rebuy:N"; при ErrNotParticipant → "Ты не участник"; при успехе → "Докуп записан!" + updateHub
- `HandleCancelRebuy` — обрабатывает "cancel_rebuy:N"; при успехе → "Докуп отменён" + updateHub
- `updateHub` — строит playerMap для всех участников + создателя, рендерит RenderHub, вызывает EditMessageText с HubKeyboard
- `parseGameIDFromCallback` — извлекает gameID из "action:N" формата
Зарегистрированы в bot.go через RegisterHandlerMatchFunc по HasPrefix ("join:", "rebuy:", "cancel_rebuy:").
go vet чист, go test ./... проходит.
**Следующий шаг:** TASK-029 (finish callback) — разблокирован. TASK-027 (/game команда в личном чате) — также разблокирован.

---

### [TASK-029] Callback хендлер «finish»: подтверждение и переход к сбору результатов
**Дата:** 2026-04-11
**Статус:** done
**Summary:** Реализован `HandleFinish` в `internal/bot/handlers/hub_callbacks.go`:
- Two-tap confirmation: первый тап сохраняет `finish_confirm_game_id` и `finish_confirm_time` в FSM Data и показывает alert "Точно завершить игру? ..."; второй тап в течение 30 секунд вызывает `GameService.FinishGame`
- После FinishGame: `updateHub` обновляет хаб в группе (статус переходит в "сбор результатов", ⏳ рядом с каждым участником)
- `sendCollectResultsMessages`: рассылает личные сообщения всем участникам "🏁 Игра #N завершена! Введи свои финальные данные — напиши /game"
- Обновлён `NewHubCallbackHandler` — добавлен параметр `*fsm.Store`
- Зарегистрирован в `bot.go` через `HasPrefix("finish:")`
- go vet чист, go test ./... проходит
**Следующий шаг:** TASK-030 (View и FSM: личное сообщение для сбора финальных данных участника) — разблокирован. Параллельно: TASK-033 (SettlementService.Validate), TASK-036 (персональный результат view), TASK-012 (SettlementRepository), TASK-018 (/name), TASK-027 (/game).

---

### [TASK-030] View и FSM: личное сообщение для сбора финальных данных
**Дата:** 2026-04-13
**Статус:** done
**Summary:**
- `internal/service/game_service.go`: добавлены `GetGameByID`, `GetParticipant`, `AdjustRebuyInCollection(delta ±1)` — работает только при статусе `collecting_results`, возвращает обновлённый Participant
- `internal/bot/keyboards/keyboards.go`: добавлены `ChipsCollectionKeyboard(gameID)` (ряд ➖/➕ + ряд chips/rubles с embedded gameID в формате `chips_mode:<mode>:<id>`) и `ResultConfirmKeyboard(gameID)`
- `internal/bot/views/collect_results.go`: `RenderChipsInput` (бай-ин, докупы, итого), `RenderChipsConfirm` (докупов / осталось / результат ±)
- `internal/bot/handlers/collect_results.go`: `CollectResultsHandler` — `HandleRebuyPlus/Minus`, `HandleChipsMode` (FSM→StateAwaitingChipsInput, сохраняет game_id/mode/msg_id), `HandleChipsText` (парсинг >= 0, показывает confirm + ResultConfirmKeyboard, сохраняет chips в FSM для TASK-031)
- `internal/bot/bot.go`: зарегистрированы `collect_rebuy_plus:*`, `collect_rebuy_minus:*`, `chips_mode:*`, текст при `StateAwaitingChipsInput`
- `collect_results_test.go`: 4 unit-теста для view (profit/loss/break-even/zero-rebuys)
- go vet чист, go test ./... проходит
**Следующий шаг:** TASK-031 (подтверждение финальных данных: `SubmitResult`, обработчики `confirm_result`/`edit_result`) — разблокирован. Параллельно: TASK-033 (SettlementService.Validate), TASK-036 (personal result view).

---

### [TASK-031] Подтверждение финальных данных участника и обновление хаба
**Дата:** 2026-04-13
**Статус:** done
**Summary:**
- `internal/service/game_service.go`: добавлен `SubmitResult(ctx, gameID, playerID, finalChips)` — транзакционно сохраняет `final_chips` и `results_confirmed=true`; идемпотентен (повторный вызов возвращает текущее состояние без ошибки)
- `internal/bot/handlers/collect_results.go`: добавлены `HandleConfirmResult` (обрабатывает `confirm_result:N`, берёт chips из FSM Data, вызывает `SubmitResult`, редактирует личное сообщение, обновляет хаб в группе) и `HandleEditResult` (обрабатывает `edit_result:N`, возвращает к ChipsCollectionKeyboard)
- `updateHubAfterConfirm` — приватный метод `CollectResultsHandler`, дублирует логику `HubCallbackHandler.updateHub` (зависимость на `players` уже была в struct)
- `internal/bot/bot.go`: зарегистрированы `confirm_result:*` и `edit_result:*`
- `game_service_test.go`: 3 теста — Success, Idempotent (chips не меняются при повторном confirm), ErrGameNotActive
- go vet чист, go test ./... проходит
**Следующий шаг:** TASK-033 (SettlementService.Validate) и TASK-036 (View персонального результата) — оба critical и разблокированы.

---

### [TASK-033] SettlementService.Validate: проверка сходимости банка
**Дата:** 2026-04-13
**Статус:** done
**Summary:**
- `internal/domain/errors.go`: `ErrBankMismatch` sentinel остался для `errors.Is`; добавлен custom type `BankMismatchError{Expected, Actual, Diff int64}` с методом `Error()`
- `internal/service/settlement_service.go`: добавлен `Validate(participants []Participant, buyIn int64) error` — если не все `ResultsConfirmed`, возвращает nil (deferred); иначе сравнивает Σbuy_in*(1+rebuy) с Σfinal_chips; при расхождении возвращает `*BankMismatchError`; добавлен хелпер `IsBankMismatch(err) (*BankMismatchError, bool)` через `errors.As`
- `internal/bot/handlers/collect_results.go`: `CollectResultsHandler` получил поле `settlements *service.SettlementService`; в `HandleConfirmResult` после обновления хаба вызывается `Validate`; при `BankMismatchError` — `SendMessage` в групповой чат с ⚠️ и цифрами расхождения
- `internal/bot/bot.go`: `Deps` расширен полем `Settlements *service.SettlementService`
- `cmd/bot/main.go`: создаётся `settlementSvc` и передаётся в `Deps`
- `settlement_service_test.go`: добавлены 4 теста Validate (match, mismatch+поля, deferred, with rebuys)
- go vet чист, go test ./... проходит
**Следующий шаг:** TASK-036 (View персонального результата с реквизитами) и TASK-012 (SettlementRepository) — оба разблокированы. TASK-035 разблокируется после TASK-012.

---

### [TASK-036] View: персональное сообщение с результатом
**Дата:** 2026-04-13
**Статус:** done
**Summary:**
- `internal/bot/views/personal_result.go`: `RenderPersonalResult(gameID, playerID int64, transfers []domain.Transfer, players map[int64]*domain.Player) string`
  - Проигравший (balance < 0): 📉, «Тебе нужно перевести» с именем, суммой, `<code>телефон</code>` и банком получателя
  - Выигравший (balance > 0): 🎉, «Тебе должны перевести» только с именами (без реквизитов должников)
  - Баланс 0 (нет переводов): «🤝 Ты остался при своих. Никому ничего не должен»
  - Parse mode HTML, суммы в формате «X ₽»
- `internal/bot/views/personal_result_test.go`: 5 тестов — Loser (phone+bank в `<code>`), Winner (нет телефонов должников), BreakEven (сообщение «остался при своих»), MultipleTransfers (список нескольких кредиторов + итоговая сумма), HTMLParseMode (`<b>` и `<code>` теги)
- go vet чист, go test ./... проходит (15/15 в views)
**Следующий шаг:** TASK-012 (SettlementRepository: SaveAll + ListByGame) — разблокирует TASK-035.

---

### [TASK-012] SettlementRepository: SaveAll и ListByGame
**Дата:** 2026-04-13
**Статус:** done
**Summary:**
- `internal/service/interfaces.go`: добавлен интерфейс `SettlementRepository` с `SaveAll(ctx, gameID, []Transfer) error` и `ListByGame(ctx, gameID) ([]Settlement, error)`
- `internal/storage/settlement_repo.go`: `SettlementRepo` реализует интерфейс:
  - `SaveAll` — пустой слайс/nil = no-op; итерирует и вставляет через `extractDB(ctx)` (участвует в транзакции если есть)
  - `ListByGame` — возвращает все переводы для игры
- `internal/storage/settlement_repo_test.go`: 4 теста (SaveAll+ListByGame, пустой слайс, транзакционный rollback, пустой ListByGame)
- go vet чист, go test ./... проходит
**Следующий шаг:** TASK-035 (оркестрация расчёта) — теперь разблокирован (все зависимости done).

---

### [TASK-035] Оркестрация финального расчёта
**Дата:** 2026-04-13
**Статус:** done
**Summary:**
- `internal/service/game_service.go`: добавлено поле `settlements SettlementRepository` в `GameService`, обновлён `NewGameService` (4 аргумента), добавлен метод `FinalizeGame(ctx, gameID, transfers)` — в транзакции: `SaveAll`, `UpdateStatus(Finished)`, `SetFinishedAt(now)`, возвращает обновлённую Game
- `internal/service/game_service_test.go`: все вызовы `NewGameService` обновлены (добавлен `storage.NewSettlementRepo(db)`)
- `internal/bot/views/game_summary.go`: новый view `RenderGameSummary` — заголовок с длительностью и банком, список участников с медалями (🥇🥈🥉/❌) отсортированный по результату, секция «💸 Переводы:», HTML parse mode
- `internal/bot/handlers/collect_results.go`: `HandleConfirmResult` расширен — после всех подтверждений: Validate → BankMismatch warn → Compute → FinalizeGame → персональные сообщения каждому участнику → групповая сводка; добавлен `buildPlayerMap` хелпер
- `cmd/bot/main.go`: добавлен `settlementRepo := storage.NewSettlementRepo(db)`, передан в `NewGameService`
- go vet чист, go test ./... все проходят
**Следующий шаг:** TASK-038 (end-to-end интеграция) — зависит от TASK-035, TASK-036 (done), TASK-037 (pending). TASK-037 (итоговая сводка view) и TASK-026 (rate-limited hub updater) — следующие по приоритету.

---

### [TASK-037] View: итоговая сводка результатов для публикации в групповом чате
**Дата:** 2026-04-13
**Статус:** done
**Summary:**
- `internal/bot/views/game_summary.go`: функция `RenderGameSummary` уже была реализована в TASK-035. Содержит: заголовок 🎰 с ID игры, длительность (Xч Yмин), банк, число игроков; список участников отсортированный по убыванию баланса с медалями 🥇🥈🥉 для первых трёх положительных и ❌ для проигравших; секция 💸 Переводы; HTML parse mode.
- `internal/bot/views/game_summary_test.go`: 5 новых тестов — Medals (6 игроков: 3 медали + 3 ❌, порядок сортировки), Duration (3ч 20мин), Transfers (→ стрелка, имена, сумма), HTMLParseMode (<b> теги), Header (эмодзи 🎰, ID, банк).
- go vet чист, go test ./... все проходят (19 тестов в views)
**Следующий шаг:** TASK-038 (end-to-end интеграция) — все зависимости теперь done. TASK-026 (rate-limited hub updater) параллельно.
