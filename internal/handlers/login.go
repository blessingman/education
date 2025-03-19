package handlers

import (
	"fmt"
	"strings"

	"education/internal/auth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Remove duplicated comments
func processLoginMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI, state, text string) {
	chatID := update.Message.Chat.ID

	// Храним временные данные логина в loginTempDataMap
	ld, ok := loginTempDataMap[chatID]
	if !ok {
		ld = &loginData{}
		loginTempDataMap[chatID] = ld
	}

	// Trim spaces from input
	text = strings.TrimSpace(text)

	switch state {
	case LoginStateWaitingForRegCode:
		// Validate registration code format (more flexible to accept both student and teacher codes)
		if !strings.HasPrefix(text, "ST-") && !strings.HasPrefix(text, "TH-") {
			msg := tgbotapi.NewMessage(chatID, "❌ Некорректный формат кода. Примеры: ST-4056, TR-1203")
			sendAndTrackMessage(bot, msg)
			return
		}

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

		// Проверяем, не авторизован ли уже пользователь в другом чате
		if user.TelegramID != 0 && user.TelegramID != chatID {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Этот аккаунт уже авторизован в другом чате.")
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

		// Показываем главное меню с данными пользователя без удаления сообщений
		sendMainMenu(chatID, bot, user)

		// Сбрасываем логин-состояния
		delete(loginStates, chatID)
		delete(loginTempDataMap, chatID)
		return
	}
}
