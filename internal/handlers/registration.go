package handlers

import (
	"education/internal/auth"
	"fmt"
	"regexp"
	"strings"

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

	// Trim spaces from input
	text = strings.TrimSpace(text)

	switch state {
	case StateWaitingForPass:
		if tempData.Faculty == "" || tempData.Group == "" {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка: факультет или группа не выбраны.")
			sendAndTrackMessage(bot, msg)
			return
		}

		// Validate registration code format
		if !validateRegCode(text, "ST-") {
			msg := tgbotapi.NewMessage(chatID, "❌ Некорректный формат кода. Пример: ST-456")
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
		msg := tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль (минимум 6 символов):")
		sendAndTrackMessage(bot, msg)
		return

	case StateWaitingForPassword:
		if !validatePassword(text) {
			msg := tgbotapi.NewMessage(chatID, "❌ Пароль слишком короткий или небезопасный. Используйте минимум 6 символов.")
			sendAndTrackMessage(bot, msg)
			return
		}

		if err := completeRegistration(chatID, tempData.FoundUserID, text, tempData.Faculty, tempData.Group); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ Ошибка: %s", err.Error()))
			sendAndTrackMessage(bot, msg)
			return
		}

		// Получаем пользователя для показа меню
		userInDB, _ := auth.GetUserByID(tempData.FoundUserID)

		msg := tgbotapi.NewMessage(chatID, "🎉 Регистрация успешно завершена!")
		sendAndTrackMessage(bot, msg)

		// Показываем главное меню с данными пользователя
		sendMainMenu(chatID, bot, userInDB)

		// Сбрасываем состояния
		delete(userStates, chatID)
		delete(userTempDataMap, chatID)
		return

	case StateTeacherWaitingForPass:
		// Validate teacher registration code format
		if !validateRegCode(text, "TH-") {
			msg := tgbotapi.NewMessage(chatID, "❌ Некорректный формат кода преподавателя. Пример: TR-345")
			sendAndTrackMessage(bot, msg)
			return
		}

		userInDB, err := auth.FindUnregisteredTeacher(text)
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
		msg := tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль (минимум 6 символов):")
		sendAndTrackMessage(bot, msg)
		return

	case StateTeacherWaitingForPassword:
		if !validatePassword(text) {
			msg := tgbotapi.NewMessage(chatID, "❌ Пароль слишком короткий или небезопасный. Используйте минимум 6 символов.")
			sendAndTrackMessage(bot, msg)
			return
		}

		if err := completeTeacherRegistration(chatID, tempData.FoundUserID, text, tempData.Faculty); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ Ошибка: %s", err.Error()))
			sendAndTrackMessage(bot, msg)
			return
		}

		// Получаем пользователя для показа меню
		userInDB, _ := auth.GetUserByID(tempData.FoundUserID)

		msg := tgbotapi.NewMessage(chatID, "🎉 Регистрация преподавателя успешно завершена!")
		sendAndTrackMessage(bot, msg)

		sendMainMenu(chatID, bot, userInDB)
		delete(userStates, chatID)
		delete(userTempDataMap, chatID)
		return
	}
}

// Helper functions to reduce code duplication

// validateRegCode validates registration code format
func validateRegCode(code string, prefix string) bool {
	pattern := "^" + prefix + "[0-9]{3,4}$" // Example: ST-123 or TR-1234
	match, _ := regexp.MatchString(pattern, code)
	return match
}

// validatePassword checks if password meets security requirements
func validatePassword(password string) bool {
	// Basic validation - at least 6 characters
	return len(password) >= 6
}

// completeRegistration finalizes the student registration process
func completeRegistration(chatID int64, userID int64, password, faculty, group string) error {
	userInDB, err := auth.GetUserByID(userID)
	if err != nil || userInDB == nil {
		return fmt.Errorf("пользователь не найден (возможно, уже зарегистрирован)")
	}

	userInDB.TelegramID = chatID
	userInDB.Password = password // In a production app, this should be hashed
	userInDB.Faculty = faculty
	userInDB.Group = group

	if err := auth.SaveUser(userInDB); err != nil {
		return fmt.Errorf("ошибка сохранения пользователя, попробуйте позже")
	}

	return nil
}

