package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"sort"
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

	text := FormatSchedulesByWeek(schedules, weekStart, weekEnd, user.Role, user)
	// Вместо BuildWeekNavigationKeyboard(...) используем нашу новую функцию
	keyboard := BuildWeekNavigationKeyboardFiltered(weekStart, schedules)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	return sendAndTrackMessage(bot, msg)
}

// ShowScheduleDay выводит расписание за конкретный день.
// Принимает chatID, bot, пользователя и выбранный день (time.Time).
func ShowScheduleDay(chatID int64, bot *tgbotapi.BotAPI, user *models.User, day time.Time) error {
	dayStr := day.Format("2006-01-02")
	dayStart, _ := time.Parse("2006-01-02", dayStr)
	dayEnd := dayStart.AddDate(0, 0, 1).Add(-time.Second) // последний момент выбранного дня

	var schedules []models.Schedule
	var err error
	if user.Role == "teacher" {
		schedules, err = GetSchedulesForTeacherByDateRange(user.RegistrationCode, dayStart, dayEnd)
	} else {
		schedules, err = GetSchedulesForGroupByDateRange(user.Group, dayStart, dayEnd)
	}
	if err != nil {
		return err
	}

	// Формируем текст расписания за этот день
	text := FormatSchedulesGroupedByDay(schedules, 1, 1, user.Role, user)

	// Клавиатура:
	// 1) "← Нед." (назад к неделе)
	// 2) (Опционально) фильтры или редактирование (зависит от ваших нужд)

	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	// Вычисляем начало недели (предполагаем, что неделя начинается с понедельника)
	offset := int(dayStart.Weekday())
	if offset == 0 {
		offset = 7
	}
	weekStart := dayStart.AddDate(0, 0, -(offset - 1))

	backLabel := "← Нед."
	backData := fmt.Sprintf("week_prev_%s", weekStart.Format("2006-01-02"))
	backButton := tgbotapi.NewInlineKeyboardButtonData(backLabel, backData)

	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(backButton))

	// Пример: можно добавить фильтр по курсу, редактирование, и т. д.
	// filterButton := tgbotapi.NewInlineKeyboardButtonData("🔎 Фильтр", "filter_day") // Пример
	// keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(filterButton))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard

	return sendAndTrackMessage(bot, msg)
}

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
			&s.Duration, // <-- Считываем duration
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
	total := len(schedules)
	passed := 0
	now := time.Now()
	for _, s := range schedules {
		if s.ScheduleTime.Before(now) {
			passed++
		}
	}
	progressPercent := 0
	if total > 0 {
		progressPercent = (passed * 100) / total
	}

	// Заголовок недели + прогресс
	msgText := fmt.Sprintf(
		"🗓 <b>Неделя %s – %s</b>\nПрогресс: <b>%d%%</b>\n",
		weekStart.Format("02.01.2006"), weekEnd.Format("02.01.2006"), progressPercent,
	)
	msgText += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"

	// Группируем по дате
	type dayKey string
	grouped := make(map[dayKey][]models.Schedule)
	for _, s := range schedules {
		dateOnly := s.ScheduleTime.Format("2006-01-02")
		grouped[dayKey(dateOnly)] = append(grouped[dayKey(dateOnly)], s)
	}

	// Сортируем даты
	var sortedDates []string
	for k := range grouped {
		sortedDates = append(sortedDates, string(k))
	}
	sort.Strings(sortedDates)

	// Идём по каждому дню (понедельник–воскресенье)
	for d := 0; d < 7; d++ {
		day := weekStart.AddDate(0, 0, d)
		dayStr := day.Format("2006-01-02")
		dayName := weekdayName(day.Weekday())

		// Заголовок дня
		msgText += fmt.Sprintf("🗓 <b>%s (%s)</b>\n", day.Format("02.01.2006"), dayName)
		msgText += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

		if entries, ok := grouped[dayKey(dayStr)]; ok {
			for _, s := range entries {
				timeStr := s.ScheduleTime.Format("15:04")
				if mode == "teacher" {
					msgText += fmt.Sprintf(
						"  • <b>%s</b> — %s\n    👥 Гр.: %s, 🚪 Ауд.: %s, 📋 %s, ⏱ %d мин.\n",
						timeStr, s.Description, s.GroupName, s.Auditory, s.LessonType, s.Duration,
					)
				} else {
					msgText += fmt.Sprintf(
						"  • <b>%s</b> — %s\n    👨‍🏫 Преп.: %s, 🚪 Ауд.: %s, 📋 %s, ⏱ %d мин.\n",
						timeStr, s.Description, s.TeacherRegCode, s.Auditory, s.LessonType, s.Duration,
					)
				}
			}
		} else {
			msgText += "  Нет занятий.\n"
		}
		msgText += "\n"
	}

	msgText += "<i>Планируйте неделю с умом!</i>"
	return msgText
}
