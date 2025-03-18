package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Pagination state storage for materials
var (
	materialPageState    = make(map[int64]int)    // chatID -> current page
	materialFilterState  = make(map[int64]string) // chatID -> current filter (course name)
	materialItemsPerPage = 5                      // Number of materials per page
	materialStateMutex   sync.RWMutex
)

// GetMaterialsByTeacher возвращает материалы, загруженные преподавателем с поддержкой пагинации.
func GetMaterialsByTeacher(teacherRegCode string, limit, offset int) ([]models.Material, error) {
	query := `
		SELECT m.id, m.course_id, m.group_name, m.teacher_reg_code, m.title, m.file_url, m.description
		FROM materials m
		WHERE m.teacher_reg_code = ?
		ORDER BY m.id DESC
		LIMIT ? OFFSET ?
	`
	rows, err := db.DB.Query(query, teacherRegCode, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []models.Material
	for rows.Next() {
		var m models.Material
		if err := rows.Scan(&m.ID, &m.CourseID, &m.GroupName, &m.TeacherRegCode, &m.Title, &m.FileURL, &m.Description); err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}

// GetMaterialsByGroup возвращает материалы для указанной группы с поддержкой пагинации.
func GetMaterialsByGroup(group string, limit, offset int) ([]models.Material, error) {
	query := `
		SELECT m.id, m.course_id, m.group_name, m.teacher_reg_code, m.title, m.file_url, m.description
		FROM materials m
		WHERE m.group_name = ?
		ORDER BY m.id DESC
		LIMIT ? OFFSET ?
	`
	rows, err := db.DB.Query(query, group, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []models.Material
	for rows.Next() {
		var m models.Material
		if err := rows.Scan(&m.ID, &m.CourseID, &m.GroupName, &m.TeacherRegCode, &m.Title, &m.FileURL, &m.Description); err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}

// GetMaterialsByTeacherAndCourse возвращает материалы, загруженные преподавателем для конкретного курса.
func GetMaterialsByTeacherAndCourse(teacherRegCode string, courseID int64, limit, offset int) ([]models.Material, error) {
	query := `
		SELECT m.id, m.course_id, m.group_name, m.teacher_reg_code, m.title, m.file_url, m.description
		FROM materials m
		WHERE m.teacher_reg_code = ? AND m.course_id = ?
		ORDER BY m.id DESC
		LIMIT ? OFFSET ?
	`
	rows, err := db.DB.Query(query, teacherRegCode, courseID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []models.Material
	for rows.Next() {
		var m models.Material
		if err := rows.Scan(&m.ID, &m.CourseID, &m.GroupName, &m.TeacherRegCode, &m.Title, &m.FileURL, &m.Description); err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}

// GetMaterialsByGroupAndCourse возвращает материалы для указанной группы и курса.
func GetMaterialsByGroupAndCourse(group string, courseID int64, limit, offset int) ([]models.Material, error) {
	query := `
		SELECT m.id, m.course_id, m.group_name, m.teacher_reg_code, m.title, m.file_url, m.description
		FROM materials m
		WHERE m.group_name = ? AND m.course_id = ?
		ORDER BY m.id DESC
		LIMIT ? OFFSET ?
	`
	rows, err := db.DB.Query(query, group, courseID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []models.Material
	for rows.Next() {
		var m models.Material
		if err := rows.Scan(&m.ID, &m.CourseID, &m.GroupName, &m.TeacherRegCode, &m.Title, &m.FileURL, &m.Description); err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}

// CountMaterialsByTeacher возвращает общее количество материалов для преподавателя.
func CountMaterialsByTeacher(teacherRegCode string) (int, error) {
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM materials WHERE teacher_reg_code = ?", teacherRegCode).Scan(&count)
	return count, err
}

// CountMaterialsByGroup возвращает общее количество материалов для группы.
func CountMaterialsByGroup(group string) (int, error) {
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM materials WHERE group_name = ?", group).Scan(&count)
	return count, err
}

// CountMaterialsByTeacherAndCourse возвращает общее количество материалов для преподавателя и курса.
func CountMaterialsByTeacherAndCourse(teacherRegCode string, courseID int64) (int, error) {
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM materials WHERE teacher_reg_code = ? AND course_id = ?",
		teacherRegCode, courseID).Scan(&count)
	return count, err
}

// CountMaterialsByGroupAndCourse возвращает общее количество материалов для группы и курса.
func CountMaterialsByGroupAndCourse(group string, courseID int64) (int, error) {
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM materials WHERE group_name = ? AND course_id = ?",
		group, courseID).Scan(&count)
	return count, err
}

// FormatMaterials форматирует материалы для удобного отображения с учетом пагинации.
func FormatMaterials(materials []models.Material, mode string, currentPage, totalPages int, user *models.User) (string, error) {
	if len(materials) == 0 {
		return "📚 *Материалы не найдены*.\n\nВозможно, стоит проверить фильтры или обратиться к преподавателю.", nil
	}

	// Получаем информацию о курсах
	courseMap := make(map[int64]string) // courseID -> courseName
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

	// Получаем имена преподавателей (для студенческого режима)
	teacherMap := make(map[string]string) // reg_code -> name
	if mode == "student" {
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
	}

	// Группируем материалы по курсам для более удобного отображения
	courseGroups := make(map[int64][]models.Material)
	for _, m := range materials {
		courseGroups[m.CourseID] = append(courseGroups[m.CourseID], m)
	}

	// Заголовок сообщения
	msgText := "📚 *Учебные материалы*\n\n"

	// Для каждого курса выводим материалы
	for courseID, courseMaterials := range courseGroups {
		courseName := courseMap[courseID]
		if courseName == "" {
			courseName = fmt.Sprintf("Курс #%d", courseID)
		}

		msgText += fmt.Sprintf("📘 *%s*:\n", courseName)

		for _, m := range courseMaterials {
			msgText += fmt.Sprintf("  • *%s*\n", m.Title)

			if m.Description != "" {
				msgText += fmt.Sprintf("    📝 %s\n", m.Description)
			}

			if m.FileURL != "" {
				msgText += fmt.Sprintf("    🔗 [Скачать материал](%s)\n", m.FileURL)
			}

			if mode == "teacher" {
				msgText += fmt.Sprintf("    👥 Группа: %s\n", m.GroupName)
			} else {
				teacherName := teacherMap[m.TeacherRegCode]
				if teacherName == "" {
					teacherName = m.TeacherRegCode
				}
				msgText += fmt.Sprintf("    👨‍🏫 Преподаватель: %s\n", teacherName)
			}

			msgText += "\n"
		}
	}

	// Добавляем информацию о пагинации
	if totalPages > 1 {
		msgText += fmt.Sprintf("Страница %d из %d\n", currentPage, totalPages)
	}

	return msgText, nil
}

// BuildMaterialsPaginationKeyboard создает клавиатуру для навигации по страницам материалов.
func BuildMaterialsPaginationKeyboard(currentPage, totalPages int, filter string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Кнопки навигации по страницам
	if totalPages > 1 {
		var navButtons []tgbotapi.InlineKeyboardButton

		// Кнопка "Предыдущая страница"
		if currentPage > 1 {
			navButtons = append(navButtons,
				tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("mat_page_%d", currentPage-1)))
		}

		// Кнопка "Следующая страница"
		if currentPage < totalPages {
			navButtons = append(navButtons,
				tgbotapi.NewInlineKeyboardButtonData("Вперёд ▶️", fmt.Sprintf("mat_page_%d", currentPage+1)))
		}

		if len(navButtons) > 0 {
			rows = append(rows, navButtons)
		}
	}

	// Кнопки фильтров
	filterButtons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔍 Фильтр по курсу", "mat_filter"),
	}

	// Если установлен фильтр, добавляем кнопку сброса
	if filter != "" {
		filterButtons = append(filterButtons,
			tgbotapi.NewInlineKeyboardButtonData("❌ Сбросить фильтр", "mat_filter_reset"))
	}

	rows = append(rows, filterButtons)

	// Кнопка возврата в главное меню
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🏠 В главное меню", "menu_main"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ShowMaterials отображает материалы с пагинацией и навигацией.
func ShowMaterials(chatID int64, bot *tgbotapi.BotAPI, user *models.User) error {
	if user == nil {
		msg := tgbotapi.NewMessage(chatID, "⚠️ Необходимо войти в систему для просмотра материалов.")
		return sendAndTrackMessage(bot, msg)
	}

	// Получаем текущую страницу и фильтр
	materialStateMutex.RLock()
	currentPage := materialPageState[chatID]
	filter := materialFilterState[chatID]
	materialStateMutex.RUnlock()

	if currentPage == 0 {
		currentPage = 1
		materialStateMutex.Lock()
		materialPageState[chatID] = currentPage
		materialStateMutex.Unlock()
	}

	// Вычисляем offset для пагинации
	offset := (currentPage - 1) * materialItemsPerPage

	var materials []models.Material
	var totalCount int
	var err error
	var mode string

	// Определяем режим (преподаватель или студент)
	if user.Role == "teacher" {
		mode = "teacher"

		// Проверяем, есть ли фильтр по курсу
		if filter != "" {
			courseID, err := strconv.ParseInt(filter, 10, 64)
			if err == nil {
				materials, err = GetMaterialsByTeacherAndCourse(user.RegistrationCode, courseID, materialItemsPerPage, offset)
				if err == nil {
					totalCount, err = CountMaterialsByTeacherAndCourse(user.RegistrationCode, courseID)
				}
			}
		} else {
			materials, err = GetMaterialsByTeacher(user.RegistrationCode, materialItemsPerPage, offset)
			if err == nil {
				totalCount, err = CountMaterialsByTeacher(user.RegistrationCode)
			}
		}
	} else {
		mode = "student"

		// Проверяем, есть ли фильтр по курсу
		if filter != "" {
			courseID, err := strconv.ParseInt(filter, 10, 64)
			if err == nil {
				materials, err = GetMaterialsByGroupAndCourse(user.Group, courseID, materialItemsPerPage, offset)
				if err == nil {
					totalCount, err = CountMaterialsByGroupAndCourse(user.Group, courseID)
				}
			}
		} else {
			materials, err = GetMaterialsByGroup(user.Group, materialItemsPerPage, offset)
			if err == nil {
				totalCount, err = CountMaterialsByGroup(user.Group)
			}
		}
	}

	if err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ Ошибка при получении материалов: %s", err.Error()))
		return sendAndTrackMessage(bot, msg)
	}

	// Вычисляем общее количество страниц
	totalPages := int(math.Ceil(float64(totalCount) / float64(materialItemsPerPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	// Проверяем, не превышает ли текущая страница общего количества страниц
	if currentPage > totalPages {
		currentPage = totalPages
		materialStateMutex.Lock()
		materialPageState[chatID] = currentPage
		materialStateMutex.Unlock()

		// Пересчитываем смещение и получаем материалы заново
		offset = (currentPage - 1) * materialItemsPerPage
		if mode == "teacher" {
			if filter != "" {
				courseID, _ := strconv.ParseInt(filter, 10, 64)
				materials, _ = GetMaterialsByTeacherAndCourse(user.RegistrationCode, courseID, materialItemsPerPage, offset)
			} else {
				materials, _ = GetMaterialsByTeacher(user.RegistrationCode, materialItemsPerPage, offset)
			}
		} else {
			if filter != "" {
				courseID, _ := strconv.ParseInt(filter, 10, 64)
				materials, _ = GetMaterialsByGroupAndCourse(user.Group, courseID, materialItemsPerPage, offset)
			} else {
				materials, _ = GetMaterialsByGroup(user.Group, materialItemsPerPage, offset)
			}
		}
	}

	// Форматируем материалы для отображения
	msgText, err := FormatMaterials(materials, mode, currentPage, totalPages, user)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка при форматировании материалов.")
		return sendAndTrackMessage(bot, msg)
	}

	// Создаем клавиатуру для навигации
	keyboard := BuildMaterialsPaginationKeyboard(currentPage, totalPages, filter)

	// Отправляем сообщение с материалами и клавиатурой
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	msg.DisableWebPagePreview = false // Разрешаем предпросмотр для ссылок на материалы

	return sendAndTrackMessage(bot, msg)
}

// ShowMaterialFilters отображает список курсов для фильтрации материалов.
func ShowMaterialFilters(chatID int64, bot *tgbotapi.BotAPI, user *models.User) error {
	if user == nil {
		msg := tgbotapi.NewMessage(chatID, "⚠️ Необходимо войти в систему.")
		return sendAndTrackMessage(bot, msg)
	}

	// Получаем список доступных курсов для фильтрации
	var courses []models.Course
	var err error

	if user.Role == "teacher" {
		// Для преподавателя - его курсы
		courses, err = GetCoursesByTeacherRegCode(user.RegistrationCode)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ Ошибка при получении курсов: %s", err.Error()))
			return sendAndTrackMessage(bot, msg)
		}
	} else {
		// Для студента - курсы его группы
		courses, err = GetCoursesForGroup(user.Group)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ Ошибка при получении курсов: %s", err.Error()))
			return sendAndTrackMessage(bot, msg)
		}
	}

	if len(courses) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет доступных курсов для фильтрации.")
		return sendAndTrackMessage(bot, msg)
	}

	// Создаем клавиатуру с кнопками для каждого курса
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, course := range courses {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				course.Name, fmt.Sprintf("mat_filter_set_%d", course.ID)),
		})
	}

	// Добавляем кнопку "Отмена"
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "mat_cancel"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	// Отправляем сообщение с выбором курса
	msg := tgbotapi.NewMessage(chatID, "🔍 Выберите курс для фильтрации материалов:")
	msg.ReplyMarkup = keyboard

	return sendAndTrackMessage(bot, msg)
}

