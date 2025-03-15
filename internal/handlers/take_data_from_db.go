package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"strings"
	"time"

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
		SELECT c.id, c.name
		FROM teacher_course_groups tcg
		JOIN courses c ON c.id = tcg.course_id
		WHERE tcg.teacher_id = ?
		GROUP BY c.id, c.name
	`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		if err := rows.Scan(&course.ID, &course.Name); err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}
	return courses, nil
}

// GetTeacherGroups возвращает список групп, с которыми связан преподаватель в таблице teacher_course_groups.
func GetTeacherGroups(teacherRegCode string) ([]models.TeacherCourseGroup, error) {
	rows, err := db.DB.Query(`
		SELECT id, teacher_reg_code, course_id, group_name
		FROM teacher_course_groups
		WHERE teacher_reg_code = ?
	`, teacherRegCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.TeacherCourseGroup
	for rows.Next() {
		var tg models.TeacherCourseGroup
		if err := rows.Scan(&tg.ID, &tg.TeacherRegCode, &tg.CourseID, &tg.GroupName); err != nil {
			return nil, err
		}
		groups = append(groups, tg)
	}
	return groups, nil
}

// Получаем группы преподавателя по его регистрационному коду
func GetTeacherGroupsByRegCode(teacherRegCode string) ([]models.TeacherCourseGroup, error) {
	rows, err := db.DB.Query(`
        SELECT id, teacher_reg_code, course_id, group_name
        FROM teacher_course_groups
        WHERE teacher_reg_code = ?
    `, teacherRegCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.TeacherCourseGroup
	for rows.Next() {
		var tg models.TeacherCourseGroup
		if err := rows.Scan(&tg.ID, &tg.TeacherRegCode, &tg.CourseID, &tg.GroupName); err != nil {
			return nil, err
		}
		groups = append(groups, tg)
	}
	return groups, nil
}

// Получаем курсы по регистрационному коду преподавателя
func GetCoursesByTeacherRegCode(teacherRegCode string) ([]models.Course, error) {
	rows, err := db.DB.Query(`
        SELECT c.id, c.name
        FROM teacher_course_groups tcg
        JOIN courses c ON c.id = tcg.course_id
        WHERE tcg.teacher_reg_code = ?
        GROUP BY c.id, c.name
    `, teacherRegCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		if err := rows.Scan(&course.ID, &course.Name); err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}
	return courses, nil
}

// AddSchedule добавляет новую запись в таблицу schedules.
func AddSchedule(schedule models.Schedule) error {
	// Приводим время к строке в формате RFC3339 (или используйте нужный вам формат)
	scheduleTimeStr := schedule.ScheduleTime.Format(time.RFC3339)

	query := `
	    INSERT INTO schedules (course_id, group_name, teacher_reg_code, schedule_time, description)
	    VALUES (?, ?, ?, ?, ?)
	`
	_, err := db.DB.Exec(query, schedule.CourseID, schedule.GroupName, schedule.TeacherRegCode, scheduleTimeStr, schedule.Description)
	return err
}

// AddMaterial добавляет новую запись в таблицу materials.
func AddMaterial(material models.Material) error {
	query := `
	    INSERT INTO materials (course_id, group_name, teacher_reg_code, title, file_url, description)
	    VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.DB.Exec(query, material.CourseID, material.GroupName, material.TeacherRegCode, material.Title, material.FileURL, material.Description)
	return err
}

// AddScheduleForTeacher добавляет расписание для текущего предмета, если преподаватель
// закреплён за данным курсом и группой. При успешном выполнении функция вставляет запись в таблицу schedules.
func AddScheduleForTeacher(teacherRegCode string, courseID int64, groupName string, scheduleTime time.Time, description string) error {
	// Проверка: существует ли у преподавателя назначение для данного курса и группы.
	queryCheck := `
	    SELECT COUNT(*) FROM teacher_course_groups
	    WHERE teacher_reg_code = ? AND course_id = ? AND group_name = ?АStateTeacherWaitingForPassword
	`
	var count int
	err := db.DB.QueryRow(queryCheck, teacherRegCode, courseID, groupName).Scan(&count)
	if err != nil {
		return fmt.Errorf("ошибка проверки назначения: %v", err)
	}
	if count == 0 {
		return fmt.Errorf("преподаватель с регистрационным кодом %s не закреплён за курсом %d для группы %s", teacherRegCode, courseID, groupName)
	}

	// Вставляем запись в таблицу schedules.
	scheduleTimeStr := scheduleTime.Format(time.RFC3339) // можно изменить формат при необходимости
	queryInsert := `
	    INSERT INTO schedules (course_id, group_name, teacher_reg_code, schedule_time, description)
	    VALUES (?, ?, ?, ?, ?)
	`
	_, err = db.DB.Exec(queryInsert, courseID, groupName, teacherRegCode, scheduleTimeStr, description)
	if err != nil {
		return fmt.Errorf("ошибка добавления расписания: %v", err)
	}
	return nil
}

// scheduleEntryFormatter – тип функции, которая принимает время занятия, описание, дополнительное поле и название курса, и возвращает отформатированную строку.
type scheduleEntryFormatter func(t time.Time, description, extra, courseName string) string

