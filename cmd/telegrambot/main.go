package main

import (
	"log"
	"os"

	"education/internal/db"
	"education/internal/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const workerCount = 10 // число воркеров

func main() {
	db.InitDB("education.db")

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	/*
		// Установка команд (если нужно)
		commands := []tgbotapi.BotCommand{
			{Command: "register", Description: "Зарегистрироваться"},
			{Command: "login", Description: "Войти"},
			{Command: "logout", Description: "Выйти из системы"},
			{Command: "help", Description: "Получить справку"},
			{Command: "cancel", Description: "Отменить текущую операцию"},
		}
		setCmds := tgbotapi.NewSetMyCommands(commands...)
		if _, err = bot.Request(setCmds); err != nil {
			log.Printf("Ошибка установки команд: %v", err)
		}
	*/
	deleteCmds := tgbotapi.NewDeleteMyCommands()
	// deleteCmds.Scope = &tgbotapi.BotCommandScopeDefault{} // опционально
	_, err = bot.Request(deleteCmds)
	if err != nil {
		log.Println("Ошибка удаления команд:", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Создаем канал для обновлений
	updateChan := make(chan tgbotapi.Update, 100)

	// Запускаем пул воркеров
	for i := 0; i < workerCount; i++ {
		go worker(i, bot, updateChan)
	}

	// Передаем обновления в канал
	for update := range updates {
		updateChan <- update
	}
}

func worker(id int, bot *tgbotapi.BotAPI, updateChan <-chan tgbotapi.Update) {
	for update := range updateChan {
		if update.CallbackQuery != nil {
			handlers.ProcessCallback(update.CallbackQuery, bot)
		}
		if update.Message != nil {
			handlers.ProcessMessage(&update, bot)
		}
	}
}
