package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"log"
	"sort"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getCourseAndTeacherMaps() (map[int64]string, map[string]string, error) {
	courseMap := make(map[int64]string)   // course_id -> course_name
	teacherMap := make(map[string]string) // registration_code -> teacher_name

	// Пример заполнения courseMap из таблицы courses
	rowsCourses, err := db.DB.Query("SELECT id, name FROM courses")
	if err != nil {
		return nil, nil, err
	}
	defer rowsCourses.Close()

	for rowsCourses.Next() {
		var id int64
		var name string
		if err := rowsCourses.Scan(&id, &name); err != nil {
			return nil, nil, err
		}
		courseMap[id] = name
	}

	// Пример заполнения teacherMap из таблицы users, где role='teacher'
	rowsTeachers, err := db.DB.Query("SELECT registration_code, name FROM users WHERE role = 'teacher'")
	if err != nil {
		return nil, nil, err
	}
	defer rowsTeachers.Close()

	for rowsTeachers.Next() {
		var regCode, name string
		if err := rowsTeachers.Scan(&regCode, &name); err != nil {
			return nil, nil, err
		}
		teacherMap[regCode] = name
	}

	return courseMap, teacherMap, nil
}

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
	var row []tgbotapi.InlineKeyboardButton

	// Кнопка "В начало"
	if currentPage > 1 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("<<", fmt.Sprintf("%s_page_%d", callbackPrefix, 1)))
	}
	// Кнопка "Назад"
	if currentPage > 1 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("⬅️", fmt.Sprintf("%s_page_%d", callbackPrefix, currentPage-1)))
	}

	// Текущая страница
	row = append(row, tgbotapi.NewInlineKeyboardButtonData(
		fmt.Sprintf("Стр. %d/%d", currentPage, totalPages),
		"ignore"))

	// Кнопка "Вперёд"
	if currentPage < totalPages {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("➡️", fmt.Sprintf("%s_page_%d", callbackPrefix, currentPage+1)))
	}
	// Кнопка "В конец"
	if currentPage < totalPages {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(">>", fmt.Sprintf("%s_page_%d", callbackPrefix, totalPages)))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
	return keyboard
}

