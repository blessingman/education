package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CountSchedulesByGroup возвращает общее количество записей расписания для указанной группы.
func CountSchedulesByGroup(group string) (int, error) {
	query := `SELECT COUNT(*) FROM schedules WHERE group_name = ?`
	var count int
	err := db.DB.QueryRow(query, group).Scan(&count)
	return count, err
}

// CountSchedulesByTeacher возвращает общее количество записей расписания для преподавателя.
func CountSchedulesByTeacher(teacherRegCode string) (int, error) {
	query := `SELECT COUNT(*) FROM schedules WHERE teacher_reg_code = ?`
	var count int
	err := db.DB.QueryRow(query, teacherRegCode).Scan(&count)
	return count, err
}

// GetScheduleByGroupPaginated выполняет выборку расписания для группы с использованием LIMIT и OFFSET.
func GetScheduleByGroupPaginated(group string, limit, offset int) ([]models.Schedule, error) {
	query := `
		SELECT id, course_id, group_name, teacher_reg_code, schedule_time, description
		FROM schedules
		WHERE group_name = ?
		ORDER BY schedule_time
		LIMIT ? OFFSET ?
	`
	rows, err := db.DB.Query(query, group, limit, offset)
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

// GetScheduleByTeacherPaginated выполняет выборку расписания для преподавателя с использованием LIMIT и OFFSET.
func GetScheduleByTeacherPaginated(teacherRegCode string, limit, offset int) ([]models.Schedule, error) {
	query := `
		SELECT id, course_id, group_name, teacher_reg_code, schedule_time, description
		FROM schedules
		WHERE teacher_reg_code = ?
		ORDER BY schedule_time
		LIMIT ? OFFSET ?
	`
	rows, err := db.DB.Query(query, teacherRegCode, limit, offset)
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

// FormatPaginatedSchedules формирует текст сообщения с расписанием и информацией о текущей странице.
func FormatPaginatedSchedules(schedules []models.Schedule, currentPage, totalPages int, mode string, user *models.User) string {
	if len(schedules) == 0 {
		return "Расписание не найдено."
	}

	msgText := fmt.Sprintf("Расписание (страница %d из %d):\n\n", currentPage, totalPages)
	for _, s := range schedules {
		tStr := s.ScheduleTime.Format("02.01.2006 15:04")
		if mode == "teacher" {
			// Для преподавателя выводим время, описание и группу
			msgText += fmt.Sprintf("• %s: %s (группа: %s)\n", tStr, s.Description, s.GroupName)
		} else {
			// Для студента выводим время, описание и регистрационный код преподавателя
			msgText += fmt.Sprintf("• %s: %s (Преп.: %s)\n", tStr, s.Description, s.TeacherRegCode)
		}
	}
	return msgText
}

// BuildPaginationKeyboard создаёт inline‑клавиатуру для навигации по страницам.
// callbackPrefix используется для формирования callback data (например, "schedule" для расписания).
func BuildPaginationKeyboard(currentPage, totalPages int, callbackPrefix string) tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton

	if currentPage > 1 {
		// Кнопка перехода на предыдущую страницу
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("⬅️", fmt.Sprintf("%s_page_%d", callbackPrefix, currentPage-1)))
	}

	// Текущая страница (можно сделать недоступной для нажатия)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Страница %d/%d", currentPage, totalPages), "ignore"))

	if currentPage < totalPages {
		// Кнопка перехода на следующую страницу
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("➡️", fmt.Sprintf("%s_page_%d", callbackPrefix, currentPage+1)))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons)
	return keyboard
}
