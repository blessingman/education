package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
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
	// Определяем множество дат, в которых есть занятия
	eventDays := make(map[string]bool)
	for _, s := range schedules {
		dayStr := s.ScheduleTime.Format("2006-01-02")
		eventDays[dayStr] = true
	}

	// Кнопки для перехода на предыдущую и следующую неделю
	prevWeek := weekStart.AddDate(0, 0, -7)
	nextWeek := weekStart.AddDate(0, 0, 7)
	navRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Пред. неделя", fmt.Sprintf("week_prev_%s", prevWeek.Format("2006-01-02"))),
		tgbotapi.NewInlineKeyboardButtonData("След. неделя ▶️", fmt.Sprintf("week_next_%s", nextWeek.Format("2006-01-02"))),
	)

	// Кнопки с названиями дней недели
	dayNames := []string{"П", "В", "С", "Ч", "П", "С", "В"}
	var dayRow []tgbotapi.InlineKeyboardButton
	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		dayStr := day.Format("2006-01-02")
		if eventDays[dayStr] {
			// Если в этот день есть событие – обычная кнопка
			dayRow = append(dayRow, tgbotapi.NewInlineKeyboardButtonData(dayNames[i], fmt.Sprintf("day_%s", dayStr)))
		} else {
			// Если нет – выводим кнопку с "❌" и callback "ignore"
			dayRow = append(dayRow, tgbotapi.NewInlineKeyboardButtonData("❌"+dayNames[i], "ignore"))
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(navRow, dayRow)
	return keyboard
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
