package models

type Material struct {
	ID             int64  // автоматически генерируется БД
	CourseID       int64  // идентификатор курса
	GroupName      string // группа, для которой материал
	TeacherRegCode string // регистрационный код преподавателя
	Title          string // заголовок материала
	FileURL        string // ссылка на файл (если материал представлен файлом)
	Description    string // описание или дополнительные сведения
}
