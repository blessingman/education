package handlers

import (
	"education/internal/auth"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ProcessMessage — основной обработчик входящих сообщений.
// Делегирует регистрацию и вход, проверяет "в системе" через базу.
// ProcessMessage — основной обработчик входящих сообщений.
// 1) Проверяет, не находимся ли мы в процессе входа/регистрации (loginStates / userStates).
// 2) Если пользователь уже "в системе" (user != nil), обрабатывает /logout.
// 3) Если пользователь не в системе, обрабатывает /register /login.
func ProcessMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// A. Проверка: не в процессе ли логина?
	if state, ok := loginStates[chatID]; ok {
		// Разрешаем только /cancel
		if update.Message.IsCommand() {
			if update.Message.Command() == "cancel" {
				delete(loginStates, chatID)
				delete(loginTempDataMap, chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Процесс входа отменён."))
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "Вы уже в процессе входа. Используйте /cancel, чтобы отменить."))
			}
			return
		}
		processLoginMessage(update, bot, state, text)
		return
	}

	// B. Проверка: не в процессе ли регистрации?
	if state, ok := userStates[chatID]; ok {
		// Разрешаем только /cancel
		if update.Message.IsCommand() {
			if update.Message.Command() == "cancel" {
				delete(userStates, chatID)
				delete(userTempDataMap, chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Процесс регистрации отменён."))
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "Вы уже в процессе регистрации. Используйте /cancel, чтобы отменить."))
			}
			return
		}
		processRegistrationMessage(update, bot, state, text)
		return
	}

	// C. Проверка: пользователь "в системе"?
	user, err := auth.GetUserByTelegramID(chatID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка чтения пользователя из базы. Попробуйте позже."))
		return
	}

	if user != nil {
		// Уже в системе
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "register":
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы уже зарегистрированы. Используйте /logout, чтобы выйти и зарегистрироваться заново."))
			case "login":
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы уже вошли в систему. Используйте /logout, чтобы выйти."))
			case "logout":
				// Вышли → telegram_id=0
				user.TelegramID = 0
				_ = auth.SaveUser(user) // при желании обработайте ошибку
				bot.Send(tgbotapi.NewMessage(chatID, "✅ Вы успешно вышли из системы."))
			case "cancel":
				bot.Send(tgbotapi.NewMessage(chatID, "ℹ Нечего отменять, вы уже в системе. Используйте /logout, если хотите выйти."))
			default:
				bot.Send(tgbotapi.NewMessage(chatID, "🤷 Команда не распознана или ещё не реализована."))
			}
		} else {
			// не команда → можем подсказать
			bot.Send(tgbotapi.NewMessage(chatID, "ℹ Вы уже в системе. Используйте /logout, чтобы выйти, или другие команды."))
		}
		return
	}

	// D. Пользователь не в системе
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

// ProcessCallback — обрабатывает callback‑запросы инлайн-кнопок (выбор факультета/группы).
func ProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	RegistrationProcessCallback(callback, bot)
}
