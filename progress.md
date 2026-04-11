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

