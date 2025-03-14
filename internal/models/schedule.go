package models

import "time"

type Schedule struct {
	ID             int64     // автоматически генерируется БД
	CourseID       int64     // идентификатор курса
	GroupName      string    // группа, для которой расписание
	TeacherRegCode string    // регистрационный код преподавателя
	ScheduleTime   time.Time // время проведения занятия
	Description    string    // описание занятия
}
