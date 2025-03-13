package handlers

import (
	"education/internal/models"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ProcessMessage — основной обработчик входящих сообщений.
// Делегирует обработку регистрации и входа соответствующим функциям.
// ProcessMessage — основной обработчик текстовых сообщений.
func ProcessMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// 1. Проверяем, не в процессе ли пользователь (регистрация или вход)
	if state, ok := loginStates[chatID]; ok {
		if update.Message.IsCommand() {
			if update.Message.Command() == "cancel" {
				delete(loginStates, chatID)
				delete(loginTempDataMap, chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "✅ Процесс входа отменён."))
				return
			}
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы уже в процессе входа. Завершите его или введите /cancel, чтобы отменить."))
			return
		}
		processLoginMessage(update, bot, state, text)
		return
	}

	if state, ok := userStates[chatID]; ok {
		if update.Message.IsCommand() {
			if update.Message.Command() == "cancel" {
				delete(userStates, chatID)
				delete(userTempDataMap, chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "✅ Процесс регистрации отменён."))
				return
			}
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы уже в процессе регистрации. Завершите его или введите /cancel, чтобы отменить."))
			return
		}
		processRegistrationMessage(update, bot, state, text)
		return
	}

	// 2. Проверяем, зарегистрирован ли пользователь
	if _, registered := models.UsersMap[chatID]; registered {
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "register":
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы уже зарегистрированы. Используйте /logout, чтобы выйти и зарегистрироваться заново."))
				return
			case "login":
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы уже вошли в систему. Используйте /logout, чтобы выйти."))
				return
			case "logout":
				delete(models.UsersMap, chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "✅ Вы вышли из системы. Теперь можно снова выполнить /register или /login."))
				return
			}
		}
	}

	// 3. Если пользователь не зарегистрирован, обрабатываем команды как обычно
	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			bot.Send(tgbotapi.NewMessage(chatID, "👋 Привет! Используй /register для регистрации или /login для входа."))
		case "register":
			userStates[chatID] = StateWaitingForFaculty
			userTempDataMap[chatID] = &tempUserData{}
			sendFacultySelection(chatID, bot)
		case "login":
			loginStates[chatID] = LoginStateWaitingForRegCode
			loginTempDataMap[chatID] = &loginData{}
			bot.Send(tgbotapi.NewMessage(chatID, "🔑 Введите ваш пропуск (регистрационный код):"))
		case "logout":
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы не вошли в систему. Воспользуйтесь /register или /login."))
		default:
			bot.Send(tgbotapi.NewMessage(chatID, "🤷 Команда не распознана или ещё не реализована."))
		}
	} else {
		bot.Send(tgbotapi.NewMessage(chatID, "ℹ Для начала используйте /register или /login"))
	}
}

// ProcessCallback — основной обработчик callback‑запросов.
// Делегирует обработку callback-ов функции RegistrationProcessCallback.
func ProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	RegistrationProcessCallback(callback, bot)
}
