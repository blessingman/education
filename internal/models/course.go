package models

// Course представляет собой учебный курс
type Course struct {
	ID   int64  // ID курса
	Name string // Название курса
	// Description поле удалено, так как его нет в БД
}
