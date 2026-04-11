package middleware

import (
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
)

func TestMemberCache(t *testing.T) {
	c := newMemberCache()

	// Initially no entry
	if c.isAllowed(1) {
		t.Fatal("expected user 1 to be denied before any allow")
	}

	c.allow(1)
	if !c.isAllowed(1) {
		t.Fatal("expected user 1 to be allowed after allow()")
	}

	// Manually expire the entry
	c.mu.Lock()
	c.entries[1] = time.Now().Add(-time.Second)
	c.mu.Unlock()

	if c.isAllowed(1) {
		t.Fatal("expected expired entry to be denied")
	}
	// Expired entry should have been removed
	c.mu.Lock()
	_, exists := c.entries[1]
	c.mu.Unlock()
	if exists {
		t.Fatal("expired entry should be deleted from cache")
	}
}

func TestExtractChatInfo_Message(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			Chat: models.Chat{
				ID:   -100123,
				Type: models.ChatTypeSupergroup,
			},
			From: &models.User{ID: 42},
		},
	}

	chatID, chatType, userID := extractChatInfo(update)
	if chatID != -100123 {
		t.Errorf("chatID: got %d, want -100123", chatID)
	}
	if chatType != models.ChatTypeSupergroup {
		t.Errorf("chatType: got %s, want supergroup", chatType)
	}
	if userID != 42 {
		t.Errorf("userID: got %d, want 42", userID)
	}
}

func TestExtractChatInfo_PrivateMessage(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			Chat: models.Chat{
				ID:   99,
				Type: models.ChatTypePrivate,
			},
			From: &models.User{ID: 7},
		},
	}

	chatID, chatType, userID := extractChatInfo(update)
	if chatID != 99 {
		t.Errorf("chatID: got %d, want 99", chatID)
	}
	if chatType != models.ChatTypePrivate {
		t.Errorf("chatType: got %s, want private", chatType)
	}
	if userID != 7 {
		t.Errorf("userID: got %d, want 7", userID)
	}
}

func TestExtractChatInfo_CallbackQuery(t *testing.T) {
	update := &models.Update{
		CallbackQuery: &models.CallbackQuery{
			From: models.User{ID: 55},
			Message: models.MaybeInaccessibleMessage{
				Type: models.MaybeInaccessibleMessageTypeMessage,
				Message: &models.Message{
					Chat: models.Chat{
						ID:   -100456,
						Type: models.ChatTypeSupergroup,
					},
				},
			},
		},
	}

	chatID, chatType, userID := extractChatInfo(update)
	if chatID != -100456 {
		t.Errorf("chatID: got %d, want -100456", chatID)
	}
	if chatType != models.ChatTypeSupergroup {
		t.Errorf("chatType: got %s, want supergroup", chatType)
	}
	if userID != 55 {
		t.Errorf("userID: got %d, want 55", userID)
	}
}

func TestExtractChatInfo_Empty(t *testing.T) {
	update := &models.Update{}
	chatID, _, userID := extractChatInfo(update)
	if chatID != 0 || userID != 0 {
		t.Errorf("expected zero values for empty update, got chatID=%d userID=%d", chatID, userID)
	}
}
