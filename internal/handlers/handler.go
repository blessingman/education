package handlers

import (
	"education/internal/auth"
	"education/internal/models"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	greetedUsers   = make(map[int64]bool)
	greetedUsersMu sync.RWMutex
	// Хранилище для всех MessageID сообщений, отправленных ботом
	chatMessages   = make(map[int64][]int) // chatID -> []MessageID
	chatMessagesMu sync.RWMutex
)

// sendMessageAndTrack отправляет сообщение и сохраняет его MessageID в tempUserData или loginData

// sendAndTrackMessage отправляет сообщение и сохраняет его MessageID в глобальном хранилище
func sendAndTrackMessage(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig) error {
	sentMsg, err := bot.Send(msg)
	if err != nil {
		fmt.Println("Ошибка отправки сообщения:", err)
		return err
	}

	// Сохраняем MessageID в глобальном хранилище
	chatMessagesMu.Lock()
	chatMessages[msg.ChatID] = append(chatMessages[msg.ChatID], sentMsg.MessageID)
	chatMessagesMu.Unlock()

	return nil
}

// deleteMessages удаляет все сообщения, связанные с процессом
// deleteMessages удаляет все сообщения, связанные с данным chatID
func deleteMessages(chatID int64, bot *tgbotapi.BotAPI, delay time.Duration) {
	chatMessagesMu.Lock()
	defer chatMessagesMu.Unlock()

	// Получаем все MessageID для данного chatID
	msgIDs, exists := chatMessages[chatID]
	if !exists {
		return // Нет сообщений для удаления
	}

	// Задержка перед удалением (если задана)
	if delay > 0 {
		time.Sleep(delay)
	}

	// Удаляем все сообщения
	for _, msgID := range msgIDs {
		delMsg := tgbotapi.NewDeleteMessage(chatID, msgID)
		if _, err := bot.Request(delMsg); err != nil {
			fmt.Println("Ошибка удаления сообщения:", err)
		}
	}

	// Очищаем список MessageID для данного chatID
	chatMessages[chatID] = nil
}

// sendMainMenu формирует меню (Reply-кнопка «Главное меню» + Inline-кнопки),
// с приветствием при первом вызове и коротким текстом при повторных вызовах.
func sendMainMenu(chatID int64, bot *tgbotapi.BotAPI, user *models.User) {
	// Кнопка «Главное меню»
	replyKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	replyKeyboard.OneTimeKeyboard = false
	replyKeyboard.ResizeKeyboard = true

	// Проверка, приветствовался ли уже пользователь
	greetedUsersMu.RLock()
	alreadyGreeted := greetedUsers[chatID]
	greetedUsersMu.RUnlock()

	var firstMsgText string
	if !alreadyGreeted {
		firstMsgText = "Привет! 👋 Нажми «🏠 Главное меню», если захочешь вернуться к списку действий."
		greetedUsersMu.Lock()
		greetedUsers[chatID] = true
		greetedUsersMu.Unlock()
	} else if user != nil {
		if user.Role == "teacher" {
			// Для преподавателя показываем базовую информацию без курсов и групп
			firstMsgText = fmt.Sprintf("👤 Привет, %s!\n🏫 Факультет: %s\n🔑 Роль: %s",
				user.Name, user.Faculty, user.Role)
		} else {
			// Для студента
			firstMsgText = fmt.Sprintf("👤 Привет, %s!\n🏫 Факультет: %s\n📚 Группа: %s\n🔑 Роль: %s",
				user.Name, user.Faculty, user.Group, user.Role)
		}
	} else {
		firstMsgText = "🤖 Готов к работе! Выбирай действие ниже."
	}

	// Отправляем сообщение приветствия
	msg1 := tgbotapi.NewMessage(chatID, firstMsgText)
	msg1.ReplyMarkup = replyKeyboard
	sendAndTrackMessage(bot, msg1)

	// Формируем inline-кнопки меню
	var rows [][]tgbotapi.InlineKeyboardButton

	if user == nil {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Регистрация", "menu_register"),
			tgbotapi.NewInlineKeyboardButtonData("🔑 Вход", "menu_login"),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗓 Расписание", "menu_schedule"),
			tgbotapi.NewInlineKeyboardButtonData("📚 Материалы", "menu_materials"),
		))
		if user.Role == "teacher" {
			// Новая кнопка для преподавателя
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Мои предметы и группы", "menu_teacher_courses"),
			))
			// Существующие кнопки для преподавателя
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🛠 Изменить расписание", "menu_edit_schedule"),
				tgbotapi.NewInlineKeyboardButtonData("🛠 Изменить материалы", "menu_edit_materials"),
			))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚪 Выход", "menu_logout"),
		))
	}

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg2 := tgbotapi.NewMessage(chatID, "Выберите действие:")
	msg2.ReplyMarkup = inlineKeyboard
	sendAndTrackMessage(bot, msg2)
}

// Остальные функции (ProcessMessage, ProcessCallback) можно оставить без изменений,
// или при желании тоже подкорректировать тексты сообщений.

