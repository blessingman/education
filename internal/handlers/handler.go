package handlers

import (
	"education/internal/auth"
	"education/internal/models"
	"fmt"
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

	// Получим пользователя (если нужен во многих ветках)
	user, err := auth.GetUserByTelegramID(chatID)
	if err != nil {
		// Если мы не можем получить пользователя, то часть функций будет недоступна
		// но можем вывести callback
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка получения данных пользователя"))
		return
	}

	// Проверяем, не является ли callback связанным с материалами
	if user != nil && ProcessMaterialsCallback(callback, bot, user) {
		return
	}

	// Проверяем, не является ли callback связанным с фильтрами расписания
	if strings.HasPrefix(data, "filter_") {
		if data == "filter_course_menu" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Выбор курса для фильтра"))
			ShowCourseFilterMenu(chatID, bot)
			return
		} else if data == "filter_lesson_type_menu" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Выбор типа занятия для фильтра"))
			ShowLessonTypeFilterMenu(chatID, bot)
			return
		} else if data == "filter_reset_all" {
			ResetUserFilters(chatID)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Фильтры сброшены"))
			ShowFilterMenu(chatID, bot)
			return
		} else if data == "filter_course_reset" {
			filter := GetUserFilter(chatID)
			filter.CourseID = 0
			filter.CourseName = ""
			SetUserFilter(chatID, filter)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Фильтр по курсу сброшен"))
			ShowFilterMenu(chatID, bot)
			return
		} else if data == "filter_lesson_type_reset" {
			filter := GetUserFilter(chatID)
			filter.LessonType = ""
			SetUserFilter(chatID, filter)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Фильтр по типу занятия сброшен"))
			ShowFilterMenu(chatID, bot)
			return
		} else if data == "filter_menu" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Возврат к меню фильтров"))
			ShowFilterMenu(chatID, bot)
			return
		} else if data == "filter_apply" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Применение фильтров"))
			// Получаем текущую дату
			now := time.Now()
			offset := int(now.Weekday())
			if offset == 0 {
				offset = 7
			}
			weekStart := now.AddDate(0, 0, -(offset - 1))
			ShowScheduleWeek(chatID, bot, user, weekStart)
			return
		} else if strings.HasPrefix(data, "filter_course_") && !strings.HasPrefix(data, "filter_course_menu") && !strings.HasPrefix(data, "filter_course_reset") {
			// Формат: filter_course_ID_NAME
			parts := strings.SplitN(strings.TrimPrefix(data, "filter_course_"), "_", 2)
			if len(parts) == 2 {
				courseID := parts[0]
				courseName := parts[1]

				filter := GetUserFilter(chatID)
				filter.CourseID = parseID(courseID) // Добавим функцию для конвертации строки в int64
				filter.CourseName = courseName
				SetUserFilter(chatID, filter)
				bot.Request(tgbotapi.NewCallback(callback.ID, "Выбран курс: "+courseName))
				ShowFilterMenu(chatID, bot)
				return
			}
		} else if strings.HasPrefix(data, "filter_lesson_type_") && !strings.HasPrefix(data, "filter_lesson_type_menu") && !strings.HasPrefix(data, "filter_lesson_type_reset") {
			lessonType := strings.TrimPrefix(data, "filter_lesson_type_")
			filter := GetUserFilter(chatID)
			filter.LessonType = lessonType
			SetUserFilter(chatID, filter)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Выбран тип занятия: "+lessonType))
			ShowFilterMenu(chatID, bot)
			return
		}
	}

	// Обработка пагинации расписания
	// Обработка навигации по неделям
	// Обработка навигации по неделям
	// --- 1) Навигация по неделям ---
	if strings.HasPrefix(data, "week_prev_") {
		currentWeekStr := strings.TrimPrefix(data, "week_prev_")
		currentWeekStart, err := time.Parse("2006-01-02", currentWeekStr)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки даты"))
			return
		}
		newWeekStart := currentWeekStart.AddDate(0, 0, -7)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		ShowScheduleWeek(chatID, bot, user, newWeekStart)
		return
	}

	if strings.HasPrefix(data, "week_next_") {
		currentWeekStr := strings.TrimPrefix(data, "week_next_")
		currentWeekStart, err := time.Parse("2006-01-02", currentWeekStr)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки даты"))
			return
		}
		newWeekStart := currentWeekStart.AddDate(0, 0, 7)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		ShowScheduleWeek(chatID, bot, user, newWeekStart)
		return
	}

	// Удаляем обработку навигации по месяцам
	// НЕ НУЖНО: Обработка навигации по месяцам

	if data == "week_today" {
		bot.Request(tgbotapi.NewCallback(callback.ID, "Переход к текущей неделе"))
		now := time.Now()
		offset := int(now.Weekday())
		if offset == 0 {
			offset = 7
		}
		weekStart := now.AddDate(0, 0, -(offset - 1))
		ShowScheduleWeek(chatID, bot, user, weekStart)
		return
	}
	if data == "mode_day" {
		bot.Request(tgbotapi.NewCallback(callback.ID, "Переход к дневному режиму"))
		// Используем новую улучшенную версию
		selectedDay, err := time.Parse("2006-01-02", time.Now().Format("2006-01-02"))
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки даты"))
			return
		}
		err = ShowEnhancedScheduleDay(chatID, bot, user, selectedDay)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка отображения дневного расписания"))
		}
		return
	} else if data == "mode_week" {
		now := time.Now()
		offset := int(now.Weekday())
		if offset == 0 {
			offset = 7
		}
		weekStart := now.AddDate(0, 0, -(offset - 1))
		bot.Request(tgbotapi.NewCallback(callback.ID, "Переход к недельному режиму"))
		err := ShowScheduleWeek(chatID, bot, user, weekStart)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка отображения недельного расписания"))
		}
		return
	} else if data == "mode_month" {
		// Сохраняем этот обработчик, но меняем его поведение
		bot.Request(tgbotapi.NewCallback(callback.ID, "Режим 'Месяц' больше не поддерживается"))
		return
	}
	// В начале ProcessCallback, после получения user
	// Обработка таймлайна
	if data == "show_timeline" {
		now := time.Now()
		dayStart := now.Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24*time.Hour - time.Second)

		var schedules []models.Schedule
		if user.Role == "teacher" {
			schedules, err = GetSchedulesForTeacherByDateRange(user.RegistrationCode, dayStart, dayEnd)
		} else {
			schedules, err = GetSchedulesForGroupByDateRange(user.Group, dayStart, dayEnd)
		}
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка загрузки расписания"))
			return
		}

		timelineText := BuildCalendarTimeline(schedules, now)
		msg := tgbotapi.NewMessage(chatID, timelineText)
		msg.ParseMode = "HTML"
		if err := sendAndTrackMessage(bot, msg); err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка отображения таймлайна"))
		} else {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Таймлайн загружен"))
		}
		return
	}

	// --- 2) Навигация по конкретному дню ---
	if strings.HasPrefix(data, "day_") {
		dayStr := strings.TrimPrefix(data, "day_")
		selectedDay, err := time.Parse("2006-01-02", dayStr)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки даты"))
			return
		}
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		// Используем новую улучшенную версию вместо старой
		ShowEnhancedScheduleDay(chatID, bot, user, selectedDay)
		return
	}

	// --- 3) Фильтрация по курсу ---
	// 3.1) Кнопка, открывающая меню выбора курса
	if data == "filter_menu" {
		bot.Request(tgbotapi.NewCallback(callback.ID, "Выбор фильтра"))
		ShowFilterMenu(chatID, bot)
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
		// Улучшенное меню выбора режима расписания
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			// Первый ряд с режимами просмотра
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📅 День", "mode_day"),
				tgbotapi.NewInlineKeyboardButtonData("📊 Неделя", "mode_week"),
			),
			// Второй ряд с кнопкой фильтров, выделенной и заметной
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔍 Настроить фильтры", "filter_menu"),
			),
		)
		msg := tgbotapi.NewMessage(chatID, "<b>Просмотр расписания</b>\n\nВыберите режим отображения или настройте фильтры:")
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard
		sendAndTrackMessage(bot, msg)
		return

	case "menu_materials":
		bot.Request(tgbotapi.NewCallback(callback.ID, "📚 Материалы"))
		user, _ := auth.GetUserByTelegramID(chatID)

		// Сбрасываем состояние пагинации материалов при первом входе
		materialStateMutex.Lock()
		materialPageState[chatID] = 1       // Начинаем с первой страницы
		delete(materialFilterState, chatID) // Сбрасываем фильтр
		materialStateMutex.Unlock()

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
		// Answer callback immediately to stop the looping animation
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		// Получаем пользователя по chatID
		user, err := auth.GetUserByTelegramID(chatID)
		if err != nil || user == nil || user.Role != "teacher" {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Нет доступа"))
			return
		}

		// Получаем курсы и группы преподавателя
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

		// Группируем группы по курсам
		courseGroups := make(map[int64][]string)
		for _, g := range groups {
			courseGroups[g.CourseID] = append(courseGroups[g.CourseID], g.GroupName)
		}

		// Формируем текстовое сообщение с использованием strings.Builder
		var builder strings.Builder
		for _, course := range courses {
			if groupsForCourse, ok := courseGroups[course.ID]; !ok || len(groupsForCourse) == 0 {
				builder.WriteString(fmt.Sprintf("📘 %s: нет групп\n", course.Name))
			} else {
				builder.WriteString(fmt.Sprintf("📘 %s: %s\n", course.Name, strings.Join(groupsForCourse, ", ")))
			}
			// Добавляем информацию о ближайших событиях (пока как заглушку)
		}
		msgText := builder.String()
		if strings.TrimSpace(msgText) == "" {
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

// Вспомогательная функция для конвертации строкового ID в int64
func parseID(idStr string) int64 {
	var id int64
	fmt.Sscanf(idStr, "%d", &id)
	return id
}
