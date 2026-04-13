package middleware

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const cacheTTL = 10 * time.Minute

// memberCache is an in-memory cache of user membership checks with TTL.
type memberCache struct {
	mu      sync.Mutex
	entries map[int64]time.Time // userID -> expiry
}

func newMemberCache() *memberCache {
	return &memberCache{
		entries: make(map[int64]time.Time),
	}
}

func (c *memberCache) isAllowed(userID int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp, ok := c.entries[userID]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(c.entries, userID)
		return false
	}
	return true
}

func (c *memberCache) allow(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[userID] = time.Now().Add(cacheTTL)
}

// Auth is the access control middleware. Group chat updates are only
// accepted from allowedChatID. Private chat updates are accepted only
// from users who are members of allowedChatID (checked via getChatMember
// and cached in memory for cacheTTL).
type Auth struct {
	allowedChatID int64
	cache         *memberCache
}

// NewAuth creates a new Auth middleware for the given allowed group chat ID.
func NewAuth(allowedChatID int64) *Auth {
	return &Auth{
		allowedChatID: allowedChatID,
		cache:         newMemberCache(),
	}
}

// Middleware returns a bot.Middleware that enforces access control.
func (a *Auth) Middleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, chatType, userID := extractChatInfo(update)
		if userID == 0 {
			// Cannot determine user; let it through (e.g. channel posts).
			next(ctx, b, update)
			return
		}

		allowed := a.isAllowed(ctx, b, chatID, chatType, userID)

		if !allowed {
			if chatID != 0 {
				_, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   "Этот бот работает только в закрытой группе. Обратись к владельцу.",
				})
				if err != nil {
					slog.Error("auth: failed to send rejection message", "err", err)
				}
			}
			return
		}

		next(ctx, b, update)
	}
}

func (a *Auth) isAllowed(ctx context.Context, b *bot.Bot, chatID int64, chatType models.ChatType, userID int64) bool {
	switch chatType {
	case models.ChatTypePrivate:
		if a.cache.isAllowed(userID) {
			return true
		}
		member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{
			ChatID: chatID,
			UserID: userID,
		})
		if err != nil {
			slog.Error("auth: getChatMember failed", "userID", userID, "err", err)
			return false
		}
		switch member.Type {
		case models.ChatMemberTypeOwner, models.ChatMemberTypeAdministrator, models.ChatMemberTypeMember:
			a.cache.allow(userID)
			return true
		case models.ChatMemberTypeRestricted:
			if member.Restricted != nil && member.Restricted.IsMember {
				a.cache.allow(userID)
				return true
			}
		}
		return false
	case models.ChatTypeGroup, models.ChatTypeSupergroup:
		return chatID == a.allowedChatID
	default:
		return false
	}
}

// extractChatInfo extracts chatID, chatType and userID from an update.
// Returns zero values when the information is unavailable.
func extractChatInfo(update *models.Update) (chatID int64, chatType models.ChatType, userID int64) {
	if update.Message != nil {
		chatID = update.Message.Chat.ID
		chatType = update.Message.Chat.Type
		if update.Message.From != nil {
			userID = update.Message.From.ID
		}
		return
	}
	if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
		msg := update.CallbackQuery.Message
		if msg.Message != nil {
			chatID = msg.Message.Chat.ID
			chatType = msg.Message.Chat.Type
		} else if msg.InaccessibleMessage != nil {
			chatID = msg.InaccessibleMessage.Chat.ID
			chatType = msg.InaccessibleMessage.Chat.Type
		}
		return
	}
	return
}
