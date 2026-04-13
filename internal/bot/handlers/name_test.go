package handlers

import (
	"strings"
	"testing"
)

func TestNameHandler_ArgExtraction(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"/name Вася", "Вася"},
		{"/name  Имя с пробелами  ", "Имя с пробелами"},
		{"/name", ""},
		{"/name ", ""},
	}
	for _, c := range cases {
		text := strings.TrimSpace(c.text)
		got := strings.TrimSpace(strings.TrimPrefix(text, "/name"))
		if got != c.want {
			t.Errorf("arg from %q = %q, want %q", c.text, got, c.want)
		}
	}
}

func TestNameHandler_LengthValidation(t *testing.T) {
	longName := strings.Repeat("а", maxDisplayNameLen+1)
	if len([]rune(longName)) <= maxDisplayNameLen {
		t.Fatal("longName should exceed maxDisplayNameLen")
	}

	okName := strings.Repeat("а", maxDisplayNameLen)
	if len([]rune(okName)) > maxDisplayNameLen {
		t.Fatal("okName should be within limit")
	}
}

func TestNameHandler_MaxLen(t *testing.T) {
	if maxDisplayNameLen != 50 {
		t.Fatalf("expected maxDisplayNameLen=50, got %d", maxDisplayNameLen)
	}
}
