package handlers

import (
	"education/internal/auth"
	"education/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// sendMainMenu отправляет меню, которое зависит от того, авторизован пользователь или нет.
func sendMainMenu(chatID int64, bot *tgbotapi.BotAPI, user *models.User) {
	var rows [][]tgbotapi.InlineKeyboardButton

	if user == nil {
		// Пользователь не авторизован – кнопки "Регистрация" и "Вход"
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Регистрация", "menu_register"),
			tgbotapi.NewInlineKeyboardButtonData("Вход", "menu_login"),
		))
	} else {
		// Общие кнопки для всех авторизованных пользователей
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Расписание", "menu_schedule"),
			tgbotapi.NewInlineKeyboardButtonData("Материалы", "menu_materials"),
		))
		// Если роль – "teacher", даём доп. возможности
		if user.Role == "teacher" {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Редактировать расписание", "menu_edit_schedule"),
				tgbotapi.NewInlineKeyboardButtonData("Редактировать материалы", "menu_edit_materials"),
			))
		}
		// Кнопка "Выход"
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Выход", "menu_logout"),
		))
	}

	// Кнопка "Справка" доступна всем
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
	text := update.Message.Text

	// 1. Проверка на команду /cancel – если пользователь решил отменить текущий процесс
	if update.Message.IsCommand() && update.Message.Command() == "cancel" {
		if userStates[chatID] != "" {
			delete(userStates, chatID)
			delete(userTempDataMap, chatID)
			bot.Send(tgbotapi.NewMessage(chatID, "Регистрация отменена."))
		}
		if loginStates[chatID] != "" {
			delete(loginStates, chatID)
			delete(loginTempDataMap, chatID)
			bot.Send(tgbotapi.NewMessage(chatID, "Вход отменён."))
		}
		// Показываем главное меню
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return
	}

	// 2. Если пользователь находится в процессе логина, вызываем соответствующую обработку
	if state, ok := loginStates[chatID]; ok {
		processLoginMessage(update, bot, state, text)
		return
	}

	// 3. Если пользователь находится в процессе регистрации, вызываем обработку регистрации
	if state, ok := userStates[chatID]; ok {
		processRegistrationMessage(update, bot, state, text)
		return
	}

	// 4. Если нет активного процесса, обрабатываем команды и обычные сообщения
	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			user, _ := auth.GetUserByTelegramID(chatID)
			sendMainMenu(chatID, bot, user)
			return
		default:
			user, _ := auth.GetUserByTelegramID(chatID)
			sendMainMenu(chatID, bot, user)
			return
		}
	} else {
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return
	}
}

// ProcessCallback — обрабатывает callback‑запросы инлайн-кнопок (главное меню + вызов RegistrationProcessCallback).
func ProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID

	// Если пользователь уже находится в процессе регистрации или логина,
	// блокируем попытки начать новый процесс.
	if userStates[chatID] != "" || loginStates[chatID] != "" {
		switch callback.Data {
		case "menu_register", "menu_login":
			bot.Request(tgbotapi.NewCallback(callback.ID, "У вас уже идёт процесс. Сначала завершите или отмените его."))
			return
		}
	}

	switch callback.Data {
	case "menu_register":
		userStates[chatID] = StateWaitingForRole
		userTempDataMap[chatID] = &tempUserData{}
		bot.Request(tgbotapi.NewCallback(callback.ID, "Переходим к регистрации"))
		sendRoleSelection(chatID, bot)
		return

	case "menu_login":
		loginStates[chatID] = LoginStateWaitingForRegCode
		loginTempDataMap[chatID] = &loginData{}
		bot.Request(tgbotapi.NewCallback(callback.ID, "Переходим к входу"))
		bot.Send(tgbotapi.NewMessage(chatID, "🔑 Введите ваш регистрационный код:"))
		return

	case "menu_schedule":
		bot.Request(tgbotapi.NewCallback(callback.ID, "Расписание"))
		bot.Send(tgbotapi.NewMessage(chatID, "Ваше расписание: (здесь может быть реальная логика)"))
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return

	case "menu_materials":
		bot.Request(tgbotapi.NewCallback(callback.ID, "Материалы"))
		bot.Send(tgbotapi.NewMessage(chatID, "Список материалов: (здесь может быть реальная логика)"))
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return

	case "menu_logout":
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Вы не авторизованы"))
		} else {
			// 1. Сбрасываем Telegram ID (пользователь выходит из системы)
			user.TelegramID = 0
			_ = auth.SaveUser(user)

			// 2. Отправляем уведомление
			bot.Request(tgbotapi.NewCallback(callback.ID, "Вы вышли из системы"))
			bot.Send(tgbotapi.NewMessage(chatID, "✅ Вы успешно вышли из системы."))

			// 3. Удаляем старое сообщение с кнопками (чтобы пользователь не мог нажать их повторно)
			delMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
			bot.Request(delMsg)
		}

		// 4. Отправляем новое меню (для неавторизованного пользователя)
		sendMainMenu(chatID, bot, nil)
		return

	case "menu_help":
		bot.Request(tgbotapi.NewCallback(callback.ID, "Справка"))
		bot.Send(tgbotapi.NewMessage(chatID, "Доступные действия:\n"+
			"• Для студентов: Просмотр расписания и материалов\n"+
			"• Для преподавателей: Просмотр, а также редактирование расписания и материалов\n"+
			"• Выход — завершить работу с ботом"))
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return

	// Дополнительные команды для преподавателей
	case "menu_edit_schedule":
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil || user.Role != "teacher" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Доступ запрещён."))
			return
		}
		bot.Request(tgbotapi.NewCallback(callback.ID, "Переходим к редактированию расписания"))
		// Здесь реализуйте логику редактирования расписания
		bot.Send(tgbotapi.NewMessage(chatID, "Редактирование расписания: (здесь ваша логика)"))
		sendMainMenu(chatID, bot, user)
		return

	case "menu_edit_materials":
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil || user.Role != "teacher" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Доступ запрещён."))
			return
		}
		bot.Request(tgbotapi.NewCallback(callback.ID, "Переходим к редактированию материалов"))
		// Здесь реализуйте логику редактирования материалов
		bot.Send(tgbotapi.NewMessage(chatID, "Редактирование материалов: (здесь ваша логика)"))
		sendMainMenu(chatID, bot, user)
		return
	}

	// Если callback не относится к главному меню, передаём обработку регистрации или других шагов.
	RegistrationProcessCallback(callback, bot)
}
