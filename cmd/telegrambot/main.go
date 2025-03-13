package main

import (
	"log"
	"os"

	"education/internal/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Создаем экземпляр бота с использованием токена.
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Здесь задаём команды
	commands := []tgbotapi.BotCommand{
		{
			Command:     "register",
			Description: "Зарегистрироваться",
		},
		{
			Command:     "login",
			Description: "Войти",
		},
		{
			Command:     "logout",
			Description: "Выйти из системы",
		},
		{
			Command:     "help",
			Description: "Получить справку",
		},
		{
			Command:     "cancel",
			Description: "Отказаться от операции",
		},
	}

	// Создаём запрос на установку команд
	setCmds := tgbotapi.NewSetMyCommands(commands...)

	// Отправляем запрос в Telegram
	_, err = bot.Request(setCmds)
	if err != nil {
		log.Printf("Ошибка установки команд: %v", err)
	}

	// Настройка получения обновлений.
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Обработка обновлений: как текстовые сообщения, так и callback-запросы.
	for update := range updates {
		if update.CallbackQuery != nil {
			handlers.ProcessCallback(update.CallbackQuery, bot)
			continue
		}
		if update.Message != nil {
			handlers.ProcessMessage(&update, bot) // Передаем указатель
		}
	}
}
