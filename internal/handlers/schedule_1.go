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

// ShowScheduleWeek отправляет расписание за выбранную неделю.
// weekStart – дата понедельника недели, которую надо показать.
func ShowScheduleWeek(chatID int64, bot *tgbotapi.BotAPI, user *models.User, weekStart time.Time) error {
	weekEnd := weekStart.AddDate(0, 0, 6)

	var schedules []models.Schedule
	var err error
	if user.Role == "teacher" {
		schedules, err = GetSchedulesForTeacherByDateRange(user.RegistrationCode, weekStart, weekEnd)
	} else {
		schedules, err = GetSchedulesForGroupByDateRange(user.Group, weekStart, weekEnd)
	}
	if err != nil {
		return err
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
            id, course_id, group_name, teacher_reg_code,
            schedule_time, description, auditory, lesson_type, duration
        FROM schedules
        WHERE teacher_reg_code = ? AND date(schedule_time) BETWEEN ? AND ?
        ORDER BY schedule_time
    `
	rows, err := db.DB.Query(query, teacherRegCode, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var scheduleTimeStr string
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
		); err != nil {
			return nil, err
		}
		s.ScheduleTime, err = time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func GetSchedulesForGroupByDateRange(group string, start, end time.Time) ([]models.Schedule, error) {
	query := `
       SELECT
         id, course_id, group_name, teacher_reg_code,
         schedule_time, description, auditory, lesson_type, duration
       FROM schedules
       WHERE group_name = ? AND date(schedule_time) BETWEEN ? AND ?
       ORDER BY schedule_time
    `
	rows, err := db.DB.Query(query, group, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var scheduleTimeStr string
		if err := rows.Scan(
			&s.ID,
			&s.CourseID,
			&s.GroupName,
			&s.TeacherRegCode,
			&scheduleTimeStr,
			&s.Description,
			&s.Auditory,
			&s.LessonType,
			&s.Duration, // <-- duration
		); err != nil {
			return nil, err
		}
		s.ScheduleTime, err = time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return nil, err
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
