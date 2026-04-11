package fsm_test

import (
	"sync"
	"testing"
	"time"

	"poker-bot/internal/fsm"
)

func TestSetAndGet(t *testing.T) {
	s := fsm.NewStore()
	defer s.Stop()

	sess := &fsm.Session{
		State: fsm.StateAwaitingPhone,
		Data:  map[string]any{"phone": "+79991234567"},
	}
	s.Set(1, sess)

	got, ok := s.Get(1)
	if !ok {
		t.Fatal("expected session to exist")
	}
	if got.State != fsm.StateAwaitingPhone {
		t.Fatalf("expected state %v, got %v", fsm.StateAwaitingPhone, got.State)
	}
	if got.Data["phone"] != "+79991234567" {
		t.Fatalf("unexpected phone: %v", got.Data["phone"])
	}
}

func TestClear(t *testing.T) {
	s := fsm.NewStore()
	defer s.Stop()

	s.Set(1, &fsm.Session{State: fsm.StateIdle, Data: map[string]any{}})
	s.Clear(1)

	got, ok := s.Get(1)
	if ok || got != nil {
		t.Fatal("expected session to be cleared")
	}
}

func TestGetMissing(t *testing.T) {
	s := fsm.NewStore()
	defer s.Stop()

	got, ok := s.Get(999)
	if ok || got != nil {
		t.Fatal("expected no session for unknown user")
	}
}

func TestUpdatedAtIsSet(t *testing.T) {
	s := fsm.NewStore()
	defer s.Stop()

	before := time.Now()
	s.Set(1, &fsm.Session{State: fsm.StateIdle, Data: map[string]any{}})
	after := time.Now()

	got, _ := s.Get(1)
	if got.UpdatedAt.Before(before) || got.UpdatedAt.After(after) {
		t.Fatalf("UpdatedAt %v not in range [%v, %v]", got.UpdatedAt, before, after)
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := fsm.NewStore()
	defer s.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		userID := int64(i % 10)
		go func() {
			defer wg.Done()
			s.Set(userID, &fsm.Session{State: fsm.StateIdle, Data: map[string]any{}})
		}()
		go func() {
			defer wg.Done()
			s.Get(userID)
		}()
	}
	wg.Wait()
}