func BuildPaginationKeyboardWithNumbers(currentPage, totalPages int, callbackPrefix string) tgbotapi.InlineKeyboardMarkup {
	const maxButtons = 7
	var rows [][]tgbotapi.InlineKeyboardButton

	// 1) Верхняя строка: "В начало" / "Назад"
	var topRow []tgbotapi.InlineKeyboardButton
	if currentPage > 1 {
		topRow = append(topRow, tgbotapi.NewInlineKeyboardButtonData("<<", fmt.Sprintf("%s_page_%d", callbackPrefix, 1)))
		topRow = append(topRow, tgbotapi.NewInlineKeyboardButtonData("⬅️", fmt.Sprintf("%s_page_%d", callbackPrefix, currentPage-1)))
	}
	if len(topRow) > 0 {
		rows = append(rows, topRow)
	}

	// 2) Вторая строка: кнопки с номерами страниц
	var pagesRow []tgbotapi.InlineKeyboardButton
	start := currentPage - 3
	if start < 1 {
		start = 1
	}
	end := start + maxButtons - 1
	if end > totalPages {
		end = totalPages
	}

	for p := start; p <= end; p++ {
		if p == currentPage {
			pagesRow = append(pagesRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("[%d]", p), "ignore"))
		} else {
			pagesRow = append(pagesRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d", p), fmt.Sprintf("%s_page_%d", callbackPrefix, p)))
		}
	}
	if len(pagesRow) > 0 {
		rows = append(rows, pagesRow)
	}

	// 3) Третья строка: "Вперёд" / "В конец"
	var bottomRow []tgbotapi.InlineKeyboardButton
	if currentPage < totalPages {
		bottomRow = append(bottomRow, tgbotapi.NewInlineKeyboardButtonData("➡️", fmt.Sprintf("%s_page_%d", callbackPrefix, currentPage+1)))
		bottomRow = append(bottomRow, tgbotapi.NewInlineKeyboardButtonData(">>", fmt.Sprintf("%s_page_%d", callbackPrefix, totalPages)))
	}
	if len(bottomRow) > 0 {
		rows = append(rows, bottomRow)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func GetScheduleByGroupCachedPaginated(group string, limit, offset int) ([]models.Schedule, int, error) {
	// Ключ для кеша – например, группа
	key := group

	// Попытка получить данные из кеша
	cacheEntry, found := GetCachedSchedule(key)
	if found {
		totalRecords := len(cacheEntry.Schedules)
		// Извлекаем нужную порцию (пагинация)
		end := offset + limit
		if end > totalRecords {
			end = totalRecords
		}
		if offset > totalRecords {
			return []models.Schedule{}, totalRecords, nil
		}
		return cacheEntry.Schedules[offset:end], totalRecords, nil
	}

	// Если данных в кеше нет, выполняем запрос к базе
	schedules, err := GetScheduleByGroup(group)
	if err != nil {
		return nil, 0, err
	}
	// Сохраняем данные в кеш
	SetCachedSchedule(key, schedules)
	totalRecords := len(schedules)
	// Применяем пагинацию
	end := offset + limit
	if end > totalRecords {
		end = totalRecords
	}
	if offset > totalRecords {
		return []models.Schedule{}, totalRecords, nil
	}
	return schedules[offset:end], totalRecords, nil
}

// GetScheduleByTeacherCachedPaginated возвращает расписание для преподавателя с кешированием.
// teacherRegCode – регистрационный код преподавателя, limit – количество записей на страницу, offset – смещение.
func GetScheduleByTeacherCachedPaginated(teacherRegCode string, limit, offset int) ([]models.Schedule, int, error) {
	// Используем регистрационный код в качестве ключа для кеша.
	key := teacherRegCode

	// Пытаемся получить данные из кеша.
	cacheEntry, found := GetCachedSchedule(key)
	if found {
		totalRecords := len(cacheEntry.Schedules)
		// Применяем пагинацию.
		end := offset + limit
		if end > totalRecords {
			end = totalRecords
		}
		if offset > totalRecords {
			return []models.Schedule{}, totalRecords, nil
		}
		return cacheEntry.Schedules[offset:end], totalRecords, nil
	}

	// Если данных в кеше нет, выполняем запрос к базе.
	schedules, err := GetScheduleByTeacher(teacherRegCode)
	if err != nil {
		return nil, 0, err
	}

	// Сохраняем полученные данные в кеш.
	SetCachedSchedule(key, schedules)
	totalRecords := len(schedules)
	// Применяем пагинацию.
	end := offset + limit
	if end > totalRecords {
		end = totalRecords
	}
	if offset > totalRecords {
		return []models.Schedule{}, totalRecords, nil
	}
	return schedules[offset:end], totalRecords, nil
}

func FormatSchedulesGroupedByDay(schedules []models.Schedule, currentPage, totalPages int, mode string, user *models.User) string {
	if len(schedules) == 0 {
		return "Расписание не найдено."
	}

	// Включаем, например, Markdown
	// msgText := fmt.Sprintf("*Расписание (страница %d из %d)*\n\n", currentPage, totalPages)
	// Если используем HTML:
	msgText := fmt.Sprintf("<b>Расписание (страница %d из %d)</b>\n\n", currentPage, totalPages)

	// Группируем по дате (без учёта времени)
	type dayKey string // YYYY-MM-DD
	grouped := make(map[dayKey][]models.Schedule)

	for _, s := range schedules {
		dateOnly := s.ScheduleTime.Format("2006-01-02") // Преобразуем в строку "год-месяц-день"
		grouped[dayKey(dateOnly)] = append(grouped[dayKey(dateOnly)], s)
	}

	// Сортируем ключи (даты) в возрастающем порядке
	var sortedKeys []string
	for k := range grouped {
		sortedKeys = append(sortedKeys, string(k))
	}
	sort.Strings(sortedKeys) // сортировка дат по возрастанию

	// Получим карты/справочники для курсов и имён преподавателей (если нужно)
	courseMap, teacherMap, err := getCourseAndTeacherMaps()
	if err != nil { // обработка ошибки, например:
		log.Println("Ошибка при получении курсов и преподавателей:", err)
		return ""

	}
	for _, dateStr := range sortedKeys {
		// Преобразуем строку в time.Time, чтобы вывести день недели
		t, _ := time.Parse("2006-01-02", dateStr)
		dayWeek := weekdayName(t.Weekday()) // функция из вашего кода

		// Заголовок дня. Пример с HTML (можно и Markdown):
		msgText += fmt.Sprintf("<b>%s (%s)</b>\n", t.Format("02.01.2006"), dayWeek)

		// Перебираем записи за этот день
		for _, s := range grouped[dayKey(dateStr)] {
			timePart := s.ScheduleTime.Format("15:04")
			courseName := courseMap[s.CourseID]
			teacherName := teacherMap[s.TeacherRegCode]

			if mode == "teacher" {
				// Для преподавателя — группа и курс
				msgText += fmt.Sprintf("• <i>%s</i> [%s]: %s (группа: %s)\n",
					timePart, courseName, s.Description, s.GroupName)
			} else {
				// Для студента — преподаватель и курс
				msgText += fmt.Sprintf("• <i>%s</i> [%s]: %s (Преп.: %s)\n",
					timePart, courseName, s.Description, teacherName)
			}
		}
		msgText += "\n" // отступ между днями
	}

	return msgText
}
