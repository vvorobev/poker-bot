package handlers

import (
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"79991234567", "+79991234567"},
		{"+79991234567", "+79991234567"},
		{" +79991234567 ", "+79991234567"},
		{"7 999 123 45 67", "+7 999 123 45 67"}, // raw passthrough; validation handles format
	}
	for _, c := range cases {
		got := normalizePhone(c.input)
		if got != c.want {
			t.Errorf("normalizePhone(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
