package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"sort"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ShowEnhancedScheduleDay shows an enhanced version of the daily schedule
func ShowEnhancedScheduleDay(chatID int64, bot *tgbotapi.BotAPI, user *models.User, day time.Time) error {
	dayStart := day.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24*time.Hour - time.Second)

	fmt.Printf("ShowEnhancedScheduleDay for user %+v, day: %s\n", user, day.Format("2006-01-02"))

	var schedules []models.Schedule
	var err error
	if user.Role == "teacher" {
		schedules, err = GetSchedulesForTeacherByDateRange(user.RegistrationCode, dayStart, dayEnd)
	} else {
		schedules, err = GetSchedulesForGroupByDateRange(user.Group, dayStart, dayEnd)
	}
	if err != nil {
		// Return a clear error message for daily schedule display
		errMsg := "Ошибка отображения дневного расписания: " + err.Error()
		msg := tgbotapi.NewMessage(chatID, errMsg)
		return sendAndTrackMessage(bot, msg)
	}

	// Применяем фильтры к полученному расписанию
	filter := GetUserFilter(chatID)
	filteredSchedules := ApplyFilters(schedules, filter)

	text := FormatEnhancedDaySchedule(filteredSchedules, day, user.Role)

	// Add navigation buttons for previous/next day
	prevDay := day.AddDate(0, 0, -1)
	nextDay := day.AddDate(0, 0, 1)

	// Создаем навигационные кнопки
	navRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Пред. день", fmt.Sprintf("day_%s", prevDay.Format("2006-01-02"))),
		tgbotapi.NewInlineKeyboardButtonData("Сегодня", "mode_day"),
		tgbotapi.NewInlineKeyboardButtonData("След. день ▶️", fmt.Sprintf("day_%s", nextDay.Format("2006-01-02"))),
	)

	// Получаем базовую клавиатуру переключения режимов
	modeKeyboard := BuildModeSwitchKeyboard("mode_day")

	// Добавляем кнопку фильтров
	filterRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔍 Настроить фильтры", "filter_menu"),
	)

	// Если фильтры активны, добавляем информацию о них в сообщение
	if filter.CourseName != "" || filter.LessonType != "" {
		text += "\n\n<b>📌 Активные фильтры:</b>\n"
		if filter.CourseName != "" {
			text += fmt.Sprintf("• Курс: <b>%s</b>\n", filter.CourseName)
		}
		if filter.LessonType != "" {
			text += fmt.Sprintf("• Тип занятия: <b>%s</b>\n", filter.LessonType)
		}
	}

	// Объединяем все ряды кнопок
	var allRows [][]tgbotapi.InlineKeyboardButton
	allRows = append(allRows, navRow)
	allRows = append(allRows, modeKeyboard.InlineKeyboard...)
	allRows = append(allRows, filterRow)

	enhancedKeyboard := tgbotapi.NewInlineKeyboardMarkup(allRows...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = enhancedKeyboard
	return sendAndTrackMessage(bot, msg)
}

