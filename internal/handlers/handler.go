package handlers

import (
	"education/internal/auth"
	"education/internal/models"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// sendMainMenu отправляет меню, которое зависит от того, авторизован пользователь или нет.
func sendMainMenu(chatID int64, bot *tgbotapi.BotAPI, user *models.User) {
	var rows [][]tgbotapi.InlineKeyboardButton

	if user == nil {
		// Пользователь не авторизован → показываем «Регистрация» и «Вход»
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Регистрация", "menu_register"),
			tgbotapi.NewInlineKeyboardButtonData("Вход", "menu_login"),
		))
	} else {
		// Пользователь авторизован → показываем «Расписание», «Материалы» и «Выход»
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Расписание", "menu_schedule"),
			tgbotapi.NewInlineKeyboardButtonData("Материалы", "menu_materials"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Выход", "menu_logout"),
		))
	}

	// «Справка» доступна всем
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Справка", "menu_help"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, "Выберите действие:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// ProcessMessage — основной обработчик входящих сообщений.
func ProcessMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// A. Проверка: не в процессе ли логина?
	if state, ok := loginStates[chatID]; ok {
		processLoginMessage(update, bot, state, text)
		return
	}

	// B. Проверка: не в процессе ли регистрации?
	if state, ok := userStates[chatID]; ok {
		processRegistrationMessage(update, bot, state, text)
		return
	}

	// C. Проверка: получаем текущего пользователя (если он авторизован)
	user, err := auth.GetUserByTelegramID(chatID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка чтения пользователя из базы. Попробуйте позже."))
		return
	}

	// D. Обработка команд /start и т.п.
	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			sendMainMenu(chatID, bot, user)
			return
		default:
			sendMainMenu(chatID, bot, user)
			return
		}
	} else {
		// Если пришло обычное сообщение, просто показываем меню
		sendMainMenu(chatID, bot, user)
		return
	}
}

// ProcessCallback — обрабатывает callback‑запросы инлайн-кнопок (главное меню + вызов RegistrationProcessCallback).
func ProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID

	switch callback.Data {

	// --- Меню для неавторизованных ---
	case "menu_register":
		userStates[chatID] = StateWaitingForFaculty
		userTempDataMap[chatID] = &tempUserData{}
		bot.Request(tgbotapi.NewCallback(callback.ID, "Переходим к регистрации"))
		sendFacultySelection(chatID, bot)
		return

	case "menu_login":
		loginStates[chatID] = LoginStateWaitingForRegCode
		loginTempDataMap[chatID] = &loginData{}
		bot.Request(tgbotapi.NewCallback(callback.ID, "Переходим к входу"))
		bot.Send(tgbotapi.NewMessage(chatID, "🔑 Введите ваш регистрационный код:"))
		return

	// --- Меню для авторизованных ---
	case "menu_schedule":
		bot.Request(tgbotapi.NewCallback(callback.ID, "Расписание"))
		bot.Send(tgbotapi.NewMessage(chatID, "Ваше расписание: (здесь может быть реальная логика)"))
		// Показываем меню снова
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return

	case "menu_materials":
		bot.Request(tgbotapi.NewCallback(callback.ID, "Материалы"))
		bot.Send(tgbotapi.NewMessage(chatID, "Список материалов: (здесь может быть реальная логика)"))
		// Показываем меню снова
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return

	case "menu_logout":
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Вы не авторизованы"))
		} else {
			user.TelegramID = 0
			_ = auth.SaveUser(user)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Вы вышли из системы"))
			bot.Send(tgbotapi.NewMessage(chatID, "✅ Вы успешно вышли из системы."))
		}
		// После выхода показываем меню для неавторизованных
		sendMainMenu(chatID, bot, nil)
		return

	case "menu_help":
		bot.Request(tgbotapi.NewCallback(callback.ID, "Справка"))
		bot.Send(tgbotapi.NewMessage(chatID, "Доступные действия:\n"+
			"• Регистрация / Вход — для неавторизованных\n"+
			"• Расписание / Материалы / Выход — для авторизованных\n"+
			"• Справка — показать это сообщение"))
		// Возвращаемся в меню (в зависимости от авторизации)
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return
	}

	// Если callback не относится к главному меню, передаём обработку регистрации (выбор факультета/группы).
	RegistrationProcessCallback(callback, bot)
}

// --- Заглушки для отсутствующих функций ---
// Если у вас функции sendFacultySelection и RegistrationProcessCallback реализованы в другом файле,
// убедитесь, что все файлы находятся в одном пакете. Если их нет, можно добавить следующие заглушки:
