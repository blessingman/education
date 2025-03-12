package handlers

import (
	"fmt"
	"strings"

	"education/internal/auth"
	"education/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Константы состояний регистрации.
const (
	StateWaitingForFaculty = "waiting_for_faculty"
	StateWaitingForGroup   = "waiting_for_group"
	StateWaitingForFIO     = "waiting_for_fio"
	StateWaitingForPass    = "waiting_for_pass"
)

// VerifiedParticipant хранит данные верифицированного участника.
type VerifiedParticipant struct {
	FIO     string // ФИО участника
	Faculty string // Разрешённый факультет
	Group   string // Разрешённая группа (например, "AA-25-07")
	Pass    string // Индивидуальный пропуск
	Role    string // Роль: "student", "teacher" и т.д.
}

// Предопределённый список верифицированных участников.
// В будущем через админ-интерфейс можно будет импортировать/редактировать этот список.
var verifiedParticipants = map[string]VerifiedParticipant{
	"Иван Иванов":       {FIO: "Иван Иванов", Faculty: "Факультет Информатики", Group: "AA-25-07", Pass: "ST-456", Role: "student"},
	"Петр Петров":       {FIO: "Петр Петров", Faculty: "Факультет Механики", Group: "BB-10-07", Pass: "TR-345", Role: "teacher"},
	"Светлана Соколова": {FIO: "Светлана Соколова", Faculty: "Факультет Информатики", Group: "AA-25-08", Pass: "ST-456", Role: "student"},
	"Мария Смирнова":    {FIO: "Мария Смирнова", Faculty: "Факультет Физики", Group: "CC-15-01", Pass: "ST-456", Role: "student"},
	"Алексей Козлов":    {FIO: "Алексей Козлов", Faculty: "Факультет Механики", Group: "BB-10-08", Pass: "TR-345", Role: "teacher"},
	"Елена Васильева":   {FIO: "Елена Васильева", Faculty: "Факультет Физики", Group: "CC-15-02", Pass: "AD-314", Role: "admin"},
	"Сергей Иванов":     {FIO: "Сергей Иванов", Faculty: "Факультет Информатики", Group: "AA-25-07", Pass: "ST-456", Role: "student"},
	"Ольга Новикова":    {FIO: "Ольга Новикова", Faculty: "Факультет Механики", Group: "BB-10-07", Pass: "TR-345", Role: "teacher"},
	"Дмитрий Соколов":   {FIO: "Дмитрий Соколов", Faculty: "Факультет Физики", Group: "CC-15-01", Pass: "ST-456", Role: "student"},
	"Анна Кузнецова":    {FIO: "Анна Кузнецова", Faculty: "Факультет Экономики", Group: "EE-20-01", Pass: "ST-456", Role: "student"},
}

// Обновлённый список факультетов и их групп.
var faculties = map[string][]string{
	"Факультет Информатики": {"AA-25-07", "AA-25-08", "AA-25-09"},
	"Факультет Механики":    {"BB-10-07", "BB-10-08"},
	"Факультет Физики":      {"CC-15-01", "CC-15-02"},
	"Факультет Экономики":   {"EE-20-01", "EE-20-02"},
}

// tempUserData хранит временные данные регистрации.
type tempUserData struct {
	Faculty string // Выбранный факультет
	Group   string // Выбранная группа
	Name    string // Введённое ФИО
}

// userStates хранит текущий этап регистрации для каждого пользователя (по chatID).
var userStates = make(map[int64]string)

// userTempDataMap хранит временные данные регистрации для каждого пользователя.
var userTempDataMap = make(map[int64]*tempUserData)

// ProcessMessage обрабатывает входящие текстовые сообщения.
func ProcessMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Если пользователь уже находится в процессе регистрации, обрабатываем этапы.
	if state, exists := userStates[chatID]; exists {
		switch state {
		case StateWaitingForFIO:
			// Сохраняем введённое ФИО и проверяем наличие в верифицированном списке.
			if _, ok := verifiedParticipants[text]; !ok {
				msg := tgbotapi.NewMessage(chatID, "ФИО не найдено в верифицированном списке. Обратитесь к администратору.")
				bot.Send(msg)
				// Завершаем процесс регистрации.
				delete(userStates, chatID)
				delete(userTempDataMap, chatID)
				return
			}
			// Сохраняем ФИО.
			if tempData, ok := userTempDataMap[chatID]; ok {
				tempData.Name = text
			} else {
				userTempDataMap[chatID] = &tempUserData{Name: text}
			}
			// Переходим к вводу пропуска.
			userStates[chatID] = StateWaitingForPass
			msg := tgbotapi.NewMessage(chatID, "Введите ваш пропуск:")
			bot.Send(msg)
			return

		case StateWaitingForPass:
			tempData, ok := userTempDataMap[chatID]
			if !ok || tempData.Name == "" {
				msg := tgbotapi.NewMessage(chatID, "Ошибка регистрации. Попробуйте снова.")
				bot.Send(msg)
				return
			}
			// Получаем данные верифицированного участника по ФИО.
			verified, exists := verifiedParticipants[tempData.Name]
			if !exists {
				msg := tgbotapi.NewMessage(chatID, "ФИО не найдено. Регистрация отклонена.")
				bot.Send(msg)
				delete(userStates, chatID)
				delete(userTempDataMap, chatID)
				return
			}
			// Проверяем, что выбранный факультет и группа совпадают с разрешёнными.
			if tempData.Faculty != verified.Faculty || tempData.Group != verified.Group {
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Выбранные факультет (%s) или группа (%s) не соответствуют разрешённым (%s, %s).",
					tempData.Faculty, tempData.Group, verified.Faculty, verified.Group))
				bot.Send(msg)
				// Предлагаем повторно выбрать факультет.
				userStates[chatID] = StateWaitingForFaculty
				sendFacultySelection(chatID, bot)
				return
			}
			// Проверяем пропуск.
			if text != verified.Pass {
				msg := tgbotapi.NewMessage(chatID, "Неверный пропуск. Попробуйте ещё раз:")
				bot.Send(msg)
				return
			}

			// Все проверки пройдены — завершаем регистрацию.
			newUser := &models.User{
				TelegramID: chatID,
				Role:       verified.Role,
				Name:       verified.FIO,
				Group:      verified.Group,
			}
			auth.SaveUser(newUser)
			delete(userStates, chatID)
			delete(userTempDataMap, chatID)
			msgText := fmt.Sprintf("Регистрация завершена!\nФИО: %s\nФакультет: %s\nГруппа: %s\nРоль: %s",
				newUser.Name, verified.Faculty, newUser.Group, newUser.Role)
			msg := tgbotapi.NewMessage(chatID, msgText)
			bot.Send(msg)
			return

		case StateWaitingForGroup:
			msg := tgbotapi.NewMessage(chatID, "Пожалуйста, выберите группу, нажав на одну из кнопок.")
			bot.Send(msg)
			return

		case StateWaitingForFaculty:
			msg := tgbotapi.NewMessage(chatID, "Пожалуйста, выберите факультет, нажав на одну из кнопок.")
			bot.Send(msg)
			return
		}
	}

	// Если пользователь не в процессе регистрации, обрабатываем команды.
	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			msg := tgbotapi.NewMessage(chatID, "Привет! Используй /register для регистрации.")
			bot.Send(msg)
		case "register":
			// Запускаем регистрацию: сначала выбор факультета.
			userStates[chatID] = StateWaitingForFaculty
			sendFacultySelection(chatID, bot)
		default:
			if _, ok := models.UsersMap[chatID]; !ok {
				msg := tgbotapi.NewMessage(chatID, "Сначала зарегистрируйтесь командой /register")
				bot.Send(msg)
				return
			}
			msg := tgbotapi.NewMessage(chatID, "Команда не распознана или ещё не реализована.")
			bot.Send(msg)
		}
	} else {
		if _, ok := models.UsersMap[chatID]; !ok {
			msg := tgbotapi.NewMessage(chatID, "Для начала используйте /register")
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(chatID, "Вы уже зарегистрированы. Используйте команды для дальнейшей работы.")
			bot.Send(msg)
		}
	}
}

