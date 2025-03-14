package handlers

import (
	"education/internal/auth"
	"fmt"

	// Функции GetAllFaculties и GetGroupsByFaculty определены в faculty.go
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// processRegistrationMessage — обрабатывает ввод от пользователя в ходе регистрации.
// processRegistrationMessage — обрабатывает ввод от пользователя в ходе регистрации.
// processRegistrationMessage — обрабатывает ввод от пользователя в ходе регистрации.
// processRegistrationMessage — обрабатывает ввод от пользователя в ходе регистрации.
// processRegistrationMessage — обрабатывает ввод от пользователя в ходе регистрации.
// processRegistrationMessage — обрабатывает ввод от пользователя в ходе регистрации.
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
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка: факультет или группа не выбраны.")
			sendAndTrackMessage(bot, msg)
			return
		}
		// Поиск студента по выбранным факультету, группе и введённому регистрационному коду
		userInDB, err := auth.FindUnregisteredUser(tempData.Faculty, tempData.Group, text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if userInDB == nil {
			msg := tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код). Попробуйте ещё раз.")
			sendAndTrackMessage(bot, msg)
			return
		}
		tempData.FoundUserID = userInDB.ID
		userStates[chatID] = StateWaitingForPassword
		msg := tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:")
		sendAndTrackMessage(bot, msg)
		return

	case StateWaitingForPassword:
		if tempData.FoundUserID == 0 {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка регистрации. Начните заново с /register.")
			sendAndTrackMessage(bot, msg)
			return
		}
		userInDB, err := auth.GetUserByID(tempData.FoundUserID)
		if err != nil || userInDB == nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Пользователь не найден (возможно, уже зарегистрирован).")
			sendAndTrackMessage(bot, msg)
			return
		}
		userInDB.TelegramID = chatID
		userInDB.Password = text
		// Переносим поле Faculty, выбранное ранее пользователем
		userInDB.Faculty = tempData.Faculty
		userInDB.Group = tempData.Group
		if err := auth.SaveUser(userInDB); err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка сохранения пользователя. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}

		// Показываем главное меню с данными пользователя
		sendMainMenu(chatID, bot, userInDB)

		// Сбрасываем состояния
		delete(userStates, chatID)
		delete(userTempDataMap, chatID)
		return

	case StateTeacherWaitingForPass:
		userInDB, err := auth.GetUserByRegCode(text) // data = введённый код, например "TR-345"
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if userInDB == nil {
			msg := tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код). Попробуйте ещё раз.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if userInDB.Role != "teacher" {
			msg := tgbotapi.NewMessage(chatID, "❌ Этот код не принадлежит преподавателю.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if userInDB.TelegramID != 0 {
			msg := tgbotapi.NewMessage(chatID, "❌ Этот код уже зарегистрирован другим пользователем.")
			sendAndTrackMessage(bot, msg)
			return
		}

		// ВАЖНО: Проверяем, совпадает ли faculty в БД с выбранным преподавателем
		if userInDB.Faculty != "" && userInDB.Faculty != tempData.Faculty {
			msg := tgbotapi.NewMessage(chatID,
				fmt.Sprintf("❌ Вы выбрали '%s', но этот код преподавателя принадлежит факультету: %s",
					tempData.Faculty, userInDB.Faculty))
			sendAndTrackMessage(bot, msg)
			return
		}

		// Если всё ок, переходим к вводу пароля
		userTempDataMap[chatID].FoundUserID = userInDB.ID
		userStates[chatID] = StateTeacherWaitingForPassword
		msg := tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:")
		sendAndTrackMessage(bot, msg)
		return

	case StateTeacherWaitingForPassword:
		if tempData.FoundUserID == 0 {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка регистрации. Начните заново с /register.")
			sendAndTrackMessage(bot, msg)
			return
		}
		userInDB, err := auth.GetUserByID(tempData.FoundUserID)
		if err != nil || userInDB == nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Пользователь не найден (возможно, уже зарегистрирован).")
			sendAndTrackMessage(bot, msg)
			return
		}
		userInDB.TelegramID = chatID
		userInDB.Password = text

		if err := auth.SaveUser(userInDB); err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка сохранения пользователя. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}

		// Показываем главное меню с данными пользователя без удаления сообщений
		sendMainMenu(chatID, bot, userInDB)

		// Сбрасываем состояния
		delete(userStates, chatID)
		delete(userTempDataMap, chatID)
		return

	}
}
func RegistrationProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// --- 0) Отмена процесса ---
	if data == "cancel_process" {
		edit := tgbotapi.NewEditMessageReplyMarkup(
			chatID,
			callback.Message.MessageID,
			tgbotapi.InlineKeyboardMarkup{},
		)
		bot.Request(edit)

		if userStates[chatID] != "" {
			delete(userStates, chatID)
			delete(userTempDataMap, chatID)
		}
		if loginStates[chatID] != "" {
			delete(loginStates, chatID)
			delete(loginTempDataMap, chatID)
		}

		deleteMessages(chatID, bot, 4)
		msg := tgbotapi.NewMessage(chatID, "❌ Процесс отменён.")
		sendAndTrackMessage(bot, msg)
		user, _ := auth.GetUserByTelegramID(chatID)
		sendMainMenu(chatID, bot, user)
		return
	}

	// --- 1) Проверяем наличие состояния регистрации ---
	state, exists := userStates[chatID]
	if !exists {
		bot.Request(tgbotapi.NewCallback(callback.ID, "Нечего выбирать в данный момент."))
		return
	}

	// --- 2) Удаляем клавиатуру текущего сообщения ---
	edit := tgbotapi.NewEditMessageReplyMarkup(
		chatID,
		callback.Message.MessageID,
		tgbotapi.InlineKeyboardMarkup{},
	)
	bot.Request(edit)

	// --- 3) Обрабатываем шаг регистрации ---
	switch state {
	case StateWaitingForRole:
		if data == "role_student" {
			userTempDataMap[chatID].Role = "student"
			userStates[chatID] = StateWaitingForFaculty
			bot.Request(tgbotapi.NewCallback(callback.ID, "Студент выбран"))
			sendFacultySelection(chatID, bot)
		} else if data == "role_teacher" {
			userTempDataMap[chatID].Role = "teacher"
			// Для преподавателя сразу переходим к выбору факультета
			userStates[chatID] = StateWaitingForFaculty
			bot.Request(tgbotapi.NewCallback(callback.ID, "Преподаватель выбран"))
			sendFacultySelection(chatID, bot)
		}

	case StateWaitingForFaculty:
		// Записываем выбранный факультет
		userTempDataMap[chatID].Faculty = data
		bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Факультет '%s' выбран", data)))
		if userTempDataMap[chatID].Role == "teacher" {
			// Для преподавателей переход сразу к вводу регистрационного кода
			userStates[chatID] = StateTeacherWaitingForPass
			msg := tgbotapi.NewMessage(chatID, "🔐 Введите ваш регистрационный код (например, TR-345):")
			sendAndTrackMessage(bot, msg)
		} else {
			// Для студентов переходим к выбору группы
			userStates[chatID] = StateWaitingForGroup
			sendGroupSelection(chatID, userTempDataMap[chatID].Faculty, bot)
		}

	case StateWaitingForGroup:
		// Выбор группы (только для студентов)
		userTempDataMap[chatID].Group = data
		userStates[chatID] = StateWaitingForPass
		bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Группа '%s' выбрана", data)))
		msg := tgbotapi.NewMessage(chatID, "🔐 Введите ваш регистрационный код (например, ST-456):")
		sendAndTrackMessage(bot, msg)
		return

	case StateWaitingForPass:
		// Обработка кода студента и переход к установке пароля
		if userTempDataMap[chatID].Faculty == "" || userTempDataMap[chatID].Group == "" {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка: факультет или группа не выбраны.")
			sendAndTrackMessage(bot, msg)
			return
		}
		userInDB, err := auth.FindUnregisteredUser(userTempDataMap[chatID].Faculty, userTempDataMap[chatID].Group, data)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if userInDB == nil {
			msg := tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код). Попробуйте ещё раз.")
			sendAndTrackMessage(bot, msg)
			return
		}
		userTempDataMap[chatID].FoundUserID = userInDB.ID
		userStates[chatID] = StateWaitingForPassword
		msg := tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:")
		sendAndTrackMessage(bot, msg)
		return

	case StateTeacherWaitingForPass:
		// Обработка кода преподавателя и переход к установке пароля
		userInDB, err := auth.GetUserByRegCode(data)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if userInDB == nil {
			msg := tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код). Попробуйте ещё раз.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if userInDB.Role != "teacher" {
			msg := tgbotapi.NewMessage(chatID, "❌ Этот код не принадлежит преподавателю.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if userInDB.TelegramID != 0 {
			msg := tgbotapi.NewMessage(chatID, "❌ Этот код уже зарегистрирован другим пользователем.")
			sendAndTrackMessage(bot, msg)
			return
		}
		userTempDataMap[chatID].FoundUserID = userInDB.ID
		userStates[chatID] = StateTeacherWaitingForPassword
		msg := tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:")
		sendAndTrackMessage(bot, msg)
		return

	case StateWaitingForPassword, StateTeacherWaitingForPassword:
		bot.Request(tgbotapi.NewCallback(callback.ID, "Эта операция уже выполнена."))
	}
}

