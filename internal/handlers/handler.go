package handlers

import (
	"education/internal/models"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ProcessMessage — основной обработчик входящих сообщений.
// Делегирует обработку регистрации и входа соответствующим функциям.
// Также добавлена логика проверки: если пользователь уже зарегистрирован или залогинен, команды /register и /login выдадут сообщение.
func ProcessMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Если пользователь находится в процессе входа.
	if state, ok := loginStates[chatID]; ok {
		// Если введена команда /cancel, можно сбросить процесс.
		if update.Message.IsCommand() && update.Message.Command() == "cancel" {
			delete(loginStates, chatID)
			delete(loginTempDataMap, chatID)
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Процесс входа отменён."))
			return
		}
		processLoginMessage(update, bot, state, text)
		return
	}

	// Если пользователь находится в процессе регистрации.
	if state, ok := userStates[chatID]; ok {
		if update.Message.IsCommand() && update.Message.Command() == "cancel" {
			delete(userStates, chatID)
			delete(userTempDataMap, chatID)
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Процесс регистрации отменён."))
			return
		}
		processRegistrationMessage(update, bot, state, text)
		return
	}

	// Если пользователь уже зарегистрован (предполагаем, что UsersMap хранит зарегистрированных пользователей по TelegramID).
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
				// Здесь удаляем пользователя из UsersMap, чтобы "выйти"
				delete(models.UsersMap, chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "✅ Вы успешно вышли из системы."))
				return
			}
		}
	}

	// Обработка команд, если пользователь не в процессе регистрации/входа.
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
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы не вошли в систему."))
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