// FormatEnhancedDaySchedule creates a beautifully formatted day schedule
func FormatEnhancedDaySchedule(schedules []models.Schedule, day time.Time, role string) string {
	if len(schedules) == 0 {
		return fmt.Sprintf("📆 <b>%s</b>\n\n🔍 <i>Нет занятий на этот день</i>",
			day.Format("02.01.2006")+" ("+weekdayName(day.Weekday())+")")
	}

	// Сортируем занятия по времени
	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].ScheduleTime.Before(schedules[j].ScheduleTime)
	})

	var sb strings.Builder
	// Заголовок с датой и днем недели
	sb.WriteString(fmt.Sprintf("📆 <b>%s</b>\n\n",
		day.Format("02.01.2006")+" ("+weekdayName(day.Weekday())+")"))

	// Разделитель заголовка
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Счетчик для занятий
	lessonCount := 0

	for _, s := range schedules {
		lessonCount++
		timeStr := s.ScheduleTime.Format("15:04")
		endTimeStr := s.ScheduleTime.Add(time.Duration(s.Duration) * time.Minute).Format("15:04")

		// Блок информации о занятии с порядковым номером
		sb.WriteString(fmt.Sprintf("📌 <b>Занятие %d</b>\n", lessonCount))
		sb.WriteString(fmt.Sprintf("⏰ <b>%s - %s</b> (%d мин.)\n", timeStr, endTimeStr, s.Duration))
		sb.WriteString(fmt.Sprintf("📚 <b>%s</b>\n", s.Description))

		if role == "teacher" {
			sb.WriteString(fmt.Sprintf("👥 Группа: %s\n", s.GroupName))
		} else {
			sb.WriteString(fmt.Sprintf("👨‍🏫 Преподаватель: %s\n", s.TeacherRegCode))
		}

		sb.WriteString(fmt.Sprintf("🚪 Аудитория: %s\n", s.Auditory))
		sb.WriteString(fmt.Sprintf("📝 Тип занятия: %s\n", s.LessonType))

		// Добавляем разделитель между занятиями
		sb.WriteString("\n")
	}

	// Итоговая информация
	sb.WriteString(fmt.Sprintf("🔢 <b>Всего занятий: %d</b>\n", lessonCount))

	// Находим общую продолжительность занятий
	var totalDuration int
	for _, s := range schedules {
		totalDuration += s.Duration
	}
	sb.WriteString(fmt.Sprintf("⌛ <b>Общая продолжительность: %d мин (%d ч %d мин)</b>\n\n",
		totalDuration, totalDuration/60, totalDuration%60))

	sb.WriteString("✨ <i>Пусть день пройдет продуктивно!</i>")

	return sb.String()
}

// ShowScheduleWeek отправляет расписание за выбранную неделю.
// weekStart – дата понедельника недели, которую надо показать.
func ShowScheduleWeek(chatID int64, bot *tgbotapi.BotAPI, user *models.User, weekStart time.Time) error {
	weekEnd := weekStart.AddDate(0, 0, 6)

	fmt.Printf("ShowScheduleWeek for user %+v, weekStart: %s\n", user, weekStart.Format("2006-01-02"))

	var schedules []models.Schedule
	var err error
	if user.Role == "teacher" {
		schedules, err = GetSchedulesForTeacherByDateRange(user.RegistrationCode, weekStart, weekEnd)
	} else {
		schedules, err = GetSchedulesForGroupByDateRange(user.Group, weekStart, weekEnd)
	}
	if err != nil {
		// Return a clear error message for weekly schedule display
		errMsg := "Ошибка отображения недельного расписания: " + err.Error()
		msg := tgbotapi.NewMessage(chatID, errMsg)
		return sendAndTrackMessage(bot, msg)
	}

	// Получаем информацию о фильтрах
	filter := GetUserFilter(chatID)
	filteredSchedules := ApplyFilters(schedules, filter)

	text := FormatSchedulesByWeek(filteredSchedules, weekStart, weekEnd, user.Role, user)

	// Если фильтры активны, добавляем информацию о них в сообщение
	if filter.CourseName != "" || filter.LessonType != "" {
		filterInfo := "\n<b>📌 Активные фильтры:</b>\n"
		if filter.CourseName != "" {
			filterInfo += fmt.Sprintf("• Курс: <b>%s</b>\n", filter.CourseName)
		}
		if filter.LessonType != "" {
			filterInfo += fmt.Sprintf("• Тип занятия: <b>%s</b>\n", filter.LessonType)
		}
		text = text + filterInfo
	}

	// Создаём базовую клавиатуру
	baseRows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("◄", fmt.Sprintf("week_prev_%s", weekStart.AddDate(0, 0, -7).Format("2006-01-02"))),
			tgbotapi.NewInlineKeyboardButtonData("Сегодня", "week_today"),
			tgbotapi.NewInlineKeyboardButtonData("►", fmt.Sprintf("week_next_%s", weekStart.AddDate(0, 0, 7).Format("2006-01-02"))),
		},
		{}, // Пустая строка для разделения
	}

	// Добавляем отдельную кнопку фильтров
	filterRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔍 Настроить фильтры", "filter_menu"),
	)
	baseRows = append(baseRows, filterRow)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(baseRows...)

	// Объединяем с клавиатурой режимов
	modeKeyboard := BuildModeSwitchKeyboard("mode_week")
	if modeKeyboard.InlineKeyboard != nil {
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, modeKeyboard.InlineKeyboard...)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	return sendAndTrackMessage(bot, msg)
}

// УДАЛЯЕМ дублирующую функцию ShowEnhancedScheduleDay, она уже определена в schedule_day.go

