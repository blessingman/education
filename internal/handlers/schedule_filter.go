package handlers

import (
	"education/internal/auth"
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ScheduleFilter структура для хранения фильтров расписания
type ScheduleFilter struct {
	CourseID   int64  // ID курса
	CourseName string // Название курса
	LessonType string // Тип занятия (Лекция, Практика, Лабораторная, Семинар)
}

var (
	// Хранилище фильтров для каждого пользователя
	userFilters     = make(map[int64]*ScheduleFilter)
	userFilterMutex sync.RWMutex
)

// Получение фильтра пользователя
func GetUserFilter(chatID int64) *ScheduleFilter {
	userFilterMutex.RLock()
	defer userFilterMutex.RUnlock()

	filter, exists := userFilters[chatID]
	if !exists {
		return &ScheduleFilter{} // Возвращаем пустой фильтр, если нет сохраненного
	}
	return filter
}

// Установка фильтра пользователя
func SetUserFilter(chatID int64, filter *ScheduleFilter) {
	userFilterMutex.Lock()
	defer userFilterMutex.Unlock()

	userFilters[chatID] = filter
}

// Сброс фильтра пользователя
func ResetUserFilters(chatID int64) {
	userFilterMutex.Lock()
	defer userFilterMutex.Unlock()

	userFilters[chatID] = &ScheduleFilter{}
}

// Применение фильтров к выборке расписания
func ApplyFilters(schedules []models.Schedule, filter *ScheduleFilter) []models.Schedule {
	if filter == nil || (filter.CourseID == 0 && filter.CourseName == "" && filter.LessonType == "") {
		return schedules
	}

	var filtered []models.Schedule
	for _, s := range schedules {
		// Фильтр по курсу
		if filter.CourseID != 0 && s.CourseID != filter.CourseID {
			continue
		}

		// Фильтр по названию курса (если указан)
		if filter.CourseName != "" && !strings.Contains(strings.ToLower(s.Description), strings.ToLower(filter.CourseName)) {
			continue
		}

		// Фильтр по типу занятия (если указан)
		if filter.LessonType != "" && s.LessonType != filter.LessonType {
			continue
		}

		filtered = append(filtered, s)
	}

	return filtered
}

// GetRelevantCoursesForUser возвращает только курсы, которые есть в расписании пользователя
func GetRelevantCoursesForUser(user *models.User) ([]models.Course, error) {
	var query string
	var args []interface{}

	if user.Role == "teacher" {
		query = `
			SELECT DISTINCT c.id, c.name 
			FROM courses c
			JOIN schedules s ON s.course_id = c.id
			WHERE s.teacher_reg_code = ?
			ORDER BY c.name
		`
		args = append(args, user.RegistrationCode)
	} else {
		query = `
			SELECT DISTINCT c.id, c.name 
			FROM courses c
			JOIN schedules s ON s.course_id = c.id
			WHERE s.group_name = ?
			ORDER BY c.name
		`
		args = append(args, user.Group)
	}

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		if err := rows.Scan(&course.ID, &course.Name); err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}
	return courses, rows.Err()
}

// GetRelevantLessonTypes возвращает только типы занятий, которые есть в расписании пользователя
func GetRelevantLessonTypes(user *models.User) ([]string, error) {
	var query string
	var args []interface{}

	if user.Role == "teacher" {
		query = `
			SELECT DISTINCT lesson_type
			FROM schedules
			WHERE teacher_reg_code = ?
			ORDER BY lesson_type
		`
		args = append(args, user.RegistrationCode)
	} else {
		query = `
			SELECT DISTINCT lesson_type
			FROM schedules
			WHERE group_name = ?
			ORDER BY lesson_type
		`
		args = append(args, user.Group)
	}

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessonTypes []string
	for rows.Next() {
		var lessonType string
		if err := rows.Scan(&lessonType); err != nil {
			return nil, err
		}
		lessonTypes = append(lessonTypes, lessonType)
	}
	return lessonTypes, rows.Err()
}

