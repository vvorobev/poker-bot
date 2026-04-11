package keyboards

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

// HubKeyboard returns the inline keyboard for the game hub message.
// callback_data format: "action:game_id", e.g. "join:42"
func HubKeyboard(gameID int64) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "➕ Присоединиться", CallbackData: fmt.Sprintf("join:%d", gameID)},
				{Text: "💵 Докупиться", CallbackData: fmt.Sprintf("rebuy:%d", gameID)},
			},
			{
				{Text: "➖ Отменить докуп", CallbackData: fmt.Sprintf("cancel_rebuy:%d", gameID)},
				{Text: "🏁 Завершить игру", CallbackData: fmt.Sprintf("finish:%d", gameID)},
			},
		},
	}
}

// BankKeyboard returns the inline keyboard for bank selection during onboarding.
func BankKeyboard() *models.InlineKeyboardMarkup {
	banks := []string{
		"Тинькофф", "Сбербанк", "Альфа-Банк", "ВТБ",
		"Райффайзен", "Озон Банк", "Яндекс Банк", "Другой",
	}

	rows := make([][]models.InlineKeyboardButton, 0, (len(banks)+1)/2)
	for i := 0; i < len(banks); i += 2 {
		row := []models.InlineKeyboardButton{
			{Text: banks[i], CallbackData: "bank:" + banks[i]},
		}
		if i+1 < len(banks) {
			row = append(row, models.InlineKeyboardButton{
				Text:         banks[i+1],
				CallbackData: "bank:" + banks[i+1],
			})
		}
		rows = append(rows, row)
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// BuyInKeyboard returns the inline keyboard for buy-in selection.
func BuyInKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "1000 ₽ (по умолчанию)", CallbackData: "buyin:1000"},
			},
		},
	}
}

// ChipsInputKeyboard returns the inline keyboard for chips input mode selection.
func ChipsInputKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Ввести в фишках", CallbackData: "chips_mode:chips"},
				{Text: "Ввести в рублях", CallbackData: "chips_mode:rubles"},
			},
		},
	}
}
