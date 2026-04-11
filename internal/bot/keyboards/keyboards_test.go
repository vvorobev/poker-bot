package keyboards_test

import (
	"strings"
	"testing"

	"poker-bot/internal/bot/keyboards"
)

func TestHubKeyboard_CallbackData(t *testing.T) {
	kb := keyboards.HubKeyboard(42)
	if kb == nil {
		t.Fatal("HubKeyboard returned nil")
	}

	// Flatten all buttons
	var allData []string
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			allData = append(allData, btn.CallbackData)
		}
	}

	expected := []string{"join:42", "rebuy:42", "cancel_rebuy:42", "finish:42"}
	for _, want := range expected {
		found := false
		for _, got := range allData {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected callback_data %q not found in hub keyboard", want)
		}
	}
}

func TestHubKeyboard_ContainsSuffix(t *testing.T) {
	kb := keyboards.HubKeyboard(99)
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			if !strings.HasSuffix(btn.CallbackData, ":99") {
				t.Errorf("button %q callback_data does not end with :99", btn.Text)
			}
		}
	}
}

func TestBankKeyboard_Count(t *testing.T) {
	kb := keyboards.BankKeyboard()
	if kb == nil {
		t.Fatal("BankKeyboard returned nil")
	}

	var count int
	for _, row := range kb.InlineKeyboard {
		count += len(row)
	}
	if count != 8 {
		t.Errorf("expected 8 bank buttons, got %d", count)
	}
}

func TestBankKeyboard_CallbackPrefix(t *testing.T) {
	kb := keyboards.BankKeyboard()
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			if !strings.HasPrefix(btn.CallbackData, "bank:") {
				t.Errorf("bank button %q has unexpected callback_data %q", btn.Text, btn.CallbackData)
			}
		}
	}
}

func TestBuyInKeyboard(t *testing.T) {
	kb := keyboards.BuyInKeyboard()
	if kb == nil {
		t.Fatal("BuyInKeyboard returned nil")
	}
	if len(kb.InlineKeyboard) == 0 || len(kb.InlineKeyboard[0]) == 0 {
		t.Fatal("BuyInKeyboard has no buttons")
	}
	btn := kb.InlineKeyboard[0][0]
	if btn.CallbackData != "buyin:1000" {
		t.Errorf("expected buyin:1000, got %q", btn.CallbackData)
	}
}

func TestChipsInputKeyboard(t *testing.T) {
	kb := keyboards.ChipsInputKeyboard()
	if kb == nil {
		t.Fatal("ChipsInputKeyboard returned nil")
	}
	if len(kb.InlineKeyboard) == 0 || len(kb.InlineKeyboard[0]) != 2 {
		t.Fatal("ChipsInputKeyboard should have 2 buttons in first row")
	}
}
