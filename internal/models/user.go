package models

// User — одна таблица, хранит и "seed"-участников (telegram_id=0) и реальных.
type User struct {
	ID               int64 // PRIMARY KEY AUTOINCREMENT
	TelegramID       int64
	Role             string
	Name             string
	Group            string
	Password         string
	RegistrationCode string
}
