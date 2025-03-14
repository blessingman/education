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
// processLoginMessage обрабатывает этапы логина:
/*
   1) Пользователь ввёл код → LoginStateWaitingForRegCode
   2) Пользователь ввёл пароль → LoginStateWaitingForPassword
*/
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
		// Пользователь вводит код (например, ST-456)
		ld.RegCode = text
		loginStates[chatID] = LoginStateWaitingForPassword

		msg := tgbotapi.NewMessage(chatID, "🔑 Введите ваш пароль:")
		sendAndTrackMessage(bot, msg)
		return

	case LoginStateWaitingForPassword:
		// Пользователь вводит пароль
		regCode := ld.RegCode

		// Ищем пользователя в БД по коду
		user, err := auth.GetUserByRegCode(regCode)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка чтения из БД. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if user == nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Пользователь с таким пропуском не найден.")
			sendAndTrackMessage(bot, msg)
			return
		}

		// Сверяем пароль
		if user.Password != text {
			msg := tgbotapi.NewMessage(chatID, "❌ Неверный пароль. Попробуйте ещё раз.")
			sendAndTrackMessage(bot, msg)
			return
		}

		// Привязываем пользователя к текущему чату (устанавливаем telegram_id)
		user.TelegramID = chatID
		if err := auth.SaveUser(user); err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка сохранения пользователя. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🎉 Вход выполнен успешно! Добро пожаловать, %s", user.Name))
		sendAndTrackMessage(bot, msg)

		// Удаляем все сообщения из чата
		deleteMessages(chatID, bot, 6)

		// Показываем главное меню
		sendMainMenu(chatID, bot, user)

		// Сбрасываем логин-состояния
		delete(loginStates, chatID)
		delete(loginTempDataMap, chatID)
		return
	}
}
