package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ShowScheduleWeek отправляет расписание за выбранную неделю.
// weekStart – дата понедельника недели, которую надо показать.
func ShowScheduleWeek(chatID int64, bot *tgbotapi.BotAPI, user *models.User, weekStart time.Time) error {
	weekEnd := weekStart.AddDate(0, 0, 6) // до воскресенья включительно

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

	// Форматируем расписание за неделю (группировка по дням)
	text := FormatSchedulesByWeek(schedules, weekStart, weekEnd, user.Role, user)
	// Формируем клавиатуру навигации по неделям и дням
	keyboard := BuildWeekNavigationKeyboard(weekStart)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	return sendAndTrackMessage(bot, msg)
}

// ShowScheduleDay выводит расписание за конкретный день.
// Принимает chatID, bot, пользователя и выбранный день (time.Time).
func ShowScheduleDay(chatID int64, bot *tgbotapi.BotAPI, user *models.User, day time.Time) error {
	// Определяем начало и конец дня
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

	// Форматируем расписание для одного дня.
	// Здесь можно использовать функцию группировки по дням – при одном дне она вернет один блок.
	text := FormatSchedulesGroupedByDay(schedules, 1, 1, user.Role, user)

	// Создаем клавиатуру с кнопкой "Назад к неделе".
	// Вычисляем начало недели для выбранного дня (предполагается, что неделя начинается с понедельника).
	offset := int(dayStart.Weekday())
	// Если Sunday (0) – приводим к 7, чтобы понедельник был первым
	if offset == 0 {
		offset = 7
	}
	weekStart := dayStart.AddDate(0, 0, -(offset - 1))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад к неделе", fmt.Sprintf("week_prev_%s", weekStart.Format("2006-01-02"))),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard

	return sendAndTrackMessage(bot, msg)
}

func GetSchedulesForTeacherByDateRange(teacherRegCode string, start, end time.Time) ([]models.Schedule, error) {
	query := `
       SELECT id, course_id, group_name, teacher_reg_code, schedule_time, description
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
		if err := rows.Scan(&s.ID, &s.CourseID, &s.GroupName, &s.TeacherRegCode, &scheduleTimeStr, &s.Description); err != nil {
			return nil, err
		}
		s.ScheduleTime, _ = time.Parse(time.RFC3339, scheduleTimeStr)
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func GetSchedulesForGroupByDateRange(group string, start, end time.Time) ([]models.Schedule, error) {
	query := `
       SELECT id, course_id, group_name, teacher_reg_code, schedule_time, description
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
		if err := rows.Scan(&s.ID, &s.CourseID, &s.GroupName, &s.TeacherRegCode, &scheduleTimeStr, &s.Description); err != nil {
			return nil, err
		}
		s.ScheduleTime, _ = time.Parse(time.RFC3339, scheduleTimeStr)
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func FormatSchedulesByWeek(schedules []models.Schedule, weekStart, weekEnd time.Time, mode string, user *models.User) string {
	msgText := fmt.Sprintf("<b>Расписание на неделю (%s - %s)</b>\n\n",
		weekStart.Format("02.01.2006"), weekEnd.Format("02.01.2006"))

	// Группируем занятия по датам
	type dayKey string
	grouped := make(map[dayKey][]models.Schedule)
	for _, s := range schedules {
		dateOnly := s.ScheduleTime.Format("2006-01-02")
		grouped[dayKey(dateOnly)] = append(grouped[dayKey(dateOnly)], s)
	}

	// Для каждого дня недели от weekStart до weekEnd
	for d := 0; d < 7; d++ {
		day := weekStart.AddDate(0, 0, d)
		dayStr := day.Format("2006-01-02")
		dayName := weekdayName(day.Weekday())
		msgText += fmt.Sprintf("📅 <b>%s (%s)</b>\n", day.Format("02.01.2006"), dayName)
		if entries, ok := grouped[dayKey(dayStr)]; ok {
			for _, s := range entries {
				timeStr := s.ScheduleTime.Format("15:04")
				// Здесь можно добавить получение courseMap и teacherMap, аналогично предыдущим функциям
				// Для простоты, оставим как есть:
				if mode == "teacher" {
					msgText += fmt.Sprintf("   • <i>%s</i>: %s (группа: %s)\n", timeStr, s.Description, s.GroupName)
				} else {
					msgText += fmt.Sprintf("   • <i>%s</i>: %s (Преп.: %s)\n", timeStr, s.Description, s.TeacherRegCode)
				}
			}
		} else {
			msgText += "   Нет занятий.\n"
		}
		msgText += "\n"
	}
	msgText += "<i>Планируйте свою неделю с умом!</i>"
	return msgText
}