func GetSchedulesForTeacherByDateRange(teacherRegCode string, start, end time.Time) ([]models.Schedule, error) {
	query := `
        SELECT
            s.id, s.course_id, s.group_name, s.teacher_reg_code,
            s.schedule_time, s.description, s.auditory, s.lesson_type, s.duration,
            COALESCE(u.name, s.teacher_reg_code) AS teacher_name,
            COALESCE(c.name, 'Неизвестный курс') AS course_name
        FROM schedules s
        LEFT JOIN users u ON s.teacher_reg_code = u.registration_code
        LEFT JOIN courses c ON s.course_id = c.id
        WHERE s.teacher_reg_code = ? AND date(s.schedule_time) BETWEEN ? AND ?
        ORDER BY s.schedule_time
    `

	// Debug log
	fmt.Printf("GetSchedulesForTeacherByDateRange: %s, %s, %s\n",
		teacherRegCode, start.Format("2006-01-02"), end.Format("2006-01-02"))

	rows, err := db.DB.Query(query, teacherRegCode, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		fmt.Printf("Database query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var scheduleTimeStr string
		var teacherName, courseName string
		if err := rows.Scan(
			&s.ID,
			&s.CourseID,
			&s.GroupName,
			&s.TeacherRegCode,
			&scheduleTimeStr,
			&s.Description,
			&s.Auditory,
			&s.LessonType,
			&s.Duration,
			&teacherName,
			&courseName,
		); err != nil {
			fmt.Printf("Row scan error: %v\n", err)
			return nil, err
		}

		// Try to parse the time with error handling
		if parsedTime, err := time.Parse(time.RFC3339, scheduleTimeStr); err != nil {
			fmt.Printf("Time parse error: %v for string %s\n", err, scheduleTimeStr)

			// Try alternative formats as fallback
			layouts := []string{"2006-01-02T15:04:05Z", "2006-01-02 15:04:05", "2006-01-02T15:04:05"}
			parsed := false

			for _, layout := range layouts {
				if parsedTime, err := time.Parse(layout, scheduleTimeStr); err == nil {
					s.ScheduleTime = parsedTime
					parsed = true
					break
				}
			}

			if !parsed {
				// Use current time as last resort to avoid crashing
				s.ScheduleTime = time.Now()
			}
		} else {
			s.ScheduleTime = parsedTime
		}

		// Store teacher name and course name in context
		s.TeacherRegCode = teacherName // Use teacher name instead of reg code

		// Only prepend course name if it's a valid course
		if courseName != "Неизвестный курс" {
			s.Description = courseName + ": " + s.Description
		}

		schedules = append(schedules, s)
	}

	fmt.Printf("Found %d schedules for teacher %s\n", len(schedules), teacherRegCode)
	return schedules, rows.Err()
}

func GetSchedulesForGroupByDateRange(group string, start, end time.Time) ([]models.Schedule, error) {
	query := `
       SELECT
         s.id, s.course_id, s.group_name, s.teacher_reg_code,
         s.schedule_time, s.description, s.auditory, s.lesson_type, s.duration,
         COALESCE(u.name, s.teacher_reg_code) AS teacher_name,
         COALESCE(c.name, 'Неизвестный курс') AS course_name
       FROM schedules s
       LEFT JOIN users u ON s.teacher_reg_code = u.registration_code
       LEFT JOIN courses c ON s.course_id = c.id
       WHERE s.group_name = ? AND date(s.schedule_time) BETWEEN ? AND ?
       ORDER BY s.schedule_time
    `

	// Debug log
	fmt.Printf("GetSchedulesForGroupByDateRange: %s, %s, %s\n",
		group, start.Format("2006-01-02"), end.Format("2006-01-02"))

	rows, err := db.DB.Query(query, group, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		fmt.Printf("Database query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var scheduleTimeStr string
		var teacherName, courseName string
		if err := rows.Scan(
			&s.ID,
			&s.CourseID,
			&s.GroupName,
			&s.TeacherRegCode,
			&scheduleTimeStr,
			&s.Description,
			&s.Auditory,
			&s.LessonType,
			&s.Duration,
			&teacherName,
			&courseName,
		); err != nil {
			fmt.Printf("Row scan error: %v\n", err)
			return nil, err
		}

		// Try to parse the time with error handling
		if parsedTime, err := time.Parse(time.RFC3339, scheduleTimeStr); err != nil {
			fmt.Printf("Time parse error: %v for string %s\n", err, scheduleTimeStr)

			// Try alternative formats as fallback
			layouts := []string{"2006-01-02T15:04:05Z", "2006-01-02 15:04:05", "2006-01-02T15:04:05"}
			parsed := false

			for _, layout := range layouts {
				if parsedTime, err := time.Parse(layout, scheduleTimeStr); err == nil {
					s.ScheduleTime = parsedTime
					parsed = true
					break
				}
			}

			if !parsed {
				// Use current time as last resort to avoid crashing
				s.ScheduleTime = time.Now()
			}
		} else {
			s.ScheduleTime = parsedTime
		}

		// Store teacher name and course name in context
		s.TeacherRegCode = teacherName // Use teacher name instead of reg code

		// Only prepend course name if it's a valid course
		if courseName != "Неизвестный курс" {
			s.Description = courseName + ": " + s.Description
		}

		schedules = append(schedules, s)
	}

	fmt.Printf("Found %d schedules for group %s\n", len(schedules), group)
	return schedules, rows.Err()
}

// GetSchedulesByTeacher возвращает расписание преподавателя, учитывая все поля структуры Schedule.
func GetSchedulesByTeacher(teacherRegCode string) ([]models.Schedule, error) {
	rows, err := db.DB.Query(`
		SELECT
			s.id,
			s.course_id,
			s.group_name,
			s.teacher_reg_code,
			s.schedule_time,
			s.description,
			s.auditory,
			s.lesson_type,
			s.duration,
			COALESCE(u.name, s.teacher_reg_code) AS teacher_name,
			COALESCE(c.name, 'Неизвестный курс') AS course_name
		FROM schedules s
		LEFT JOIN users u ON s.teacher_reg_code = u.registration_code
		LEFT JOIN courses c ON s.course_id = c.id
		WHERE s.teacher_reg_code = ?
		ORDER BY s.schedule_time
	`, teacherRegCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var scheduleTimeStr string
		var teacherName, courseName string
		if err := rows.Scan(
			&s.ID,
			&s.CourseID,
			&s.GroupName,
			&s.TeacherRegCode,
			&scheduleTimeStr,
			&s.Description,
			&s.Auditory,
			&s.LessonType,
			&s.Duration,
			&teacherName,
			&courseName,
		); err != nil {
			return nil, err
		}

		s.ScheduleTime, err = time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return nil, err
		}

		// Store teacher name and course name in context
		s.TeacherRegCode = teacherName

		// Only prepend course name if it's a valid course
		if courseName != "Неизвестный курс" {
			s.Description = courseName + ": " + s.Description
		}

		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

// GetSchedulesByGroup возвращает расписание для указанной группы, учитывая все поля структуры Schedule.
func GetSchedulesByGroup(group string) ([]models.Schedule, error) {
	rows, err := db.DB.Query(`
		SELECT
			s.id,
			s.course_id,
			s.group_name,
			s.teacher_reg_code,
			s.schedule_time,
			s.description,
			s.auditory,
			s.lesson_type,
			s.duration,
			COALESCE(u.name, s.teacher_reg_code) AS teacher_name,
			COALESCE(c.name, 'Неизвестный курс') AS course_name
		FROM schedules s
		LEFT JOIN users u ON s.teacher_reg_code = u.registration_code
		LEFT JOIN courses c ON s.course_id = c.id
		WHERE s.group_name = ?
		ORDER BY s.schedule_time
	`, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var scheduleTimeStr string
		var teacherName, courseName string
		if err := rows.Scan(
			&s.ID,
			&s.CourseID,
			&s.GroupName,
			&s.TeacherRegCode,
			&scheduleTimeStr,
			&s.Description,
			&s.Auditory,
			&s.LessonType,
			&s.Duration,
			&teacherName,
			&courseName,
		); err != nil {
			return nil, err
		}

		s.ScheduleTime, err = time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return nil, err
		}

		// Store teacher name and course name in context
		s.TeacherRegCode = teacherName

		// Only prepend course name if it's a valid course
		if courseName != "Неизвестный курс" {
			s.Description = courseName + ": " + s.Description
		}

		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func FormatSchedulesByWeek(
	schedules []models.Schedule,
	weekStart, weekEnd time.Time,
	mode string,
	user *models.User,
) string {
	if len(schedules) == 0 {
		return fmt.Sprintf("📆 <b>Неделя %s – %s</b>\n\n🔍 <i>Нет занятий на эту неделю</i>",
			weekStart.Format("02.01.2006"), weekEnd.Format("02.01.2006"))
	}

	// Группировка по дням
	grouped := make(map[string][]models.Schedule)
	for _, s := range schedules {
		date := s.ScheduleTime.Format("2006-01-02")
		grouped[date] = append(grouped[date], s)
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("📆 <b>Неделя %s – %s</b>\n\n",
		weekStart.Format("02.01.2006"), weekEnd.Format("02.01.2006")))

	// Проходим по дням недели
	hasEvents := false
	for d := 0; d < 7; d++ {
		day := weekStart.AddDate(0, 0, d)
		dayStr := day.Format("2006-01-02")

		if entries, ok := grouped[dayStr]; ok {
			hasEvents = true

			// Заголовок дня
			msg.WriteString(fmt.Sprintf("🗓 <b>%s (%s)</b>\n",
				day.Format("02.01.2006"), weekdayName(day.Weekday())))
			msg.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

			// Сортируем занятия по времени
			sort.Slice(entries, func(i, j int) bool {
				return entries[i].ScheduleTime.Before(entries[j].ScheduleTime)
			})

			for _, s := range entries {
				timeStr := s.ScheduleTime.Format("15:04")
				endTimeStr := s.ScheduleTime.Add(time.Duration(s.Duration) * time.Minute).Format("15:04")

				// Полный блок информации о занятии
				msg.WriteString(fmt.Sprintf("\n⏰ <b>%s - %s</b> (%d мин.)\n", timeStr, endTimeStr, s.Duration))
				msg.WriteString(fmt.Sprintf("📚 <b>%s</b>\n", s.Description))

				if mode == "teacher" {
					msg.WriteString(fmt.Sprintf("👥 Группа: %s\n", s.GroupName))
				} else {
					msg.WriteString(fmt.Sprintf("👨‍🏫 Преподаватель: %s\n", s.TeacherRegCode))
				}

				msg.WriteString(fmt.Sprintf("🚪 Аудитория: %s\n", s.Auditory))
				msg.WriteString(fmt.Sprintf("📝 Тип: %s\n", s.LessonType))
			}
			msg.WriteString("\n")
		}
	}

	if !hasEvents {
		msg.WriteString("🔍 <i>Нет занятий на эту неделю</i>\n")
	} else {
		// Добавляем позитивное завершение
		msg.WriteString("\n<i>✨ Удачной и продуктивной недели!</i>")
	}

	return msg.String()
}

// BuildWeekNavigationKeyboardFiltered строит клавиатуру недели, отображая кнопки только для дней, где есть события.
// Если в день нет событий, кнопка выводится с префиксом "❌" и callback_data="ignore".
func BuildWeekNavigationKeyboardFiltered(weekStart time.Time, schedules []models.Schedule) tgbotapi.InlineKeyboardMarkup {
	eventDays := make(map[string]bool)
	for _, s := range schedules {
		eventDays[s.ScheduleTime.Format("02.01")] = true
	}

	prevWeek := weekStart.AddDate(0, 0, -7)
	nextWeek := weekStart.AddDate(0, 0, 7)

	var dayRow []tgbotapi.InlineKeyboardButton
	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		dayStr := day.Format("02.01")
		label := dayStr
		if !eventDays[dayStr] {
			label = "—" + dayStr
		}
		dayRow = append(dayRow, tgbotapi.NewInlineKeyboardButtonData(
			label,
			fmt.Sprintf("day_%s", day.Format("2006-01-02")),
		))
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◄", fmt.Sprintf("week_prev_%s", prevWeek.Format("2006-01-02"))),
			tgbotapi.NewInlineKeyboardButtonData("Сегодня", "week_today"),
			tgbotapi.NewInlineKeyboardButtonData("►", fmt.Sprintf("week_next_%s", nextWeek.Format("2006-01-02"))),
		),
		dayRow,
	)
}

func BuildWeekNavigationKeyboardFilteredWithFilter(weekStart time.Time, schedules []models.Schedule) tgbotapi.InlineKeyboardMarkup {
	// Строим обычную клавиатуру с кнопками для дней недели (как в BuildWeekNavigationKeyboardFiltered)
	weekKeyboard := BuildWeekNavigationKeyboardFiltered(weekStart, schedules)
	// Дополнительный ряд кнопок с фильтрами (пример)
	filterRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Фильтр: Математика", "filter_course_Математика"),
		tgbotapi.NewInlineKeyboardButtonData("Фильтр: Физика", "filter_course_Физика"),
	)
	// Объединяем ряды (напрямую или добавляя новый ряд в существующую клавиатуру)
	allRows := weekKeyboard.InlineKeyboard
	allRows = append(allRows, filterRow)
	return tgbotapi.NewInlineKeyboardMarkup(allRows...)
}

func ShowScheduleModeMenu(chatID int64, bot *tgbotapi.BotAPI) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("День", "mode_day"),
			tgbotapi.NewInlineKeyboardButtonData("Неделя", "mode_week"),
			// Удаляем опцию "Месяц"
		),
		// Добавляем строку с кнопкой фильтров
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Фильтры", "filter_course_menu"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "Выберите режим отображения расписания:")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	return sendAndTrackMessage(bot, msg)
}

