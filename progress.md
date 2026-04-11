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

