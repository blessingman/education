package main

import (
	"log"
	"os"

	"education/internal/db"
	"education/internal/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Инициализируем SQLite, seed-пользователей => "education.db"
	db.InitDB("education.db")

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	commands := []tgbotapi.BotCommand{
		{Command: "register", Description: "Зарегистрироваться"},
		{Command: "login", Description: "Войти"},
		{Command: "logout", Description: "Выйти из системы"},
		{Command: "cancel", Description: "Отменить операцию"},
	}

	if _, err := bot.Request(tgbotapi.NewSetMyCommands(commands...)); err != nil {
		log.Printf("Ошибка установки команд: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.CallbackQuery != nil {
			handlers.ProcessCallback(update.CallbackQuery, bot)
			continue
		}
		if update.Message != nil {
			handlers.ProcessMessage(&update, bot)
		}
	}
}