// GetCoursesForGroup возвращает список курсов, доступных для группы.
func GetCoursesForGroup(group string) ([]models.Course, error) {
	query := `
		SELECT DISTINCT c.id, c.name
		FROM courses c
		JOIN teacher_course_groups tcg ON c.id = tcg.course_id
		WHERE tcg.group_name = ?
		ORDER BY c.name
	`
	rows, err := db.DB.Query(query, group)
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

// ProcessMaterialsCallback обрабатывает коллбэки, связанные с материалами.
func ProcessMaterialsCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI, user *models.User) bool {
	data := callback.Data
	chatID := callback.Message.Chat.ID

	// Обработка навигации по страницам
	if strings.HasPrefix(data, "mat_page_") {
		pageStr := strings.TrimPrefix(data, "mat_page_")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, "⚠️ Ошибка обработки страницы"))
			return true
		}

		materialStateMutex.Lock()
		materialPageState[chatID] = page
		materialStateMutex.Unlock()

		bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("📖 Страница %d", page)))
		ShowMaterials(chatID, bot, user)
		return true
	}

	// Показать фильтры
	if data == "mat_filter" {
		bot.Request(tgbotapi.NewCallback(callback.ID, "🔍 Выбор фильтра"))
		ShowMaterialFilters(chatID, bot, user)
		return true
	}

	// Сбросить фильтр
	if data == "mat_filter_reset" {
		materialStateMutex.Lock()
		delete(materialFilterState, chatID)
		materialPageState[chatID] = 1 // Сбрасываем страницу на первую
		materialStateMutex.Unlock()

		bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Фильтр сброшен"))
		ShowMaterials(chatID, bot, user)
		return true
	}

	// Установить фильтр по курсу
	if strings.HasPrefix(data, "mat_filter_set_") {
		courseIDStr := strings.TrimPrefix(data, "mat_filter_set_")

		materialStateMutex.Lock()
		materialFilterState[chatID] = courseIDStr
		materialPageState[chatID] = 1 // При изменении фильтра возвращаемся на первую страницу
		materialStateMutex.Unlock()

		bot.Request(tgbotapi.NewCallback(callback.ID, "🔍 Фильтр установлен"))
		ShowMaterials(chatID, bot, user)
		return true
	}

	// Отмена действия в материалах
	if data == "mat_cancel" {
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Отменено"))
		ShowMaterials(chatID, bot, user)
		return true
	}

	// Переход в главное меню из материалов
	if data == "menu_main" {
		bot.Request(tgbotapi.NewCallback(callback.ID, "🏠 Главное меню"))
		sendMainMenu(chatID, bot, user)
		return true
	}

	// Не наш callback
	return false
}
