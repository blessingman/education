package models

type User struct {
	ID               int64
	TelegramID       int64
	Role             string
	Name             string
	Faculty          string
	Group            string // Для преподавателей не используется
	Password         string
	RegistrationCode string
}
