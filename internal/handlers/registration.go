package handlers

import (
	"fmt"

	"education/internal/auth"
	// Функции GetAllFaculties и GetGroupsByFaculty определены в faculty.go
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
		// Пользователь вводит registration_code (пропуск)
		if tempData.Faculty == "" || tempData.Group == "" {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка: факультет или группа не выбраны."))
			return
		}
		// Ищем «seed»-пользователя (telegram_id=0) в БД, у которого group_name=? и registration_code=?
		userInDB, err := auth.FindUnregisteredUser(tempData.Faculty, tempData.Group, text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже."))
			return
		}
		if userInDB == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код). Попробуйте ещё раз."))
			return
		}
		// Запоминаем ID найденного пользователя
		tempData.FoundUserID = userInDB.ID

		// Переходим к вводу пароля
		userStates[chatID] = StateWaitingForPassword
		bot.Send(tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:"))
		return

	case StateWaitingForPassword:
		// На этом шаге пользователь вводит пароль
		if tempData.FoundUserID == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка регистрации. Начните заново с /register."))
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
			userInDB.Name,
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

	// Если для данного чата нет состояния, уведомляем пользователя
	if state, exists := userStates[chatID]; !exists {
		bot.Request(tgbotapi.NewCallback(callback.ID, "Нечего выбирать в данный момент."))
		return
	} else {
		// Обновляем сообщение, удаляя inline‑клавиатуру, чтобы предотвратить повторное нажатие
		edit := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
		bot.Request(edit)

		switch state {
		case StateWaitingForFaculty:
			// Пользователь выбирает факультет
			userTempDataMap[chatID].Faculty = data
			userStates[chatID] = StateWaitingForGroup

			bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Факультет '%s' выбран", data)))
			// Отправляем меню выбора группы
			sendGroupSelection(chatID, data, bot)
		case StateWaitingForGroup:
			// Пользователь выбирает группу
			userTempDataMap[chatID].Group = data
			userStates[chatID] = StateWaitingForPass

			bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Группа '%s' выбрана", data)))
			bot.Send(tgbotapi.NewMessage(chatID, "🔐 Введите ваш регистрационный код (например, ST-456):"))
		default:
			// Если состояние не соответствует ни одному ожидаемому шагу
			bot.Request(tgbotapi.NewCallback(callback.ID, "Это действие уже выполнено."))
		}
	}
}

// sendFacultySelection — отправляет inline-кнопки факультетов с использованием in‑memory кэша.
func sendFacultySelection(chatID int64, bot *tgbotapi.BotAPI) {
	// Сначала пытаемся получить факультеты из кэша.
	facs := GetFaculties()
	if len(facs) == 0 {
		// Если кэш пуст, загружаем данные из БД
		var err error
		facs, err = GetAllFaculties()
		if err != nil || len(facs) == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Нет факультетов для выбора."))
			return
		}
		// Обновляем кэш
		SetFaculties(facs)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, f := range facs {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(f, f),
		)
		rows = append(rows, row)
	}

	msg := tgbotapi.NewMessage(chatID, "📚 Выберите ваш факультет:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.Send(msg)
}

// sendGroupSelection — отправляет inline-кнопки групп для выбранного факультета с использованием in‑memory кэша.
func sendGroupSelection(chatID int64, facultyName string, bot *tgbotapi.BotAPI) {
	// Пытаемся получить группы для факультета из кэша.
	groups := GetGroups(facultyName)
	if len(groups) == 0 {
		var err error
		groups, err = GetGroupsByFaculty(facultyName)
		if err != nil || len(groups) == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Нет групп для факультета "+facultyName+"."))
			return
		}
		// Обновляем кэш
		SetGroups(facultyName, groups)
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
