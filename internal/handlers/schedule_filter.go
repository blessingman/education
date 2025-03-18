package handlers

import (
	"education/internal/models"
	"fmt"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ScheduleFilter структура для хранения фильтров расписания
type ScheduleFilter struct {
	CourseID   int64  // ID курса
	CourseName string // Название курса
	LessonType string // Тип занятия (Лекция, Практика, Лабораторная, Семинар)
}

var (
	// Хранилище фильтров для каждого пользователя
	userFilters     = make(map[int64]*ScheduleFilter)
	userFilterMutex sync.RWMutex
)

// Получение фильтра пользователя
func GetUserFilter(chatID int64) *ScheduleFilter {
	userFilterMutex.RLock()
	defer userFilterMutex.RUnlock()

	filter, exists := userFilters[chatID]
	if !exists {
		return &ScheduleFilter{} // Возвращаем пустой фильтр, если нет сохраненного
	}
	return filter
}

// Установка фильтра пользователя
func SetUserFilter(chatID int64, filter *ScheduleFilter) {
	userFilterMutex.Lock()
	defer userFilterMutex.Unlock()

	userFilters[chatID] = filter
}

// Сброс фильтра пользователя
func ResetUserFilters(chatID int64) {
	userFilterMutex.Lock()
	defer userFilterMutex.Unlock()

	userFilters[chatID] = &ScheduleFilter{}
}

// Применение фильтров к выборке расписания
func ApplyFilters(schedules []models.Schedule, filter *ScheduleFilter) []models.Schedule {
	if filter == nil || (filter.CourseID == 0 && filter.CourseName == "" && filter.LessonType == "") {
		return schedules
	}

	var filtered []models.Schedule
	for _, s := range schedules {
		// Фильтр по курсу
		if filter.CourseID != 0 && s.CourseID != filter.CourseID {
			continue
		}

		// Фильтр по названию курса (если указан)
		if filter.CourseName != "" && !strings.Contains(strings.ToLower(s.Description), strings.ToLower(filter.CourseName)) {
			continue
		}

		// Фильтр по типу занятия (если указан)
		if filter.LessonType != "" && s.LessonType != filter.LessonType {
			continue
		}

		filtered = append(filtered, s)
	}

	return filtered
}

// Показать меню выбора фильтров с информацией об активных фильтрах
func ShowFilterMenu(chatID int64, bot *tgbotapi.BotAPI) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 По курсу", "filter_course_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 По типу занятия", "filter_lesson_type_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Сбросить все фильтры", "filter_reset_all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Применить фильтры", "filter_apply"),
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_schedule"),
		),
	)

	// Получаем информацию о текущих фильтрах для отображения
	filter := GetUserFilter(chatID)
	filterInfo := ""
	if filter.CourseName != "" {
		filterInfo += fmt.Sprintf("📚 Текущий фильтр по курсу: <b>%s</b>\n", filter.CourseName)
	}
	if filter.LessonType != "" {
		filterInfo += fmt.Sprintf("📝 Текущий фильтр по типу занятия: <b>%s</b>\n", filter.LessonType)
	}
	if filterInfo != "" {
		filterInfo += "\n"
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🔍 <b>Меню фильтров расписания</b>\n\n%sВыберите тип фильтра:", filterInfo))
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard

	return sendAndTrackMessage(bot, msg)
}

// ShowLessonTypeFilterMenu отображает меню выбора типа занятия для фильтрации
func ShowLessonTypeFilterMenu(chatID int64, bot *tgbotapi.BotAPI) error {
	// Доступные типы занятий
	lessonTypes := []string{"Лекция", "Практика", "Лабораторная", "Семинар"}

	var rows [][]tgbotapi.InlineKeyboardButton

	// Создаем кнопки для каждого типа занятия
	for _, lessonType := range lessonTypes {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(lessonType, "filter_lesson_type_"+lessonType),
		))
	}

	// Добавляем кнопку для сброса фильтра и возврата
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Сбросить фильтр типа", "filter_lesson_type_reset"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Назад к фильтрам", "filter_menu"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, "Выберите тип занятия для фильтрации:")
	msg.ReplyMarkup = keyboard
	return sendAndTrackMessage(bot, msg)
}
