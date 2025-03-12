package models

// User хранит информацию о пользователе.
type User struct {
	TelegramID       int64  // Идентификатор чата в Telegram
	Role             string // Роль: "student", "teacher", "admin"
	Group            string // Название группы
	Name             string // ФИО пользователя
	Password         string // Пароль (хранится в открытом виде; в реальном проекте — в зашифрованном виде)
	RegistrationCode string // Регистрационный код (пропуск)
}

// UsersMap — глобальное хранилище пользователей по TelegramID.
var UsersMap = make(map[int64]*User)

// UsersByRegCode — глобальное хранилище пользователей по RegistrationCode.
var UsersByRegCode = make(map[string]*User)
