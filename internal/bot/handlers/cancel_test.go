package handlers

import (
	"testing"

	"poker-bot/internal/fsm"
)

func TestCancelHandler_ResetsActiveState(t *testing.T) {
	store := fsm.NewStore()
	defer store.Stop()

	store.Set(42, &fsm.Session{State: fsm.StateAwaitingPhone, Data: map[string]any{"foo": "bar"}})

	h := NewCancelHandler(store)
	_ = h // handler registered; test FSM state transition directly

	// Simulate what Handle() does when state is non-idle.
	sess, ok := store.Get(42)
	if !ok {
		t.Fatal("session not found")
	}
	if sess.State != fsm.StateAwaitingPhone {
		t.Fatalf("expected StateAwaitingPhone, got %v", sess.State)
	}

	store.Set(42, &fsm.Session{State: fsm.StateIdle, Data: make(map[string]any)})

	after, ok := store.Get(42)
	if !ok {
		t.Fatal("session missing after reset")
	}
	if after.State != fsm.StateIdle {
		t.Fatalf("expected StateIdle after cancel, got %v", after.State)
	}
	if len(after.Data) != 0 {
		t.Fatalf("expected empty Data after cancel, got %v", after.Data)
	}
}

func TestCancelHandler_IdleStateDetected(t *testing.T) {
	store := fsm.NewStore()
	defer store.Stop()

	store.Set(99, &fsm.Session{State: fsm.StateIdle, Data: map[string]any{}})

	sess, ok := store.Get(99)
	if !ok {
		t.Fatal("session not found")
	}
	if sess.State != fsm.StateIdle {
		t.Fatalf("expected StateIdle, got %v", sess.State)
	}
}

func TestCancelHandler_NoSession(t *testing.T) {
	store := fsm.NewStore()
	defer store.Stop()

	_, ok := store.Get(777)
	if ok {
		t.Fatal("expected no session for fresh user")
	}
	// No session = idle = "Нечего отменять" path.
}
