package handlers

import (
	"fmt"

	"education/internal/auth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// processLoginMessage обрабатывает этапы логина:
/*
   1) Пользователь ввёл код → LoginStateWaitingForRegCode
   2) Пользователь ввёл пароль → LoginStateWaitingForPassword
*/
func processLoginMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI, state, text string) {
	chatID := update.Message.Chat.ID

	// Храним временные данные логина в loginTempDataMap
	ld, ok := loginTempDataMap[chatID]
	if !ok {
		ld = &loginData{}
		loginTempDataMap[chatID] = ld
	}

	switch state {
	case LoginStateWaitingForRegCode:
		// Пользователь вводит код (ST-456 и т.п.)
		ld.RegCode = text
		loginStates[chatID] = LoginStateWaitingForPassword

		bot.Send(tgbotapi.NewMessage(chatID, "🔑 Введите ваш пароль:"))
		return

	case LoginStateWaitingForPassword:
		// Пользователь вводит пароль
		regCode := ld.RegCode

		// Ищем запись пользователя в базе по коду
		user, err := auth.GetUserByRegCode(regCode)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка чтения из БД. Попробуйте позже."))
			return
		}
		if user == nil {
			// Не нашли запись с таким кодом
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Пользователь с таким пропуском не найден."))
			return
		}

		// Сверяем пароль
		if user.Password != text {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Неверный пароль. Попробуйте ещё раз."))
			return
		}

		// Если всё хорошо, «привязываем» этого пользователя к текущему чату (telegram_id)
		user.TelegramID = chatID
		if err := auth.SaveUser(user); err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка сохранения пользователя. Попробуйте позже."))
			return
		}

		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🎉 Вход выполнен успешно! Добро пожаловать, %s", user.Name)))

		// Сбрасываем состояние логина
		delete(loginStates, chatID)
		delete(loginTempDataMap, chatID)
		return
	}
}