// Улучшенная версия BuildCalendarTimeline
func BuildCalendarTimeline(schedules []models.Schedule, day time.Time) string {
	if len(schedules) == 0 {
		return fmt.Sprintf("📆 <b>%s</b>\n\n🔍 <i>Нет занятий на этот день</i>",
			day.Format("02.01.2006")+" ("+weekdayName(day.Weekday())+")")
	}

	// Сортируем занятия по времени
	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].ScheduleTime.Before(schedules[j].ScheduleTime)
	})

	var sb strings.Builder
	// Заголовок с датой и днем недели
	sb.WriteString(fmt.Sprintf("📆 <b>%s</b>\n\n",
		day.Format("02.01.2006")+" ("+weekdayName(day.Weekday())+")"))

	// Сначала фильтруем только расписание для указанного дня
	dayStart := day.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)

	// Разделитель заголовка
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	seen := make(map[string]bool)
	hasEvents := false

	for _, s := range schedules {
		// Проверяем, что занятие относится к запрошенному дню
		if s.ScheduleTime.Before(dayStart) || s.ScheduleTime.After(dayEnd) {
			continue
		}

		hasEvents = true

		// Улучшенный ключ для проверки уникальности
		key := fmt.Sprintf("%s-%d-%s-%s-%s-%s",
			s.ScheduleTime.Format("15:04"),
			s.CourseID,
			s.Description,
			s.GroupName,
			s.Auditory,
			s.LessonType,
		)
		if seen[key] {
			continue
		}
		seen[key] = true

		timeStr := s.ScheduleTime.Format("15:04")
		endTimeStr := s.ScheduleTime.Add(time.Duration(s.Duration) * time.Minute).Format("15:04")

		// Блок информации о занятии
		sb.WriteString(fmt.Sprintf("⏰ <b>%s - %s</b> (%d мин.)\n", timeStr, endTimeStr, s.Duration))
		sb.WriteString(fmt.Sprintf("📚 <b>%s</b>\n", s.Description))
		sb.WriteString(fmt.Sprintf("👨‍🏫 Преподаватель: %s\n", s.TeacherRegCode))
		sb.WriteString(fmt.Sprintf("👥 Группа: %s\n", s.GroupName))
		sb.WriteString(fmt.Sprintf("🚪 Аудитория: %s\n", s.Auditory))
		sb.WriteString(fmt.Sprintf("📝 Тип: %s\n", s.LessonType))
		sb.WriteString("\n")
	}

	if !hasEvents {
		sb.WriteString("🔍 <i>Нет занятий на этот день</i>\n")
	}

	return sb.String()
}

// weekdayName возвращает название дня недели на русском языке.
func weekdayName(wd time.Weekday) string {
	switch wd {
	case time.Monday:
		return "Понедельник"
	case time.Tuesday:
		return "Вторник"
	case time.Wednesday:
		return "Среда"
	case time.Thursday:
		return "Четверг"
	case time.Friday:
		return "Пятница"
	case time.Saturday:
		return "Суббота"
	case time.Sunday:
		return "Воскресенье"
	default:
		return ""
	}
}