// getSchedulesFormatted выполняет запрос к базе с указанным условием (filterClause) и аргументами,
// группирует записи по курсу, а затем форматирует вывод с помощью функции formatter.
func getSchedulesFormatted(filterClause string, args []interface{}, formatter scheduleEntryFormatter) (string, error) {
	// Запрос с объединением расписания и курсов
	query := fmt.Sprintf(`
		SELECT s.schedule_time, s.description, s.group_name, s.teacher_reg_code, c.name
		FROM schedules s
		JOIN courses c ON s.course_id = c.id
		%s
		ORDER BY s.schedule_time
	`, filterClause)

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	// Группируем записи по курсу
	scheduleMap := make(map[string][]string)
	for rows.Next() {
		var scheduleTimeStr, description, groupName, teacherRegCode, courseName string
		if err := rows.Scan(&scheduleTimeStr, &description, &groupName, &teacherRegCode, &courseName); err != nil {
			return "", err
		}
		t, err := time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return "", err
		}
		// Дополнительное поле зависит от роли:
		// Для преподавателя – группа, для студента – регистрационный код преподавателя.
		var extra string
		// Если в условии фильтрации присутствует "teacher_reg_code", считаем, что это расписание для преподавателя.
		if filterClauseContains(filterClause, "teacher_reg_code") {
			extra = fmt.Sprintf("группа: %s", groupName)
		} else {
			extra = fmt.Sprintf("Преп.: %s", teacherRegCode)
		}
		entry := formatter(t, description, extra, courseName)
		scheduleMap[courseName] = append(scheduleMap[courseName], entry)
	}

	// Формируем итоговое сообщение
	if len(scheduleMap) == 0 {
		return "Расписание не найдено.", nil
	}

	msgText := "Ваше расписание:\n\n"
	for course, entries := range scheduleMap {
		msgText += fmt.Sprintf("📘 %s:\n", course)
		for _, entry := range entries {
			msgText += "  " + entry + "\n"
		}
		msgText += "\n"
	}

	return msgText, nil
}

// filterClauseContains – простая функция для проверки наличия подстроки в условии запроса.

func filterClauseContains(filterClause, substr string) bool {
	return strings.Contains(filterClause, substr)
}

func GetTeacherSchedulesFormatted(teacherRegCode string) (string, error) {
	// Условие фильтрации: выбираем записи по teacher_reg_code
	filterClause := "WHERE s.teacher_reg_code = ?"
	args := []interface{}{teacherRegCode}
	formatter := func(t time.Time, description, extra, courseName string) string {
		return fmt.Sprintf("• %s – %s (%s)", t.Format("02.01.2006 15:04"), description, extra)
	}
	return getSchedulesFormatted(filterClause, args, formatter)
}

func GetStudentSchedulesFormatted(group string) (string, error) {
	// Условие фильтрации: выбираем записи по группе
	filterClause := "WHERE s.group_name = ?"
	args := []interface{}{group}
	formatter := func(t time.Time, description, extra, courseName string) string {
		return fmt.Sprintf("• %s – %s (%s)", t.Format("02.01.2006 15:04"), description, extra)
	}
	return getSchedulesFormatted(filterClause, args, formatter)
}

// GetSchedulesFormattedByWeekGeneric возвращает расписание, сгруппированное по дням недели,
// в зависимости от режима: "teacher" или "student". Для преподавателей фильтр – регистрационный код,
// для студентов – название группы.
func GetSchedulesFormattedByWeekGeneric(mode, filterValue string) (string, error) {
	var condition string
	switch mode {
	case "teacher":
		condition = "s.teacher_reg_code = ?"
	case "student":
		condition = "s.group_name = ?"
	default:
		return "", fmt.Errorf("неизвестный режим: %s", mode)
	}

	// Объединяем таблицы для получения названия курса и ФИО преподавателя
	query := fmt.Sprintf(`
		SELECT s.schedule_time, s.description, s.group_name, c.name, u.name
		FROM schedules s
		JOIN courses c ON s.course_id = c.id
		JOIN users u ON u.registration_code = s.teacher_reg_code
		WHERE %s
		ORDER BY s.schedule_time
	`, condition)

	rows, err := db.DB.Query(query, filterValue)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	// Группируем записи по дню недели
	scheduleByDay := make(map[time.Weekday][]string)
	for rows.Next() {
		var scheduleTimeStr, description, groupName, courseName, teacherName string
		if err := rows.Scan(&scheduleTimeStr, &description, &groupName, &courseName, &teacherName); err != nil {
			return "", err
		}
		t, err := time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return "", err
		}
		wd := t.Weekday()
		var entry string
		if mode == "teacher" {
			// Для преподавателя показываем группу
			entry = fmt.Sprintf("• %s [%s]: %s (группа: %s)", t.Format("15:04"), courseName, description, groupName)
		} else {
			// Для студента показываем ФИО преподавателя
			entry = fmt.Sprintf("• %s [%s]: %s (Преп.: %s)", t.Format("15:04"), courseName, description, teacherName)
		}
		scheduleByDay[wd] = append(scheduleByDay[wd], entry)
	}

	if len(scheduleByDay) == 0 {
		return "Расписание не найдено.", nil
	}

	// Порядок дней недели
	weekdaysOrder := []time.Weekday{
		time.Monday, time.Tuesday, time.Wednesday,
		time.Thursday, time.Friday, time.Saturday, time.Sunday,
	}

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
