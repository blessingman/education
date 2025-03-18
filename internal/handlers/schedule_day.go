package handlers

import (
	"education/internal/models"
	"fmt"
	"sort"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ShowEnhancedScheduleDay shows an enhanced version of the daily schedule
func ShowEnhancedScheduleDay(chatID int64, bot *tgbotapi.BotAPI, user *models.User, day time.Time) error {
	dayStart := day.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24*time.Hour - time.Second)

	var schedules []models.Schedule
	var err error
	if user.Role == "teacher" {
		schedules, err = GetSchedulesForTeacherByDateRange(user.RegistrationCode, dayStart, dayEnd)
	} else {
		schedules, err = GetSchedulesForGroupByDateRange(user.Group, dayStart, dayEnd)
	}
	if err != nil {
		return err
	}

	// Применяем фильтры к полученному расписанию
	filter := GetUserFilter(chatID)
	filteredSchedules := ApplyFilters(schedules, filter)

	text := FormatEnhancedDaySchedule(filteredSchedules, day, user.Role)

	// Add navigation buttons for previous/next day
	prevDay := day.AddDate(0, 0, -1)
	nextDay := day.AddDate(0, 0, 1)

	// Создаем навигационные кнопки
	navRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Пред. день", fmt.Sprintf("day_%s", prevDay.Format("2006-01-02"))),
		tgbotapi.NewInlineKeyboardButtonData("Сегодня", "mode_day"),
		tgbotapi.NewInlineKeyboardButtonData("След. день ▶️", fmt.Sprintf("day_%s", nextDay.Format("2006-01-02"))),
	)

	// Получаем базовую клавиатуру переключения режимов
	modeKeyboard := BuildModeSwitchKeyboard("mode_day")

	// Добавляем кнопку фильтров
	filterRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔍 Настроить фильтры", "filter_menu"),
	)

	// Если фильтры активны, добавляем информацию о них в сообщение
	if filter.CourseName != "" || filter.LessonType != "" {
		text += "\n\n<b>📌 Активные фильтры:</b>\n"
		if filter.CourseName != "" {
			text += fmt.Sprintf("• Курс: <b>%s</b>\n", filter.CourseName)
		}
		if filter.LessonType != "" {
			text += fmt.Sprintf("• Тип занятия: <b>%s</b>\n", filter.LessonType)
		}
	}

	// Объединяем все ряды кнопок
	var allRows [][]tgbotapi.InlineKeyboardButton
	allRows = append(allRows, navRow)
	allRows = append(allRows, modeKeyboard.InlineKeyboard...)
	allRows = append(allRows, filterRow)

	enhancedKeyboard := tgbotapi.NewInlineKeyboardMarkup(allRows...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = enhancedKeyboard
	return sendAndTrackMessage(bot, msg)
}

// FormatEnhancedDaySchedule creates a beautifully formatted day schedule
func FormatEnhancedDaySchedule(schedules []models.Schedule, day time.Time, role string) string {
	if len(schedules) == 0 {
		return fmt.Sprintf("📆 <b>%s</b>\n\n🔍 <i>Нет занятий на этот день</i>",
			day.Format("02.01.2006")+" ("+weekdayName(day.Weekday())+")")
	}

	// Сортируем занятия по времени
	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].ScheduleTime.Before(schedules[j].ScheduleTime)
	})

	var sb strings.Builder
	// Заголовок с датой и днем недели
	sb.WriteString(fmt.Sprintf("📆 <b>%s</b>\n\n",
		day.Format("02.01.2006")+" ("+weekdayName(day.Weekday())+")"))

	// Разделитель заголовка
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Счетчик для занятий
	lessonCount := 0

	for _, s := range schedules {
		lessonCount++
		timeStr := s.ScheduleTime.Format("15:04")
		endTimeStr := s.ScheduleTime.Add(time.Duration(s.Duration) * time.Minute).Format("15:04")

		// Блок информации о занятии с порядковым номером
		sb.WriteString(fmt.Sprintf("📌 <b>Занятие %d</b>\n", lessonCount))
		sb.WriteString(fmt.Sprintf("⏰ <b>%s - %s</b> (%d мин.)\n", timeStr, endTimeStr, s.Duration))
		sb.WriteString(fmt.Sprintf("📚 <b>%s</b>\n", s.Description))

		if role == "teacher" {
			sb.WriteString(fmt.Sprintf("👥 Группа: %s\n", s.GroupName))
		} else {
			sb.WriteString(fmt.Sprintf("👨‍🏫 Преподаватель: %s\n", s.TeacherRegCode))
		}

		sb.WriteString(fmt.Sprintf("🚪 Аудитория: %s\n", s.Auditory))
		sb.WriteString(fmt.Sprintf("📝 Тип занятия: %s\n", s.LessonType))

		// Добавляем разделитель между занятиями
		sb.WriteString("\n")
	}

	// Итоговая информация
	sb.WriteString(fmt.Sprintf("🔢 <b>Всего занятий: %d</b>\n", lessonCount))

	// Находим общую продолжительность занятий
	var totalDuration int
	for _, s := range schedules {
		totalDuration += s.Duration
	}
	sb.WriteString(fmt.Sprintf("⌛ <b>Общая продолжительность: %d мин (%d ч %d мин)</b>\n\n",
		totalDuration, totalDuration/60, totalDuration%60))

	sb.WriteString("✨ <i>Пусть день пройдет продуктивно!</i>")

	return sb.String()
}
