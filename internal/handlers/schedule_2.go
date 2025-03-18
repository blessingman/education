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

// GetSchedulesByTeacher возвращает расписание преподавателя, учитывая все поля структуры Schedule.
func GetSchedulesByTeacher(teacherRegCode string) ([]models.Schedule, error) {
	rows, err := db.DB.Query(`
		SELECT
			id,
			course_id,
			group_name,
			teacher_reg_code,
			schedule_time,
			description,
			auditory,
			lesson_type,
			duration
		FROM schedules
		WHERE teacher_reg_code = ?
		ORDER BY schedule_time
	`, teacherRegCode)
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

// GetSchedulesByGroup возвращает расписание для указанной группы, учитывая все поля структуры Schedule.
func GetSchedulesByGroup(group string) ([]models.Schedule, error) {
	rows, err := db.DB.Query(`
		SELECT
			id,
			course_id,
			group_name,
			teacher_reg_code,
			schedule_time,
			description,
			auditory,
			lesson_type,
			duration
		FROM schedules
		WHERE group_name = ?
		ORDER BY schedule_time
	`, group)
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