// ShowFilterMenu отображает улучшенное меню фильтров расписания
func ShowFilterMenu(chatID int64, bot *tgbotapi.BotAPI) error {
	// Get user data
	user, err := auth.GetUserByTelegramID(chatID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка получения данных пользователя")
		return sendAndTrackMessage(bot, msg)
	}

	// Получаем информацию о текущих фильтрах
	filter := GetUserFilter(chatID)

	// Get available filter info based on user role
	var availableFilterInfo string
	if user.Role == "teacher" {
		// Check if teacher has any schedules with lesson types
		lessonTypes, err := GetRelevantLessonTypes(user)
		if err == nil && len(lessonTypes) > 0 {
			availableFilterInfo = fmt.Sprintf("Доступные типы занятий: %s", strings.Join(lessonTypes, ", "))
		}
	} else {
		// For students, show available lesson types as well
		lessonTypes, err := GetRelevantLessonTypes(user)
		if err == nil && len(lessonTypes) > 0 {
			availableFilterInfo = fmt.Sprintf("Доступные типы занятий: %s", strings.Join(lessonTypes, ", "))
		}
	}

	// Создаем более привлекательный заголовок
	var headerText strings.Builder
	headerText.WriteString("🔍 <b>Фильтры расписания</b>\n\n")

	// Информация о текущих фильтрах
	if filter.CourseName != "" || filter.LessonType != "" {
		headerText.WriteString("<b>Активные фильтры:</b>\n")

		if filter.CourseName != "" {
			headerText.WriteString(fmt.Sprintf("• 📚 Курс: <b>%s</b>\n", filter.CourseName))
		}

		if filter.LessonType != "" {
			headerText.WriteString(fmt.Sprintf("• 📝 Тип занятия: <b>%s</b>\n", filter.LessonType))
		}

		headerText.WriteString("\n")
	} else {
		headerText.WriteString("ℹ️ <i>Фильтры не установлены</i>\n\n")
	}

	// Add available filter info if we found any
	if availableFilterInfo != "" {
		headerText.WriteString("<i>" + availableFilterInfo + "</i>\n\n")
	}

	headerText.WriteString("Выберите опцию:")

	// Кнопки фильтров в более привлекательном формате
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📚 Фильтр по курсу", "filter_course_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Фильтр по типу занятия", "filter_lesson_type_menu"),
		),
		// Добавляем разделительную строку для кнопок сброса и применения
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Сбросить все фильтры", "filter_reset_all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Применить", "filter_apply"),
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_schedule"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, headerText.String())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard

	return sendAndTrackMessage(bot, msg)
}

// ShowCourseFilterMenu отображает улучшенное меню выбора курса для фильтрации
func ShowCourseFilterMenu(chatID int64, bot *tgbotapi.BotAPI) {
	// Get the user
	user, err := auth.GetUserByTelegramID(chatID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка получения данных пользователя")
		sendAndTrackMessage(bot, msg)
		return
	}

	// Actually use the user variable by calling GetRelevantCoursesForUser
	// Получаем только релевантные курсы для пользователя
	courses, err := GetRelevantCoursesForUser(user)
	if err != nil {
		fmt.Println("Ошибка получения списка курсов:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при загрузке списка курсов")
		sendAndTrackMessage(bot, msg)
		return
	}

	if len(courses) == 0 {
		msg := tgbotapi.NewMessage(chatID, "В вашем расписании не найдены курсы")
		sendAndTrackMessage(bot, msg)
		return
	}

	// Текущий фильтр
	filter := GetUserFilter(chatID)

	var rows [][]tgbotapi.InlineKeyboardButton

	// Создаем более привлекательный заголовок
	headerText := "📚 <b>Выберите курс для фильтрации</b>\n\n"
	if filter.CourseName != "" {
		headerText += fmt.Sprintf("Текущий выбор: <b>%s</b>\n\n", filter.CourseName)
	}
	headerText += "Доступные курсы:"

	// Создаем кнопки для каждого курса с индикатором выбора
	for _, course := range courses {
		courseID := fmt.Sprintf("%d", course.ID)
		// Добавляем отметку к названию текущего выбранного курса
		buttonText := course.Name
		if filter.CourseID == course.ID {
			buttonText = "✅ " + buttonText
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, "filter_course_"+courseID+"_"+course.Name),
		))
	}

	// Добавляем кнопку для сброса фильтра и возврата
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Сбросить фильтр курса", "filter_course_reset"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Назад к фильтрам", "filter_menu"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, headerText)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	sendAndTrackMessage(bot, msg)
}

// ShowLessonTypeFilterMenu отображает улучшенное меню выбора типа занятия
func ShowLessonTypeFilterMenu(chatID int64, bot *tgbotapi.BotAPI) error {
	user, err := auth.GetUserByTelegramID(chatID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка получения данных пользователя")
		return sendAndTrackMessage(bot, msg)
	}

	// Получаем только релевантные типы занятий для пользователя
	lessonTypes, err := GetRelevantLessonTypes(user)
	if err != nil {
		fmt.Println("Ошибка получения типов занятий:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при загрузке типов занятий")
		return sendAndTrackMessage(bot, msg)
	}

	if len(lessonTypes) == 0 {
		msg := tgbotapi.NewMessage(chatID, "В вашем расписании не найдены типы занятий")
		return sendAndTrackMessage(bot, msg)
	}

	// Текущий фильтр
	filter := GetUserFilter(chatID)

	var rows [][]tgbotapi.InlineKeyboardButton

	// Создаем более привлекательный заголовок
	headerText := "📝 <b>Выберите тип занятия для фильтрации</b>\n\n"
	if filter.LessonType != "" {
		headerText += fmt.Sprintf("Текущий выбор: <b>%s</b>\n\n", filter.LessonType)
	}
	headerText += "Доступные типы занятий:"

	// Создаем кнопки для каждого типа занятия с индикатором выбора
	for _, lessonType := range lessonTypes {
		// Добавляем отметку к текущему выбранному типу
		buttonText := lessonType
		if filter.LessonType == lessonType {
			buttonText = "✅ " + buttonText
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, "filter_lesson_type_"+lessonType),
		))
	}

	// Добавляем кнопку для сброса фильтра и возврата
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Сбросить фильтр типа", "filter_lesson_type_reset"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Назад к фильтрам", "filter_menu"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, headerText)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	return sendAndTrackMessage(bot, msg)
}
