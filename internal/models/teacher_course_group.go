package models

type TeacherCourseGroup struct {
	ID        int64
	TeacherID int64
	CourseID  int64
	GroupName string
}
