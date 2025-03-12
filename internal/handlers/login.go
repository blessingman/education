package handlers

import (
	"fmt"

	"education/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/crypto/bcrypt"
)

// processLoginMessage обрабатывает входящие текстовые сообщения для входа (логина).
func processLoginMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI, state, text string) {
	chatID := update.Message.Chat.ID
	ld, ok := loginTempDataMap[chatID]
	if !ok {
		ld = &loginData{MsgIDs: []int{}}
		loginTempDataMap[chatID] = ld
	}

	switch state {
	case LoginStateWaitingForRegCode:
		ld.RegCode = text
		loginStates[chatID] = LoginStateWaitingForPassword
		bot.Send(tgbotapi.NewMessage(chatID, "Введите ваш пароль:"))
		return
	case LoginStateWaitingForPassword:
		regCode := ld.RegCode
		user, exists := models.UsersByRegCode[regCode]
		if !exists {
			bot.Send(tgbotapi.NewMessage(chatID, "Пользователь с таким пропуском не найден. Попробуйте снова."))
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(text)); err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Неверный пароль. Попробуйте ещё раз:"))
			return
		}
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Вход выполнен успешно! Добро пожаловать, %s", user.Name)))
		delete(loginStates, chatID)
		delete(loginTempDataMap, chatID)
		return
	}
}
