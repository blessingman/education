package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// GetSchedulesByTeacher возвращает расписание преподавателя, сгруппированное по курсам и группам.
func GetSchedulesByTeacher(teacherRegCode string) ([]models.Schedule, error) {
	rows, err := db.DB.Query(`
		SELECT id, course_id, group_name, teacher_reg_code, schedule_time, description
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
		if err := rows.Scan(&s.ID, &s.CourseID, &s.GroupName, &s.TeacherRegCode, &scheduleTimeStr, &s.Description); err != nil {
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

// GetSchedulesByGroup возвращает расписание для указанной группы (уже есть в коде, но включим для полноты).
func GetSchedulesByGroup(group string) ([]models.Schedule, error) {
	rows, err := db.DB.Query(`
		SELECT id, course_id, group_name, teacher_reg_code, schedule_time, description
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
		if err := rows.Scan(&s.ID, &s.CourseID, &s.GroupName, &s.TeacherRegCode, &scheduleTimeStr, &s.Description); err != nil {
			return nil, err
		}
		s.ScheduleTime, _ = time.Parse(time.RFC3339, scheduleTimeStr)
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

// FormatSchedulesByWeek форматирует расписание по дням недели для удобного отображения.
func FormatSchedulesByWeek(schedules []models.Schedule, mode string, user *models.User) (string, error) {
	if len(schedules) == 0 {
		return "Расписание не найдено.", nil
	}

	// Карта для хранения расписания по дням недели
	scheduleByDay := make(map[time.Weekday][]string)

	// Получаем дополнительную информацию о курсах и преподавателях
	courseMap := make(map[int64]string)   // courseID -> courseName
	teacherMap := make(map[string]string) // teacherRegCode -> teacherName

	// Заполняем courseMap
	courses, err := db.DB.Query(`SELECT id, name FROM courses`)
	if err != nil {
		return "", err
	}
	defer courses.Close()
	for courses.Next() {
		var id int64
		var name string
		if err := courses.Scan(&id, &name); err != nil {
			return "", err
		}
		courseMap[id] = name
	}

	// Заполняем teacherMap
	teachers, err := db.DB.Query(`SELECT registration_code, name FROM users WHERE role = 'teacher'`)
	if err != nil {
		return "", err
	}
	defer teachers.Close()
	for teachers.Next() {
		var regCode, name string
		if err := teachers.Scan(&regCode, &name); err != nil {
			return "", err
		}
		teacherMap[regCode] = name
	}

	// Группируем расписание по дням недели
	for _, s := range schedules {
		wd := s.ScheduleTime.Weekday()
		courseName := courseMap[s.CourseID]
		teacherName := teacherMap[s.TeacherRegCode]
		var entry string
		if mode == "teacher" {
			// Для преподавателя показываем группу
			entry = fmt.Sprintf("• %s [%s]: %s (группа: %s)", s.ScheduleTime.Format("15:04"), courseName, s.Description, s.GroupName)
		} else {
			// Для студента показываем ФИО преподавателя
			entry = fmt.Sprintf("• %s [%s]: %s (Преп.: %s)", s.ScheduleTime.Format("15:04"), courseName, s.Description, teacherName)
		}
		scheduleByDay[wd] = append(scheduleByDay[wd], entry)
	}

	// Порядок дней недели
	weekdaysOrder := []time.Weekday{
		time.Monday, time.Tuesday, time.Wednesday,
		time.Thursday, time.Friday, time.Saturday, time.Sunday,
	}

	// Формируем текст сообщения
	msgText := "Ваше расписание:\n\n"
	for _, wd := range weekdaysOrder {
		entries, ok := scheduleByDay[wd]
		if !ok || len(entries) == 0 {
			continue
		}
		msgText += fmt.Sprintf("🔹 %s:\n", weekdayName(wd))
		for _, entry := range entries {
			msgText += "  " + entry + "\n"
		}
		msgText += "\n"
	}

	return msgText, nil
}

// ShowSchedule отправляет расписание пользователю в Telegram с использованием кеша.
func ShowSchedule(chatID int64, bot *tgbotapi.BotAPI, user *models.User) error {
	limit := 5
	var schedules []models.Schedule
	var totalRecords int
	var err error

	if user.Role == "teacher" {
		// Используем кешированное получение расписания для преподавателя
		schedules, totalRecords, err = GetScheduleByTeacherCachedPaginated(user.RegistrationCode, limit, 0)
		if err != nil {
			return err
		}
	} else {
		// Используем кешированное получение расписания для студента (по группе)
		schedules, totalRecords, err = GetScheduleByGroupCachedPaginated(user.Group, limit, 0)
		if err != nil {
			return err
		}
	}

	totalPages := (totalRecords + limit - 1) / limit
	text := FormatPaginatedSchedules(schedules, 1, totalPages, user.Role, user)
	keyboard := BuildPaginationKeyboardWithNumbers(1, totalPages, "schedule")

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	return sendAndTrackMessage(bot, msg)
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
