CREATE TABLE IF NOT EXISTS players (
    telegram_id      INTEGER PRIMARY KEY,
    telegram_username TEXT,
    display_name     TEXT NOT NULL,
    phone_number     TEXT NOT NULL,
    bank_name        TEXT NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at       TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS games (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id        INTEGER NOT NULL,
    creator_id     INTEGER REFERENCES players(telegram_id) ON DELETE RESTRICT,
    buy_in         INTEGER NOT NULL,
    hub_message_id INTEGER,
    status         TEXT NOT NULL DEFAULT 'active',
    created_at     TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    finished_at    TIMESTAMP
);

CREATE TABLE IF NOT EXISTS game_participants (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id            INTEGER NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    player_id          INTEGER NOT NULL REFERENCES players(telegram_id) ON DELETE RESTRICT,
    rebuy_count        INTEGER NOT NULL DEFAULT 0,
    final_chips        INTEGER,
    results_confirmed  BOOLEAN NOT NULL DEFAULT 0,
    joined_at          TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    UNIQUE(game_id, player_id)
);

CREATE TABLE IF NOT EXISTS settlements (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id        INTEGER NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    from_player_id INTEGER NOT NULL REFERENCES players(telegram_id) ON DELETE RESTRICT,
    to_player_id   INTEGER NOT NULL REFERENCES players(telegram_id) ON DELETE RESTRICT,
    amount         INTEGER NOT NULL
);
