package handlers

import (
	"fmt"

	"education/internal/auth"
	// пакет, где GetAllFaculties / GetGroupsByFaculty
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// processRegistrationMessage — обрабатывает ввод от пользователя в ходе регистрации (после выбора группы).
func processRegistrationMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI, state, text string) {
	chatID := update.Message.Chat.ID
	tempData, ok := userTempDataMap[chatID]
	if !ok {
		tempData = &tempUserData{}
		userTempDataMap[chatID] = tempData
	}

	switch state {

	case StateWaitingForPass:
		// Пользователь вводит registration_code
		if tempData.Faculty == "" || tempData.Group == "" {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка: факультет или группа не выбраны."))
			return
		}
		// Ищем «seed»-пользователя (telegram_id=0) в БД, у которого group_name=? registration_code=?
		userInDB, err := auth.FindUnregisteredUser(tempData.Faculty, tempData.Group, text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже."))
			return
		}
		if userInDB == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код). Попробуйте ещё раз."))
			return
		}
		tempData.FoundUserID = userInDB.ID // Запоминаем ID

		userStates[chatID] = StateWaitingForPassword
		bot.Send(tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:"))
		return

	case StateWaitingForPassword:
		// На этом шаге пользователь вводит пароль
		if tempData.FoundUserID == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка регистрации. Начните заново /register."))
			return
		}
		userInDB, err := auth.GetUserByID(tempData.FoundUserID)
		if err != nil || userInDB == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Пользователь не найден (возможно, уже зарегистрирован)."))
			return
		}
		userInDB.TelegramID = chatID
		userInDB.Password = text
		if err := auth.SaveUser(userInDB); err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка сохранения пользователя. Попробуйте позже."))
			return
		}

		finalMsg := fmt.Sprintf(
			"🎉 Регистрация завершена!\n\n👤 ФИО: %s\n🏫 Факультет: %s\n📚 Группа: %s\n🔑 Роль: %s",
			userInDB.Name, // Предполагаем, что faculty хранить в userInDB не нужно
			tempData.Faculty,
			userInDB.Group,
			userInDB.Role,
		)
		bot.Send(tgbotapi.NewMessage(chatID, finalMsg))

		delete(userStates, chatID)
		delete(userTempDataMap, chatID)
		return
	}
}

// RegistrationProcessCallback — обрабатывает callback-и при выборе факультета / группы.
func RegistrationProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	if state, exists := userStates[chatID]; exists {
		switch state {
		case StateWaitingForFaculty:
			// Выбран факультет
			userTempDataMap[chatID].Faculty = data
			userStates[chatID] = StateWaitingForGroup

			bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Факультет '%s' выбран", data)))
			// Просим выбрать группу
			sendGroupSelection(chatID, data, bot)
			return

		case StateWaitingForGroup:
			// Выбрана группа
			userTempDataMap[chatID].Group = data
			userStates[chatID] = StateWaitingForPass

			bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Группа '%s' выбрана", data)))
			bot.Send(tgbotapi.NewMessage(chatID, "🔐 Введите ваш регистрационный код (например, ST-456):"))
			return
		}
	}
}

// sendFacultySelection — отправляет inline-кнопки факультетов из БД
func sendFacultySelection(chatID int64, bot *tgbotapi.BotAPI) {
	// Считываем из базы
	faculties, err := GetAllFaculties()
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка чтения факультетов из БД."))
		return
	}
	if len(faculties) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Нет факультетов в базе."))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, f := range faculties {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(f, f),
		)
		rows = append(rows, row)
	}

	msg := tgbotapi.NewMessage(chatID, "📚 Выберите ваш факультет:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.Send(msg)
}

// sendGroupSelection — отправляет inline-кнопки групп данного факультета
func sendGroupSelection(chatID int64, facultyName string, bot *tgbotapi.BotAPI) {
	groups, err := GetGroupsByFaculty(facultyName)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка чтения групп из БД."))
		return
	}
	if len(groups) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Нет групп для факультета "+facultyName+"."))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, g := range groups {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(g, g),
		)
		rows = append(rows, row)
	}

	msg := tgbotapi.NewMessage(chatID, "📖 Выберите вашу группу:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.Send(msg)
}
