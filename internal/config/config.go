package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken      string
	DBPath        string
	LogPath       string
	AllowedChatID int64
	AdminUserIDs  []int64
}

func Load() (*Config, error) {
	// Load .env if it exists; ignore error if file is missing
	_ = godotenv.Load()

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required but not set")
	}

	allowedChatIDStr := os.Getenv("ALLOWED_CHAT_ID")
	if allowedChatIDStr == "" {
		return nil, fmt.Errorf("ALLOWED_CHAT_ID is required but not set")
	}
	allowedChatID, err := strconv.ParseInt(allowedChatIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("ALLOWED_CHAT_ID must be a valid int64: %w", err)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./poker.db"
	}

	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "./bot.log"
	}

	var adminUserIDs []int64
	if raw := os.Getenv("ADMIN_USER_IDS"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("ADMIN_USER_IDS contains invalid int64 %q: %w", part, err)
			}
			adminUserIDs = append(adminUserIDs, id)
		}
	}

	return &Config{
		BotToken:      botToken,
		DBPath:        dbPath,
		LogPath:       logPath,
		AllowedChatID: allowedChatID,
		AdminUserIDs:  adminUserIDs,
	}, nil
}