// sendRoleSelection — отправляет inline‑кнопки для выбора роли.
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
		tgbotapi.NewInlineKeyboardButtonData("Отмена Регистрации", "cancel_process"),
	))

	msg := tgbotapi.NewMessage(chatID, "👤 Выберите вашу роль (или отмените операцию):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	sendAndTrackMessage(bot, msg)
}

// sendFacultySelection — отправляет inline‑кнопки факультетов с использованием кэша.
// sendFacultySelection — отправляет inline‑кнопки факультетов с использованием кэша.
func sendFacultySelection(chatID int64, bot *tgbotapi.BotAPI) {
	facs := GetFaculties()
	if len(facs) == 0 {
		var err error
		facs, err = GetAllFaculties()
		if err != nil || len(facs) == 0 {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Нет факультетов для выбора.")
			sendAndTrackMessage(bot, msg)
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
		tgbotapi.NewInlineKeyboardButtonData("Отмена Регистрации", "cancel_process"),
	))

	msg := tgbotapi.NewMessage(chatID, "📚 Выберите ваш факультет (или отмените операцию):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	sendAndTrackMessage(bot, msg)
}

// sendGroupSelection — отправляет inline‑кнопки групп для выбранного факультета с использованием кэша.
// sendGroupSelection — отправляет inline‑кнопки групп для выбранного факультета с использованием кэша.
func sendGroupSelection(chatID int64, facultyName string, bot *tgbotapi.BotAPI) {
	groups := GetGroups(facultyName)
	if len(groups) == 0 {
		var err error
		groups, err = GetGroupsByFaculty(facultyName)
		if err != nil || len(groups) == 0 {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Нет групп для факультета "+facultyName+".")
			sendAndTrackMessage(bot, msg)
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
		tgbotapi.NewInlineKeyboardButtonData("Отмена Регистрации", "cancel_process"),
	))

	msg := tgbotapi.NewMessage(chatID, "📖 Выберите вашу группу (или отмените операцию):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	sendAndTrackMessage(bot, msg)
}
