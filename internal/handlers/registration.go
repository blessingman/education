package handlers

import (
	"fmt"

	"education/internal/auth"
	// Функции GetAllFaculties и GetGroupsByFaculty определены в faculty.go
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// processRegistrationMessage — обрабатывает ввод от пользователя в ходе регистрации.
func processRegistrationMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI, state, text string) {
	chatID := update.Message.Chat.ID
	tempData, ok := userTempDataMap[chatID]
	if !ok {
		tempData = &tempUserData{}
		userTempDataMap[chatID] = tempData
	}

	switch state {
	// Регистрация для студентов: ввод регистрационного кода (пропуска)
	case StateWaitingForPass:
		if tempData.Faculty == "" || tempData.Group == "" {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка: факультет или группа не выбраны."))
			return
		}
		userInDB, err := auth.FindUnregisteredUser(tempData.Faculty, tempData.Group, text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже."))
			return
		}
		if userInDB == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код). Попробуйте ещё раз."))
			return
		}
		tempData.FoundUserID = userInDB.ID
		userStates[chatID] = StateWaitingForPassword
		bot.Send(tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:"))
		return

	// Завершение регистрации для студентов: ввод пароля
	case StateWaitingForPassword:
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
		sendMainMenu(chatID, bot, userInDB)
		delete(userStates, chatID)
		delete(userTempDataMap, chatID)
		return

	// Регистрация для преподавателей: ввод регистрационного кода (пропуска)
	case StateTeacherWaitingForPass:
		userInDB, err := auth.GetUserByRegCode(text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже."))
			return
		}
		if userInDB == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код). Попробуйте ещё раз."))
			return
		}
		if userInDB.Role != "teacher" {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Этот код не принадлежит преподавателю."))
			return
		}
		if userInDB.TelegramID != 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Этот код уже зарегистрирован другим пользователем."))
			return
		}
		tempData.FoundUserID = userInDB.ID
		userStates[chatID] = StateTeacherWaitingForPassword
		bot.Send(tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:"))
		return

	// Завершение регистрации для преподавателей: ввод пароля
	case StateTeacherWaitingForPassword:
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
			"🎉 Регистрация завершена!\n\n👤 ФИО: %s\n🔑 Роль: %s",
			userInDB.Name,
			userInDB.Role,
		)
		bot.Send(tgbotapi.NewMessage(chatID, finalMsg))
		sendMainMenu(chatID, bot, userInDB)
		delete(userStates, chatID)
		delete(userTempDataMap, chatID)
		return
	}
}

// RegistrationProcessCallback — обрабатывает callback‑запросы при выборе роли, факультета и группы.
// RegistrationProcessCallback — обрабатывает callback-и при выборе роли, факультета, группы и т.д.
func RegistrationProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// --- 0) Проверяем, не нажал ли пользователь «Отмена» ---
	if data == "cancel_process" {
		// Удаляем клавиатуру у старого сообщения (чтобы кнопки не остались кликабельными)
		edit := tgbotapi.NewEditMessageReplyMarkup(
			chatID,
			callback.Message.MessageID,
			tgbotapi.InlineKeyboardMarkup{},
		)
		bot.Request(edit)

		// Сбрасываем состояние регистрации
		if userStates[chatID] != "" {
			delete(userStates, chatID)
			delete(userTempDataMap, chatID)
			bot.Send(tgbotapi.NewMessage(chatID, "Регистрация отменена."))
		}
		// Сбрасываем состояние логина (если вдруг он был)
		if loginStates[chatID] != "" {
			delete(loginStates, chatID)
			delete(loginTempDataMap, chatID)
			bot.Send(tgbotapi.NewMessage(chatID, "Вход отменён."))
		}

		// Показываем главное меню (для авторизованного или нет — зависит от того, был ли пользователь в системе)
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return
	}

	// --- 1) Проверяем, есть ли вообще состояние регистрации ---
	state, exists := userStates[chatID]
	if !exists {
		bot.Request(tgbotapi.NewCallback(callback.ID, "Нечего выбирать в данный момент."))
		return
	}

	// --- 2) Удаляем inline‑клавиатуру из текущего сообщения,
	//         чтобы предотвратить повторное нажатие ---
	edit := tgbotapi.NewEditMessageReplyMarkup(
		chatID,
		callback.Message.MessageID,
		tgbotapi.InlineKeyboardMarkup{},
	)
	bot.Request(edit)

	// --- 3) Обрабатываем шаг регистрации в зависимости от state ---
	switch state {
	case StateWaitingForRole:
		// Пользователь выбирает роль
		if data == "role_student" {
			userTempDataMap[chatID].Role = "student"
			userStates[chatID] = StateWaitingForFaculty
			bot.Request(tgbotapi.NewCallback(callback.ID, "Студент выбран"))
			sendFacultySelection(chatID, bot)

		} else if data == "role_teacher" {
			userTempDataMap[chatID].Role = "teacher"
			userStates[chatID] = StateTeacherWaitingForPass
			bot.Request(tgbotapi.NewCallback(callback.ID, "Преподаватель выбран"))
			bot.Send(tgbotapi.NewMessage(chatID, "Введите ваш регистрационный код (например, TR-345):"))
		}

	case StateWaitingForFaculty:
		// Пользователь выбирает факультет
		userTempDataMap[chatID].Faculty = data
		userStates[chatID] = StateWaitingForGroup
		bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Факультет '%s' выбран", data)))
		// Отправляем меню выбора группы
		sendGroupSelection(chatID, userTempDataMap[chatID].Faculty, bot)

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

// sendRoleSelection — отправляет inline‑кнопки для выбора роли.
func sendRoleSelection(chatID int64, bot *tgbotapi.BotAPI) {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Кнопки выбора роли
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Студент", "role_student"),
		tgbotapi.NewInlineKeyboardButtonData("Преподаватель", "role_teacher"),
	))

	// Кнопка «Отмена»
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Отмена", "cancel_process"),
	))

	msg := tgbotapi.NewMessage(chatID, "👤 Выберите вашу роль (или отмените операцию):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.Send(msg)
}

// sendFacultySelection — отправляет inline‑кнопки факультетов с использованием кэша.
func sendFacultySelection(chatID int64, bot *tgbotapi.BotAPI) {
	facs := GetFaculties()
	if len(facs) == 0 {
		var err error
		facs, err = GetAllFaculties()
		if err != nil || len(facs) == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Нет факультетов для выбора."))
			return
		}
		SetFaculties(facs)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, f := range facs {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(f, f),
		)
		rows = append(rows, row)
	}

	// Кнопка «Отмена»
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Отмена", "cancel_process"),
	))

	msg := tgbotapi.NewMessage(chatID, "📚 Выберите ваш факультет (или отмените операцию):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.Send(msg)
}

// sendGroupSelection — отправляет inline‑кнопки групп для выбранного факультета с использованием кэша.
func sendGroupSelection(chatID int64, facultyName string, bot *tgbotapi.BotAPI) {
	groups := GetGroups(facultyName)
	if len(groups) == 0 {
		var err error
		groups, err = GetGroupsByFaculty(facultyName)
		if err != nil || len(groups) == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Нет групп для факультета "+facultyName+"."))
			return
		}
		SetGroups(facultyName, groups)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, g := range groups {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(g, g),
		)
		rows = append(rows, row)
	}

	// Кнопка «Отмена»
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Отмена", "cancel_process"),
	))

	msg := tgbotapi.NewMessage(chatID, "📖 Выберите вашу группу (или отмените операцию):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.Send(msg)
}