// sendFacultySelection отправляет сообщение с инлайн-клавиатурой для выбора факультета.
func sendFacultySelection(chatID int64, bot *tgbotapi.BotAPI) {
	var buttons [][]tgbotapi.InlineKeyboardButton
	for faculty := range faculties {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(faculty, faculty),
		))
	}
	msg := tgbotapi.NewMessage(chatID, "Выберите ваш факультет:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	bot.Send(msg)
}

// sendGroupSelection отправляет сообщение с инлайн-клавиатурой для выбора группы, привязанных к выбранному факультету.
func sendGroupSelection(chatID int64, faculty string, bot *tgbotapi.BotAPI) {
	groups, exists := faculties[faculty]
	if !exists {
		msg := tgbotapi.NewMessage(chatID, "Факультет не найден.")
		bot.Send(msg)
		return
	}
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, group := range groups {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(group, group),
		))
	}
	msg := tgbotapi.NewMessage(chatID, "Выберите вашу группу:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	bot.Send(msg)
}

// ProcessCallback обрабатывает callback-запросы от инлайн-клавиатуры.
func ProcessCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// Если пользователь выбирает факультет.
	if state, exists := userStates[chatID]; exists && state == StateWaitingForFaculty {
		// Сохраняем выбранный факультет.
		if tempData, ok := userTempDataMap[chatID]; ok {
			tempData.Faculty = data
		} else {
			userTempDataMap[chatID] = &tempUserData{Faculty: data}
		}
		// Переходим к выбору группы для выбранного факультета.
		userStates[chatID] = StateWaitingForGroup
		// Отвечаем на callback.
		answer := tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Факультет '%s' выбран", data))
		if _, err := bot.Request(answer); err != nil {
			// Обработка ошибки при отправке callback-ответа (по желанию)
		}
		// Отправляем список групп для выбранного факультета.
		sendGroupSelection(chatID, data, bot)
		return
	}

	// Если пользователь выбирает группу.
	if state, exists := userStates[chatID]; exists && state == StateWaitingForGroup {
		// Сохраняем выбранную группу.
		if tempData, ok := userTempDataMap[chatID]; ok {
			tempData.Group = data
		} else {
			userTempDataMap[chatID] = &tempUserData{Group: data}
		}
		// Переходим к вводу ФИО.
		userStates[chatID] = StateWaitingForFIO
		answer := tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Группа '%s' выбрана", data))
		if _, err := bot.Request(answer); err != nil {
			// Обработка ошибки при отправке callback-ответа (по желанию)
		}
		// Запрашиваем ввод ФИО.
		msg := tgbotapi.NewMessage(chatID, "Введите ваше ФИО:")
		bot.Send(msg)
	}
}