// completeTeacherRegistration finalizes the teacher registration process
func completeTeacherRegistration(chatID int64, userID int64, password, faculty string) error {
	userInDB, err := auth.GetUserByID(userID)
	if err != nil || userInDB == nil {
		return fmt.Errorf("преподаватель не найден (возможно, уже зарегистрирован)")
	}

	userInDB.TelegramID = chatID
	userInDB.Password = password // In a production app, this should be hashed
	userInDB.Faculty = faculty

	if err := auth.SaveUser(userInDB); err != nil {
		return fmt.Errorf("ошибка сохранения пользователя, попробуйте позже")
	}

	return nil
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
			delete(userTempDataMap, chatID) // Fixed: proper delete syntax
		}
		if loginStates[chatID] != "" {
			delete(loginStates, chatID)
			delete(loginTempDataMap, chatID) // Fixed: proper delete syntax
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
			userStates[chatID] = StateWaitingForFaculty
			bot.Request(tgbotapi.NewCallback(callback.ID, "Преподаватель выбран"))
			sendFacultySelection(chatID, bot)
		}

	case StateWaitingForFaculty:
		userTempDataMap[chatID].Faculty = data
		bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Факультет '%s' выбран", data)))
		if userTempDataMap[chatID].Role == "teacher" {
			userStates[chatID] = StateTeacherWaitingForPass
			msg := tgbotapi.NewMessage(chatID, "🔐 Введите ваш регистрационный код (например, TR-345):")
			sendAndTrackMessage(bot, msg)
		} else {
			userStates[chatID] = StateWaitingForGroup
			sendGroupSelection(chatID, userTempDataMap[chatID].Faculty, bot)
		}

	case StateWaitingForGroup:
		userTempDataMap[chatID].Group = data
		userStates[chatID] = StateWaitingForPass
		bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Группа '%s' выбрана", data)))
		msg := tgbotapi.NewMessage(chatID, "🔐 Введите ваш регистрационный код (например, ST-4506):")
		sendAndTrackMessage(bot, msg)
		return

	case StateWaitingForPass:
		// Проверяем, что выбраны факультет и группа
		if userTempDataMap[chatID].Faculty == "" || userTempDataMap[chatID].Group == "" {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка: факультет или группа не выбраны.")
			sendAndTrackMessage(bot, msg)
			return
		}
		// Ищем пользователя по регистрационному коду
		userInDB, err := auth.GetUserByRegCode(data)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}
		// Проверяем, существует ли пользователь
		if userInDB == nil {
			msg := tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код). Попробуйте ещё раз.")
			sendAndTrackMessage(bot, msg)
			return
		}
		// Проверяем, совпадает ли группа
		if userInDB.Group != userTempDataMap[chatID].Group {
			msg := tgbotapi.NewMessage(chatID, "❌ Этот регистрационный код не принадлежит выбранной группе.")
			sendAndTrackMessage(bot, msg)
			return
		}
		// Проверяем, установлен ли пароль
		if userInDB.Password != "" {
			msg := tgbotapi.NewMessage(chatID, "❌ Этот код уже зарегистрирован. Пожалуйста, используйте опцию 'Вход' для авторизации.")
			sendAndTrackMessage(bot, msg)
			return
		}
		// Всё в порядке – сохраняем найденного пользователя и запрашиваем пароль
		userTempDataMap[chatID].FoundUserID = userInDB.ID
		userStates[chatID] = StateWaitingForPassword
		msg := tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:")
		sendAndTrackMessage(bot, msg)
		return

	case StateTeacherWaitingForPass:
		// Ищем преподавателя по регистрационному коду, проверяя, что пароль ещё не установлен
		userInDB, err := auth.FindUnregisteredTeacher(data)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка при поиске в БД. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}
		if userInDB == nil {
			msg := tgbotapi.NewMessage(chatID, "❌ Неверный пропуск (регистрационный код) или код уже зарегистрирован.")
			sendAndTrackMessage(bot, msg)
			return
		}
		// Дополнительно можно проверить, совпадает ли факультет, если требуется
		if userInDB.Faculty != "" && userInDB.Faculty != userTempDataMap[chatID].Faculty {
			msg := tgbotapi.NewMessage(chatID,
				fmt.Sprintf("❌ Вы выбрали '%s', но этот код принадлежит факультету: %s",
					userTempDataMap[chatID].Faculty, userInDB.Faculty))
			sendAndTrackMessage(bot, msg)
			return
		}
		// Всё в порядке – сохраняем найденного пользователя и запрашиваем ввод нового пароля
		userTempDataMap[chatID].FoundUserID = userInDB.ID
		userStates[chatID] = StateTeacherWaitingForPassword
		msg := tgbotapi.NewMessage(chatID, "✅ Код принят. Теперь введите ваш новый пароль:")
		sendAndTrackMessage(bot, msg)
		return

	case StateWaitingForPassword, StateTeacherWaitingForPassword:
		// Обработка ввода нового пароля
		if userTempDataMap[chatID].FoundUserID == 0 {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка регистрации. Начните заново с /register.")
			sendAndTrackMessage(bot, msg)
			return
		}
		userInDB, err := auth.GetUserByID(userTempDataMap[chatID].FoundUserID)
		if err != nil || userInDB == nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Пользователь не найден (возможно, уже зарегистрирован).")
			sendAndTrackMessage(bot, msg)
			return
		}
		userInDB.TelegramID = chatID
		userInDB.Password = data

		if userTempDataMap[chatID].Role != "teacher" {
			userInDB.Faculty = userTempDataMap[chatID].Faculty
			userInDB.Group = userTempDataMap[chatID].Group
		}
		if err := auth.SaveUser(userInDB); err != nil {
			msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка сохранения пользователя. Попробуйте позже.")
			sendAndTrackMessage(bot, msg)
			return
		}

		sendMainMenu(chatID, bot, userInDB)
		delete(userStates, chatID)
		delete(userTempDataMap, chatID)
		return
	}
}
