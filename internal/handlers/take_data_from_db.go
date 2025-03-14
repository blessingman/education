package handlers

import (
	"education/internal/db"
	"education/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// sendRoleSelection отправляет inline‑кнопки для выбора роли.
func sendRoleSelection(chatID int64, bot *tgbotapi.BotAPI) {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Кнопки выбора роли
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Студент", "role_student"),
		tgbotapi.NewInlineKeyboardButtonData("Преподаватель", "role_teacher"),
	))

	// Кнопка «Отмена»
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Отмена Регистрации", "cancel_process"),
	))

	msg := tgbotapi.NewMessage(chatID, "👤 Выберите вашу роль (или отмените операцию):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	sendAndTrackMessage(bot, msg)
}

// sendFacultySelection отправляет inline‑кнопки факультетов с использованием кэша.
func sendFacultySelection(chatID int64, bot *tgbotapi.BotAPI) {
	facs := GetFaculties()
	if len(facs) == 0 {
		var err error
		facs, err = GetAllFaculties()
		if err != nil || len(facs) == 0 {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Нет факультетов для выбора.")
			sendAndTrackMessage(bot, msg)
			return
		}
		SetFaculties(facs)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, f := range facs {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(f, f),
		)
		rows = append(rows, row)
	}

	// Кнопка «Отмена»
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Отмена Регистрации", "cancel_process"),
	))

	msg := tgbotapi.NewMessage(chatID, "📚 Выберите ваш факультет (или отмените операцию):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	sendAndTrackMessage(bot, msg)
}

// sendGroupSelection отправляет inline‑кнопки групп для выбранного факультета с использованием кэша.
func sendGroupSelection(chatID int64, facultyName string, bot *tgbotapi.BotAPI) {
	groups := GetGroups(facultyName)
	if len(groups) == 0 {
		var err error
		groups, err = GetGroupsByFaculty(facultyName)
		if err != nil || len(groups) == 0 {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Нет групп для факультета "+facultyName+".")
			sendAndTrackMessage(bot, msg)
			return
		}
		SetGroups(facultyName, groups)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, g := range groups {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(g, g),
		)
		rows = append(rows, row)
	}

	// Кнопка «Отмена»
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Отмена Регистрации", "cancel_process"),
	))

	msg := tgbotapi.NewMessage(chatID, "📖 Выберите вашу группу (или отмените операцию):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	sendAndTrackMessage(bot, msg)
}

// GetCoursesByTeacherID возвращает список курсов, которые ведёт преподаватель с заданным teacherID.
// Если один и тот же курс привязан к разным группам, курс будет возвращён только один раз.
func GetCoursesByTeacherID(teacherID int64) ([]models.Course, error) {
	rows, err := db.DB.Query(`
		SELECT c.id, c.name, c.description
		FROM teacher_course_groups tcg
		JOIN courses c ON c.id = tcg.course_id
		WHERE tcg.teacher_id = ?
		GROUP BY c.id, c.name, c.description
	`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		if err := rows.Scan(&course.ID, &course.Name, &course.Description); err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}
	return courses, nil
}

// GetTeacherGroups возвращает список групп, с которыми связан преподаватель в таблице teacher_course_groups.
func GetTeacherGroups(teacherID int64) ([]models.TeacherCourseGroup, error) {
	rows, err := db.DB.Query(`
		SELECT id, teacher_id, group_name
		FROM teacher_course_groups
		WHERE teacher_id = ?
	`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.TeacherCourseGroup
	for rows.Next() {
		var tg models.TeacherCourseGroup
		if err := rows.Scan(&tg.ID, &tg.TeacherID, &tg.GroupName); err != nil {
			return nil, err
		}
		groups = append(groups, tg)
	}
	return groups, nil
}
