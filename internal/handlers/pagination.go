package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
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

func BuildWeekNavigationKeyboard(weekStart time.Time) tgbotapi.InlineKeyboardMarkup {
	// Вычисляем даты для предыдущей и следующей недели
	prevWeek := weekStart.AddDate(0, 0, -7)
	nextWeek := weekStart.AddDate(0, 0, 7)

	// Первая строка: кнопки для перехода к предыдущей и следующей неделе
	navRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Пред. неделя", fmt.Sprintf("week_prev_%s", prevWeek.Format("2006-01-02"))),
		tgbotapi.NewInlineKeyboardButtonData("След. неделя ▶️", fmt.Sprintf("week_next_%s", nextWeek.Format("2006-01-02"))),
	)

	// Вторая строка: кнопки с названиями дней недели
	var dayRow []tgbotapi.InlineKeyboardButton
	dayNames := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		dayRow = append(dayRow, tgbotapi.NewInlineKeyboardButtonData(dayNames[i], fmt.Sprintf("day_%s", day.Format("2006-01-02"))))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(navRow, dayRow)
	return keyboard
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

	// Заголовок с информацией о странице
	msgText := fmt.Sprintf("<b>Расписание (страница %d из %d)</b>\n\n", currentPage, totalPages)

	// Группируем записи по дате (без времени)
	type dayKey string // формат: YYYY-MM-DD
	grouped := make(map[dayKey][]models.Schedule)
	for _, s := range schedules {
		dateOnly := s.ScheduleTime.Format("2006-01-02")
		grouped[dayKey(dateOnly)] = append(grouped[dayKey(dateOnly)], s)
	}

	// Сортировка дат по возрастанию
	var sortedDates []string
	for k := range grouped {
		sortedDates = append(sortedDates, string(k))
	}
	sort.Strings(sortedDates)

	// Получаем справочники по курсам и преподавателям (используем уже имеющийся код)
	courseMap := make(map[int64]string)
	teacherMap := make(map[string]string)
	{
		rowsCourses, err := db.DB.Query("SELECT id, name FROM courses")
		if err == nil {
			defer rowsCourses.Close()
			for rowsCourses.Next() {
				var id int64
				var name string
				_ = rowsCourses.Scan(&id, &name)
				courseMap[id] = name
			}
		}
		rowsTeachers, err := db.DB.Query("SELECT registration_code, name FROM users WHERE role = 'teacher'")
		if err == nil {
			defer rowsTeachers.Close()
			for rowsTeachers.Next() {
				var regCode, name string
				_ = rowsTeachers.Scan(&regCode, &name)
				teacherMap[regCode] = name
			}
		}
	}

	// Формируем текст для каждого дня
	for _, dateStr := range sortedDates {
		// Преобразуем дату в формат time.Time для определения дня недели
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		// Заголовок дня с эмодзи календаря и днем недели
		dayHeader := fmt.Sprintf("📅 <b>%s (%s)</b>\n", t.Format("02.01.2006"), weekdayName(t.Weekday()))
		msgText += dayHeader

		// Для каждого занятия за этот день – форматирование с отступами и эмодзи времени
		for _, s := range grouped[dayKey(dateStr)] {
			timeStr := s.ScheduleTime.Format("15:04")
			courseName := courseMap[s.CourseID]
			teacherName := teacherMap[s.TeacherRegCode]

			if mode == "teacher" {
				// Для преподавателя – отображаем группу
				msgText += fmt.Sprintf("   • <i>%s</i> [%s]: %s (<b>группа:</b> %s)\n", timeStr, courseName, s.Description, s.GroupName)
			} else {
				// Для студента – отображаем имя преподавателя
				msgText += fmt.Sprintf("   • <i>%s</i> [%s]: %s (<b>Преп.:</b> %s)\n", timeStr, courseName, s.Description, teacherName)
			}
		}
		msgText += "\n" // дополнительный отступ между днями
	}

	// Можно добавить подвал с информацией или шуткой
	msgText += "<i>Надеемся, расписание вам поможет организовать учебный процесс!</i>"

	return msgText
}
