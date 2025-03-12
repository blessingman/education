package auth

import (
	"strings"

	"education/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SaveUser(u *models.User) {
	models.UsersMap[u.TelegramID] = u
}

// RegisterUser регистрирует пользователя с ролью "student" по умолчанию.
// Ожидается ввод данных в формате: /register Имя;Группа
func RegisterUser(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	args := strings.TrimSpace(update.Message.CommandArguments())
	if args == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, введите данные в формате: Имя;Группа")
		bot.Send(msg)
		return
	}

	parts := strings.Split(args, ";")
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат. Используйте: Имя;Группа")
		bot.Send(msg)
		return
	}

	name := strings.TrimSpace(parts[0])
	group := strings.TrimSpace(parts[1])

	// Сохраняем пользователя с ролью "student" по умолчанию.
	models.UsersMap[update.Message.Chat.ID] = &models.User{
		TelegramID: update.Message.Chat.ID,
		Role:       "student",
		Group:      group,
		Name:       name,
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Регистрация прошла успешно! Добро пожаловать, "+name)
	bot.Send(msg)
}
