package models

type TeacherCourseGroup struct {
	ID             int64
	TeacherRegCode string // Изменили поле для хранения кода
	CourseID       int64
	GroupName      string
}
