package models

// User хранит информацию о пользователе.
type User struct {
	TelegramID int64  // Идентификатор чата в Telegram
	Role       string // Роль: "student", "teacher", "admin"
	Group      string // Название группы
	Name       string // Имя пользователя
}

// UsersMap — глобальное хранилище зарегистрированных пользователей.
var UsersMap = make(map[int64]*User)
