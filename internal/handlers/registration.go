package handlers

import (
	"fmt"

	"education/internal/auth"
	"education/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// processRegistrationMessage обрабатывает текстовые сообщения для регистрации.
func processRegistrationMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI, state, text string) {
	chatID := update.Message.Chat.ID
	tempData, ok := userTempDataMap[chatID]
	if !ok {
		tempData = &tempUserData{}
		userTempDataMap[chatID] = tempData
	}

	switch state {
	case StateWaitingForPass:
		if tempData.Faculty == "" || tempData.Group == "" {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка: факультет или группа не выбраны."))
			return
		}
		vp, found := FindVerifiedParticipant(tempData.Faculty, tempData.Group, text)
		if !found {
			bot.Send(tgbotapi.NewMessage(chatID, "Неверный регистрационный код. Попробуйте ещё раз:"))
			return
		}
		tempData.Verified = vp
		userStates[chatID] = StateWaitingForPassword
		bot.Send(tgbotapi.NewMessage(chatID, "Регистрационный код принят. Установите, пожалуйста, ваш новый пароль:"))
		return
	case StateWaitingForPassword:
		if tempData.Verified == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка регистрации. Попробуйте снова."))
			return
		}
		vp := tempData.Verified
		newUser := &models.User{
			TelegramID:       chatID,
			Role:             vp.Role,
			Name:             vp.FIO,
			Group:            vp.Group,
			Password:         text,
			RegistrationCode: vp.Pass,
		}
		auth.SaveUser(newUser)
		finalMsg := fmt.Sprintf("Регистрация завершена!\nФИО: %s\nФакультет: %s\nГруппа: %s\nРоль: %s",
			newUser.Name, vp.Faculty, newUser.Group, vp.Role)
		bot.Send(tgbotapi.NewMessage(chatID, finalMsg))
		delete(userStates, chatID)
		delete(userTempDataMap, chatID)
		return
	}
}

// sendFacultySelection отправляет инлайн-клавиатуру для выбора факультета.
func sendFacultySelection(chatID int64, bot *tgbotapi.BotAPI) {
	var rows [][]tgbotapi.InlineKeyboardButton
	for faculty := range faculties {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(faculty, faculty),
		))
	}
	msg := tgbotapi.NewMessage(chatID, "Выберите ваш факультет:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.Send(msg)
}

// sendGroupSelection отправляет инлайн-клавиатуру для выбора группы для выбранного факультета.
func sendGroupSelection(chatID int64, faculty string, bot *tgbotapi.BotAPI) {
	groups, exists := faculties[faculty]
	if !exists {
		bot.Send(tgbotapi.NewMessage(chatID, "Факультет не найден."))
		return
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, group := range groups {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(group, group),
		))
	}
	msg := tgbotapi.NewMessage(chatID, "Выберите вашу группу:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.Send(msg)
}

// RegistrationProcessCallback обрабатывает callback‑запросы для регистрации.
func RegistrationProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	if state, exists := userStates[chatID]; exists {
		switch state {
		case StateWaitingForFaculty:
			userTempDataMap[chatID].Faculty = data
			userStates[chatID] = StateWaitingForGroup
			bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Факультет '%s' выбран", data)))
			sendGroupSelection(chatID, data, bot)
			return
		case StateWaitingForGroup:
			userTempDataMap[chatID].Group = data
			userStates[chatID] = StateWaitingForPass
			bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Группа '%s' выбрана", data)))
			bot.Send(tgbotapi.NewMessage(chatID, "Введите ваш регистрационный код:"))
			return
		}
	}
}