// ProcessMessage — обрабатывает входящие текстовые сообщения (включая нажатие «Главное меню»).
func ProcessMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// Если пользователь нажал на кнопку «Главное меню» (ReplyKeyboard)
	if text == "🏠 Главное меню" {
		// Сбрасываем все активные процессы (регистрация, логин и т.д.)
		if userStates[chatID] != "" {
			delete(userStates, chatID)
			delete(userTempDataMap, chatID)
		}
		if loginStates[chatID] != "" {
			delete(loginStates, chatID)
			delete(loginTempDataMap, chatID)
		}

		// Показываем главное меню без удаления сообщений
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return
	}

	// --- Проверка /cancel ---
	if update.Message.IsCommand() && update.Message.Command() == "cancel" {
		// Сбрасываем состояния
		if userStates[chatID] != "" {
			delete(userStates, chatID)
			delete(userTempDataMap, chatID)
		}
		if loginStates[chatID] != "" {
			delete(loginStates, chatID)
			delete(loginTempDataMap, chatID)
		}

		// Отправляем сообщение об отмене
		msg := tgbotapi.NewMessage(chatID, "❌ Процесс отменён.")
		sendAndTrackMessage(bot, msg)

		// Показываем главное меню без удаления сообщений
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return
	}

	// Если пользователь в процессе логина
	if state, ok := loginStates[chatID]; ok {
		processLoginMessage(update, bot, state, text)
		return
	}

	// Если пользователь в процессе регистрации
	if state, ok := userStates[chatID]; ok {
		processRegistrationMessage(update, bot, state, text)
		return
	}

	// Если нет активного процесса, проверяем, не ввёл ли он другую команду
	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			user, _ := auth.GetUserByTelegramID(chatID)
			sendMainMenu(chatID, bot, user)
			return
		case "logout":
			user, err := auth.GetUserByTelegramID(chatID)
			if err != nil || user == nil {
				msg := tgbotapi.NewMessage(chatID, "Вы не авторизованы.")
				sendAndTrackMessage(bot, msg)
			} else {
				user.TelegramID = 0
				_ = auth.SaveUser(user)
				deleteMessages(chatID, bot, 4*time.Second) // Удаляем сообщения при выходе
				msg := tgbotapi.NewMessage(chatID, "Вы успешно вышли. До скорой встречи!")
				sendAndTrackMessage(bot, msg)
				sendMainMenu(chatID, bot, nil)
			}
			return
		default:
			user, _ := auth.GetUserByTelegramID(chatID)
			sendMainMenu(chatID, bot, user)
			return
		}
	} else {
		// Любой другой текст – просто показываем меню заново
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return
	}
}

