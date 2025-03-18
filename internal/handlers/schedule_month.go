package handlers

import (
	"education/internal/models"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ShowScheduleMonth displays a monthly calendar view of the schedule
// monthStart is the first day of the month to show
func ShowScheduleMonth(chatID int64, bot *tgbotapi.BotAPI, user *models.User, monthStart time.Time) error {
	// Ensure we're dealing with the first day of the month
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, monthStart.Location())

	// Calculate month end (first day of next month minus 1 second)
	nextMonth := monthStart.AddDate(0, 1, 0)
	monthEnd := nextMonth.Add(-time.Second)

	// Fetch all schedules for the month
	var schedules []models.Schedule
	var err error
	if user.Role == "teacher" {
		schedules, err = GetSchedulesForTeacherByDateRange(user.RegistrationCode, monthStart, monthEnd)
	} else {
		schedules, err = GetSchedulesForGroupByDateRange(user.Group, monthStart, monthEnd)
	}
	if err != nil {
		return err
	}

	// Generate monthly calendar view
	text := BuildMonthCalendar(schedules, monthStart, user.Role)

	// Create navigation keyboard
	keyboard := BuildMonthNavigationKeyboard(monthStart, schedules)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	return sendAndTrackMessage(bot, msg)
}

// BuildMonthCalendar generates a text representation of a monthly calendar
func BuildMonthCalendar(schedules []models.Schedule, monthStart time.Time, role string) string {
	var sb strings.Builder

	// Month header
	sb.WriteString(fmt.Sprintf("📅 <b>%s %d</b>\n\n", monthStart.Month().String(), monthStart.Year()))

	// Group schedules by day of month
	eventsByDay := make(map[int][]models.Schedule)
	for _, s := range schedules {
		day := s.ScheduleTime.Day()
		eventsByDay[day] = append(eventsByDay[day], s)
	}

	// Calculate the weekday of the first day of the month (0 = Sunday, 1 = Monday, etc.)
	firstDay := monthStart.Weekday()

	// Adjust firstDay to make Monday the first day of the week (European calendar)
	if firstDay == time.Sunday {
		firstDay = 6 // Sunday becomes the 7th day (index 6)
	} else {
		firstDay-- // Monday becomes 0, Tuesday 1, etc.
	}

	// Calendar header with weekday abbreviations (Monday first)
	sb.WriteString("Пн  Вт  Ср  Чт  Пт  Сб  Вс\n")

	// Calculate days in month
	daysInMonth := 32 - time.Date(monthStart.Year(), monthStart.Month(), 32, 0, 0, 0, 0, monthStart.Location()).Day()

	// Print calendar grid
	// Add leading spaces for the first week
	for i := 0; i < int(firstDay); i++ {
		sb.WriteString("    ")
	}

	// Print days with events highlighted
	for day := 1; day <= daysInMonth; day++ {
		if _, hasEvents := eventsByDay[day]; hasEvents {
			// Highlight days with events
			if day < 10 {
				sb.WriteString(fmt.Sprintf("<b>[%d]</b> ", day))
			} else {
				sb.WriteString(fmt.Sprintf("<b>%d</b> ", day))
			}
		} else {
			// Regular days
			if day < 10 {
				sb.WriteString(fmt.Sprintf(" %d  ", day))
			} else {
				sb.WriteString(fmt.Sprintf("%d  ", day))
			}
		}

		// Start a new line after Saturday (or after the last day)
		currentWeekday := (int(firstDay) + day - 1) % 7
		if currentWeekday == 6 || day == daysInMonth {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")

	// List events for days with schedules
	if len(schedules) > 0 {
		sb.WriteString("\n<b>Расписание на дни с занятиями:</b>\n\n")

		// Sort days
		var days []int
		for day := range eventsByDay {
			days = append(days, day)
		}
		// Sort days numerically
		for i := 0; i < len(days); i++ {
			for j := i + 1; j < len(days); j++ {
				if days[i] > days[j] {
					days[i], days[j] = days[j], days[i]
				}
			}
		}

		// For each day with events
		for _, day := range days {
			dayEvents := eventsByDay[day]

			// Create a date object for this day
			currentDay := time.Date(monthStart.Year(), monthStart.Month(), day, 0, 0, 0, 0, monthStart.Location())
			sb.WriteString(fmt.Sprintf("🗓 <b>%02d.%02d (%s)</b>\n",
				day, int(monthStart.Month()), weekdayShortName(currentDay.Weekday())))

			// Group by time slot
			byTimeSlot := make(map[string][]models.Schedule)
			for _, event := range dayEvents {
				timeKey := event.ScheduleTime.Format("15:04")
				byTimeSlot[timeKey] = append(byTimeSlot[timeKey], event)
			}

			// Sort time slots
			var timeSlots []string
			for slot := range byTimeSlot {
				timeSlots = append(timeSlots, slot)
			}
			// Sort time slots
			for i := 0; i < len(timeSlots); i++ {
				for j := i + 1; j < len(timeSlots); j++ {
					if timeSlots[i] > timeSlots[j] {
						timeSlots[i], timeSlots[j] = timeSlots[j], timeSlots[i]
					}
				}
			}

			// Output events by time slot
			for _, timeSlot := range timeSlots {
				events := byTimeSlot[timeSlot]
				for _, event := range events {
					if role == "teacher" {
						sb.WriteString(fmt.Sprintf("  ⏰ %s: %s (Гр: %s, Ауд: %s)\n",
							timeSlot, event.Description, event.GroupName, event.Auditory))
					} else {
						sb.WriteString(fmt.Sprintf("  ⏰ %s: %s (Преп: %s, Ауд: %s)\n",
							timeSlot, event.Description, event.TeacherRegCode, event.Auditory))
					}
				}
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("\n<i>Нет занятий на этот месяц</i>\n")
	}

	return sb.String()
}

// BuildMonthNavigationKeyboard creates a keyboard for month navigation
func BuildMonthNavigationKeyboard(monthStart time.Time, schedules []models.Schedule) tgbotapi.InlineKeyboardMarkup {
	prevMonth := monthStart.AddDate(0, -1, 0)
	nextMonth := monthStart.AddDate(0, 1, 0)

	// Navigation row
	navRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Пред. месяц", fmt.Sprintf("month_prev_%s", prevMonth.Format("2006-01-02"))),
		tgbotapi.NewInlineKeyboardButtonData("Текущий", "month_current"),
		tgbotapi.NewInlineKeyboardButtonData("След. месяц ▶️", fmt.Sprintf("month_next_%s", nextMonth.Format("2006-01-02"))),
	)

	// Mode selection row
	modeRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("День", "mode_day"),
		tgbotapi.NewInlineKeyboardButtonData("Неделя", "mode_week"),
		tgbotapi.NewInlineKeyboardButtonData("★ Месяц", "mode_month"),
	)

	// If there are schedules, add buttons for quick view of specific days with events
	var dayButtons [][]tgbotapi.InlineKeyboardButton
	if len(schedules) > 0 {
		// Find days with events
		daysWithEvents := make(map[int]bool)
		for _, s := range schedules {
			daysWithEvents[s.ScheduleTime.Day()] = true
		}

		// Create buttons for days with events (up to 5 days to avoid overcrowding)
		var dayBtns []tgbotapi.InlineKeyboardButton
		count := 0
		for day := 1; day <= 31; day++ {
			if daysWithEvents[day] && count < 5 {
				dateStr := time.Date(monthStart.Year(), monthStart.Month(), day, 0, 0, 0, 0, monthStart.Location()).Format("2006-01-02")
				dayBtns = append(dayBtns,
					tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d", day), fmt.Sprintf("day_%s", dateStr)))
				count++

				// Create a new row every 3 buttons
				if len(dayBtns) == 3 || (count == len(daysWithEvents) && len(dayBtns) > 0) {
					dayButtons = append(dayButtons, dayBtns)
					dayBtns = []tgbotapi.InlineKeyboardButton{}
				}
			}
		}

		// Add any remaining buttons
		if len(dayBtns) > 0 {
			dayButtons = append(dayButtons, dayBtns)
		}
	}

	// Combine all rows
	var allRows [][]tgbotapi.InlineKeyboardButton
	allRows = append(allRows, navRow)
	allRows = append(allRows, modeRow)
	for _, row := range dayButtons {
		allRows = append(allRows, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(allRows...)
}
