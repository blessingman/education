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

	// ----------------------
	// 1. Если пользователь в процессе входа
	// ----------------------
	if state, ok := loginStates[chatID]; ok {
		// Разрешаем только /cancel
		if update.Message.IsCommand() {
			if update.Message.Command() == "cancel" {
				delete(loginStates, chatID)
				delete(loginTempDataMap, chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Процесс входа отменён."))
			} else {
				// Любая другая команда — запрещаем
				bot.Send(tgbotapi.NewMessage(chatID, "Вы уже в процессе входа. Используйте /cancel, чтобы отменить."))
			}
			return
		}
		// Если это не команда, продолжаем процесс входа
		processLoginMessage(update, bot, state, text)
		return
	}

	// ----------------------
	// 2. Если пользователь в процессе регистрации
	// ----------------------
	if state, ok := userStates[chatID]; ok {
		// Разрешаем только /cancel
		if update.Message.IsCommand() {
			if update.Message.Command() == "cancel" {
				delete(userStates, chatID)
				delete(userTempDataMap, chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Процесс регистрации отменён."))
			} else {
				// Любая другая команда — запрещаем
				bot.Send(tgbotapi.NewMessage(chatID, "Вы уже в процессе регистрации. Используйте /cancel, чтобы отменить."))
			}
			return
		}
		// Если это не команда, продолжаем процесс регистрации
		processRegistrationMessage(update, bot, state, text)
		return
	}

	// ----------------------
	// 3. Если пользователь уже в системе (зарегистрирован и вошёл)
	// ----------------------
	if _, registered := models.UsersMap[chatID]; registered {
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "register":
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы уже зарегистрированы. Используйте /logout, чтобы выйти и зарегистрироваться заново."))
			case "login":
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы уже вошли в систему. Используйте /logout, чтобы выйти."))
			case "logout":
				delete(models.UsersMap, chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "✅ Вы успешно вышли из системы."))
			case "cancel":
				bot.Send(tgbotapi.NewMessage(chatID, "ℹ Нечего отменять, вы уже в системе. Используйте /logout, если хотите выйти."))
			default:
				bot.Send(tgbotapi.NewMessage(chatID, "🤷 Команда не распознана или ещё не реализована."))
			}
		} else {
			// Любые некомандные сообщения можно либо игнорировать, либо дать подсказку
			bot.Send(tgbotapi.NewMessage(chatID, "ℹ Вы уже в системе. Используйте /logout, чтобы выйти, или другие команды."))
		}
		return
	}

	// ----------------------
	// 4. Если пользователь не в процессе и не в системе
	// ----------------------
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
		case "cancel":
			bot.Send(tgbotapi.NewMessage(chatID, "ℹ Нечего отменять, вы не в процессе."))
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