// ProcessCallback — обрабатывает нажатия инлайн-кнопок (меню регистрации, входа, расписания и т.д.).
func ProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// Обработка пагинации расписания
	if strings.HasPrefix(data, "schedule_page_") {
		// Извлекаем номер страницы из callback data
		pageStr := strings.TrimPrefix(data, "schedule_page_")
		newPage, err := strconv.Atoi(pageStr)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки страницы"))
			return
		}

		// Задаём количество записей на страницу
		limit := 5
		offset := (newPage - 1) * limit

		// Получаем данные пользователя
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка получения данных пользователя"))
			return
		}

		// Выбираем расписание в зависимости от роли пользователя
		var schedules []models.Schedule
		var totalRecords int
		if user.Role == "teacher" {
			schedules, err = GetScheduleByTeacherPaginated(user.RegistrationCode, limit, offset)
			if err != nil {
				bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка получения расписания"))
				return
			}
			totalRecords, err = CountSchedulesByTeacher(user.RegistrationCode)
		} else {
			schedules, err = GetScheduleByGroupPaginated(user.Group, limit, offset)
			if err != nil {
				bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка получения расписания"))
				return
			}
			totalRecords, err = CountSchedulesByGroup(user.Group)
		}
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка подсчёта записей"))
			return
		}

		// Вычисляем общее количество страниц
		totalPages := (totalRecords + limit - 1) / limit

		// Форматируем новый текст сообщения с расписанием
		newText := FormatPaginatedSchedules(schedules, newPage, totalPages, user.Role, user)
		// Формируем новую клавиатуру с пагинацией (используя функцию с прямой навигацией по номерам страниц)
		newKeyboard := BuildPaginationKeyboardWithNumbers(newPage, totalPages, "schedule")

		// Создаем объект для редактирования сообщения
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, newText)
		editMsg.ParseMode = "HTML" // или "Markdown", в зависимости от используемого синтаксиса
		editMsg.ReplyMarkup = &newKeyboard

		// Отправляем измененное сообщение
		if _, err := bot.Send(editMsg); err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обновления расписания"))
			return
		}

		// Отправляем callback ответ для скрытия индикатора загрузки
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}
	// Если пользователь уже в процессе регистрации/логина, не даём начать другой процесс
	if userStates[chatID] != "" || loginStates[chatID] != "" {
		switch callback.Data {
		case "menu_register", "menu_login":
			bot.Request(tgbotapi.NewCallback(callback.ID,
				"Сначала заверши текущий процесс или отмени его командой /cancel."))
			return
		}
	}

	switch callback.Data {
	case "menu_register":
		userStates[chatID] = StateWaitingForRole
		userTempDataMap[chatID] = &tempUserData{}
		bot.Request(tgbotapi.NewCallback(callback.ID, "📝 Начинаем регистрацию!"))
		sendRoleSelection(chatID, bot)
		return

	case "menu_login":
		loginStates[chatID] = LoginStateWaitingForRegCode
		loginTempDataMap[chatID] = &loginData{}
		bot.Request(tgbotapi.NewCallback(callback.ID, "🔑 Выполняем вход..."))
		msg := tgbotapi.NewMessage(chatID, "Введите свой регистрационный код:")
		sendAndTrackMessage(bot, msg)
		return

	case "menu_schedule":
		bot.Request(tgbotapi.NewCallback(callback.ID, "🗓 Расписание"))
		user, _ := auth.GetUserByTelegramID(chatID)
		if err := ShowSchedule(chatID, bot, user); err != nil {
			fmt.Println("Ошибка при отправке расписания:", err)
		}
		return

	case "menu_materials":
		bot.Request(tgbotapi.NewCallback(callback.ID, "📚 Материалы"))
		user, _ := auth.GetUserByTelegramID(chatID)
		if err := ShowMaterials(chatID, bot, user); err != nil {
			fmt.Println("Ошибка при отправке материалов:", err)
		}
		return

	case "menu_logout":
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Вы не авторизованы."))
		} else {
			user.TelegramID = 0
			_ = auth.SaveUser(user)
			bot.Request(tgbotapi.NewCallback(callback.ID, "🚪 Выход"))
			msg := tgbotapi.NewMessage(chatID, "Вы успешно вышли. До скорой встречи!")
			sendAndTrackMessage(bot, msg)
			// Удаляем все сообщения из чата с задержкой
			deleteMessages(chatID, bot, 4*time.Second)
		}
		// Показываем главное меню без авторизации
		sendMainMenu(chatID, bot, nil)
		return

	case "menu_help":
		bot.Request(tgbotapi.NewCallback(callback.ID, "❓ Справка"))
		msg := tgbotapi.NewMessage(chatID,
			"Вот что я умею:\n"+
				"• Студенты: смотреть расписание и материалы\n"+
				"• Преподаватели: плюс редактировать расписание и материалы\n"+
				"• Кнопка «Выход» завершает работу\n"+
				"• В любой момент жми «🏠 Главное меню» внизу экрана")
		sendAndTrackMessage(bot, msg)
		return

	case "menu_edit_schedule":
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil || user.Role != "teacher" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "У вас нет прав для редактирования расписания."))
			return
		}
		bot.Request(tgbotapi.NewCallback(callback.ID, "🛠 Изменение расписания..."))
		msg := tgbotapi.NewMessage(chatID, "Добавьте или отредактируйте расписание (реализуйте по-своему).")
		sendAndTrackMessage(bot, msg)
		return

	case "menu_edit_materials":
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil || user.Role != "teacher" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "У вас нет прав для редактирования материалов."))
			return
		}
		bot.Request(tgbotapi.NewCallback(callback.ID, "🛠 Изменение материалов..."))
		msg := tgbotapi.NewMessage(chatID, "Здесь можно загрузить или обновить учебные материалы (реализуйте по-своему).")
		sendAndTrackMessage(bot, msg)
		return
	case "menu_teacher_courses":
		// Получаем пользователя по chatID
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil || user.Role != "teacher" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Нет доступа"))
			return
		}

		// Используем registration_code для получения данных
		courses, err := GetCoursesByTeacherRegCode(user.RegistrationCode)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка получения курсов"))
			return
		}
		groups, err := GetTeacherGroupsByRegCode(user.RegistrationCode)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка получения групп"))
			return
		}

		// Группируем группы по идентификатору курса
		courseGroups := make(map[int64][]string)
		for _, g := range groups {
			courseGroups[g.CourseID] = append(courseGroups[g.CourseID], g.GroupName)
		}

		// Формирование текстового сообщения
		var msgText string
		for _, course := range courses {
			groupsForCourse := courseGroups[course.ID]
			if len(groupsForCourse) == 0 {
				msgText += fmt.Sprintf("📘 %s: нет групп\n", course.Name)
			} else {
				msgText += fmt.Sprintf("📘 %s: %s\n", course.Name, strings.Join(groupsForCourse, ", "))
			}
		}
		if msgText == "" {
			msgText = "Нет данных для отображения."
		}

		// Отправка сообщения
		msg := tgbotapi.NewMessage(chatID, msgText)
		sendAndTrackMessage(bot, msg)
		return

	}

	// Если callback не относится к главному меню, передаём обработку регистрации/логина
	RegistrationProcessCallback(callback, bot)
}
