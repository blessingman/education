package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"sort"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

/*
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
*/

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

func BuildWeekNavigationKeyboard(weekStart time.Time) tgbotapi.InlineKeyboardMarkup {
	// Вычисляем даты для предыдущей и следующей недели
	prevWeek := weekStart.AddDate(0, 0, -7)
	nextWeek := weekStart.AddDate(0, 0, 7)

	// Кнопки навигации с компактными подписями
	navRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ -1 нед.", fmt.Sprintf("week_prev_%s", prevWeek.Format("2006-01-02"))),
		tgbotapi.NewInlineKeyboardButtonData("+1 нед. ▶️", fmt.Sprintf("week_next_%s", nextWeek.Format("2006-01-02"))),
	)

	// Кнопки с названиями дней недели (с датой)
	var dayRow []tgbotapi.InlineKeyboardButton
	dayNames := []string{"П", "В", "С", "Ч", "П", "С", "В"}
	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		dayRow = append(dayRow, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", dayNames[i], day.Format("02.01")),
			fmt.Sprintf("day_%s", day.Format("2006-01-02")),
		))
	}

	// Добавляем кнопку "Сегодня" (возврат к текущей неделе)
	todayRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📅 Сегодня", "week_today"),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(navRow, dayRow, todayRow)
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

func FormatSchedulesGroupedByDay(
	schedules []models.Schedule,
	currentPage, totalPages int,
	mode string,
	user *models.User,
) string {
	if len(schedules) == 0 {
		return "Нет занятий на выбранный день."
	}

	// Заголовок
	msgText := "📅 <b>Расписание на выбранный день</b>\n\n"

	// Группируем занятия по дате
	type dayKey string
	grouped := make(map[dayKey][]models.Schedule)
	for _, s := range schedules {
		dateOnly := s.ScheduleTime.Format("2006-01-02")
		grouped[dayKey(dateOnly)] = append(grouped[dayKey(dateOnly)], s)
	}

	// Сортируем даты
	var sortedDates []string
	for k := range grouped {
		sortedDates = append(sortedDates, string(k))
	}
	sort.Strings(sortedDates)

	// Если нужны красивые названия курсов/преподавателей, здесь можно загрузить справочники.
	// Сейчас оставляем базовую логику.

	// Формируем текст
	for _, dateStr := range sortedDates {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		// Заголовок дня + разделитель
		msgText += fmt.Sprintf("🗓 <b>%s (%s)</b>\n", t.Format("02.01.2006"), weekdayName(t.Weekday()))
		msgText += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

		for _, s := range grouped[dayKey(dateStr)] {
			timeStr := s.ScheduleTime.Format("15:04")

			// Для преподавателя (mode == "teacher")
			if mode == "teacher" {
				msgText += fmt.Sprintf(
					"  • <b>%s</b> — %s\n    👥 Гр.: %s, 🚪 Ауд.: %s, 📋 %s, ⏱ %d мин.\n",
					timeStr, s.Description, s.GroupName, s.Auditory, s.LessonType, s.Duration,
				)
			} else {
				// Для студента
				msgText += fmt.Sprintf(
					"  • <b>%s</b> — %s\n    👨‍🏫 Преп.: %s, 🚪 Ауд.: %s, 📋 %s, ⏱ %d мин.\n",
					timeStr, s.Description, s.TeacherRegCode, s.Auditory, s.LessonType, s.Duration,
				)
			}
		}
		msgText += "\n"
	}

	// Завершающая строка
	msgText += "<i>Пусть день пройдёт продуктивно!</i>"
	return msgText
}
