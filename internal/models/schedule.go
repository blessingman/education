package models

import "time"

type Schedule struct {
	ID             int64
	CourseID       int64
	GroupName      string
	TeacherRegCode string
	ScheduleTime   time.Time
	Description    string

	Auditory   string // Новое поле: аудитория
	LessonType string // Новое поле: тип занятия (лекция/практика/семинар)
	Duration   int    // Продолжительность в минутах, например
	// TeacherName string // Если хотите дублировать ФИО преподавателя прямо в расписании
}
