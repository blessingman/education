package handlers

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ProcessMessage — основной обработчик входящих сообщений.
// Делегирует обработку регистрации и входа соответствующим функциям.
func ProcessMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Если пользователь находится в процессе входа.
	if state, ok := loginStates[chatID]; ok {
		processLoginMessage(update, bot, state, text)
		return
	}

	// Если пользователь находится в процессе регистрации.
	if state, ok := userStates[chatID]; ok {
		processRegistrationMessage(update, bot, state, text)
		return
	}

	// Обработка команд.
	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			bot.Send(tgbotapi.NewMessage(chatID, "Привет! Используй /register для регистрации или /login для входа."))
		case "register":
			userStates[chatID] = StateWaitingForFaculty
			userTempDataMap[chatID] = &tempUserData{}
			sendFacultySelection(chatID, bot)
		case "login":
			loginStates[chatID] = LoginStateWaitingForRegCode
			loginTempDataMap[chatID] = &loginData{}
			bot.Send(tgbotapi.NewMessage(chatID, "Введите ваш пропуск (регистрационный код):"))
		default:
			bot.Send(tgbotapi.NewMessage(chatID, "Команда не распознана или ещё не реализована."))
		}
	} else {
		bot.Send(tgbotapi.NewMessage(chatID, "Для начала используйте /register или /login"))
	}
}

// ProcessCallback — основной обработчик callback‑запросов.
// Делегирует обработку callback-ов функции RegistrationProcessCallback.
func ProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	RegistrationProcessCallback(callback, bot)
}
