package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
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
	    WHERE teacher_reg_code = ? AND course_id = ? AND group_name = ?
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

// GetTeacherSchedulesFormatted формирует строку с расписанием для преподавателя по его регистрационному коду.
func GetTeacherSchedulesFormatted(teacherRegCode string) (string, error) {
	// Выполняем запрос с объединением таблицы schedules и courses для получения названия курса
	rows, err := db.DB.Query(`
		SELECT s.schedule_time, s.description, s.group_name, c.name
		FROM schedules s
		JOIN courses c ON s.course_id = c.id
		WHERE s.teacher_reg_code = ?
		ORDER BY s.schedule_time
	`, teacherRegCode)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	// Группируем записи по курсу
	scheduleMap := make(map[string][]string)
	for rows.Next() {
		var scheduleTimeStr, description, groupName, courseName string
		if err := rows.Scan(&scheduleTimeStr, &description, &groupName, &courseName); err != nil {
			return "", err
		}
		// Преобразуем строку времени в time.Time
		t, err := time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return "", err
		}
		// Форматируем дату/время и составляем запись
		entry := fmt.Sprintf("• %s – %s (группа: %s)", t.Format("02.01.2006 15:04"), description, groupName)
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

// GetStudentSchedulesFormatted формирует строку с расписанием для студента по его группе.
func GetStudentSchedulesFormatted(group string) (string, error) {
	// Выполняем запрос с объединением таблицы schedules и courses для получения названия курса
	rows, err := db.DB.Query(`
		SELECT s.schedule_time, s.description, s.teacher_reg_code, c.name
		FROM schedules s
		JOIN courses c ON s.course_id = c.id
		WHERE s.group_name = ?
		ORDER BY s.schedule_time
	`, group)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	// Группируем записи по курсу
	scheduleMap := make(map[string][]string)
	for rows.Next() {
		var scheduleTimeStr, description, teacherRegCode, courseName string
		if err := rows.Scan(&scheduleTimeStr, &description, &teacherRegCode, &courseName); err != nil {
			return "", err
		}
		// Преобразуем строку времени в time.Time
		t, err := time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return "", err
		}
		// Формируем запись: время – описание – преподаватель
		entry := fmt.Sprintf("• %s – %s (Преп.: %s)", t.Format("02.01.2006 15:04"), description, teacherRegCode)
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
