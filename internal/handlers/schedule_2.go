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

	// Дата предыдущей и следующей недели
	prevWeek := weekStart.AddDate(0, 0, -7)
	nextWeek := weekStart.AddDate(0, 0, 7)

	// Кнопки "⬅️ -1 нед" и "+1 нед ➡️"
	navRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ -1 нед", fmt.Sprintf("week_prev_%s", prevWeek.Format("2006-01-02"))),
		tgbotapi.NewInlineKeyboardButtonData("+1 нед ➡️", fmt.Sprintf("week_next_%s", nextWeek.Format("2006-01-02"))),
	)

	// Сокращённые названия дней: Пн, Вт, Ср, Чт, Пт, Сб, Вс
	dayNames := []string{"П", "В", "С", "Ч", "П", "С", "В"}

	var dayRow []tgbotapi.InlineKeyboardButton
	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		dayStr := day.Format("2006-01-02")

		// Если есть занятия – обычная кнопка, иначе "❌"
		if eventDays[dayStr] {
			dayLabel := fmt.Sprintf("%s %s", dayNames[i], day.Format("02"))
			dayRow = append(dayRow, tgbotapi.NewInlineKeyboardButtonData(dayLabel, fmt.Sprintf("day_%s", dayStr)))
		} else {
			dayLabel := fmt.Sprintf("❌%s", dayNames[i])
			dayRow = append(dayRow, tgbotapi.NewInlineKeyboardButtonData(dayLabel, "ignore"))
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

func ShowScheduleModeMenu(chatID int64, bot *tgbotapi.BotAPI) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("День", "mode_day"),
			tgbotapi.NewInlineKeyboardButtonData("Неделя", "mode_week"),
			tgbotapi.NewInlineKeyboardButtonData("Месяц", "mode_month"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "Выберите режим отображения расписания:")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	return sendAndTrackMessage(bot, msg)
}

func BuildMinimalTimeline(slots []string, currentTime time.Time, events map[string]string) string {
	// slots – временные метки, например: ["08:00", "10:00", "12:00", "14:00", "16:00", "18:00"]
	// events – карта: ключ – временная метка (например, "08:00"), значение – название события (например, "Математика (Иванов)")
	// currentTime – текущее время

	// Первая строка: временные метки
	timeline := "⏰ "
	for _, slot := range slots {
		timeline += fmt.Sprintf("%-12s", slot)
	}

	// Вторая строка: линия с квадратными метками
	line := ""
	// Заполняем каждый слот линией: если в слоте есть событие – обычный квадрат, если это текущий интервал – выделенный квадрат.
	for i, slot := range slots {
		// Определяем, попадает ли текущее время в интервал между данным слотом и следующим
		var marker string
		// Если в данный слот есть событие (в events) — ставим квадрат,
		// иначе выводим разделительную линию.
		if _, ok := events[slot]; ok {
			marker = "🔲"
		} else {
			marker = "────────────"
		}

		// Если есть следующий слот, проверяем, находится ли текущее время в интервале.
		if i < len(slots)-1 {
			t1, err1 := time.Parse("15:04", slot)
			t2, err2 := time.Parse("15:04", slots[i+1])
			if err1 == nil && err2 == nil {
				// Прикрепляем текущую дату к слоту
				t1 = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), t1.Hour(), t1.Minute(), 0, 0, currentTime.Location())
				t2 = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), t2.Hour(), t2.Minute(), 0, 0, currentTime.Location())
				if currentTime.After(t1) && currentTime.Before(t2) {
					marker = "🔳"
				}
			}
		}
		line += fmt.Sprintf("%-12s", marker)
	}

	// Третья строка: события (если есть)
	eventsLine := ""
	for _, slot := range slots {
		if ev, ok := events[slot]; ok {
			eventsLine += fmt.Sprintf("%-12s", ev)
		} else {
			eventsLine += fmt.Sprintf("%-12s", "")
		}
	}

	// Определяем, когда начнется следующее занятие
	var nextSlot string
	for _, slot := range slots {
		t, err := time.Parse("15:04", slot)
		if err != nil {
			continue
		}
		t = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), t.Hour(), t.Minute(), 0, 0, currentTime.Location())
		if currentTime.Before(t) {
			nextSlot = slot
			break
		}
	}
	var remaining string
	if nextSlot != "" {
		tNext, _ := time.Parse("15:04", nextSlot)
		tNext = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), tNext.Hour(), tNext.Minute(), 0, 0, currentTime.Location())
		minutesLeft := int(tNext.Sub(currentTime).Minutes())
		remaining = fmt.Sprintf("Сейчас: %s (до следующего занятия осталось %d мин.)", currentTime.Format("15:04"), minutesLeft)
	} else {
		remaining = fmt.Sprintf("Сейчас: %s", currentTime.Format("15:04"))
	}

	result := timeline + "\n" + line + "\n" + eventsLine + "\n\n" + remaining
	return result
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
