package handlers

import (
	"database/sql"
	"education/internal/db"
	"education/internal/models"
	"fmt"
	"time"
)

// FindVerifiedParticipant ищет верифицированного участника в памяти (verifiedParticipants)
// по (faculty, group, pass). Возвращает *VerifiedParticipant, если нашёл, иначе nil.
func FindVerifiedParticipantInDB(faculty, group, pass string) (*models.User, bool) {
	row := db.DB.QueryRow(`
        SELECT id, telegram_id, role, name, group_name, password, registration_code
        FROM users
        WHERE registration_code = ?
          AND group_name = ?
    `, pass, group)

	var u models.User
	err := row.Scan(&u.ID, &u.TelegramID, &u.Role, &u.Name, &u.Group, &u.Password, &u.RegistrationCode)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		fmt.Println("Ошибка:", err)
		return nil, false
	}
	return &u, true
}

// GetAllFaculties возвращает список названий факультетов (уникальных)
func GetAllFaculties() ([]string, error) {
	rows, err := db.DB.Query(`SELECT DISTINCT faculty FROM faculty_groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

// GetGroupsByFaculty возвращает все group_name, связанные с данным faculty
func GetGroupsByFaculty(faculty string) ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT group_name
		FROM faculty_groups
		WHERE faculty = ?
	`, faculty)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

// GetScheduleByGroup возвращает расписание для указанной группы.
func GetScheduleByGroup(group string) ([]models.Schedule, error) {
	rows, err := db.DB.Query(`
        SELECT id, course_id, group_name, teacher_reg_code, schedule_time, description
        FROM schedules
        WHERE group_name = ?
        ORDER BY schedule_time
    `, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var scheduleTimeStr string
		if err := rows.Scan(&s.ID, &s.CourseID, &s.GroupName, &s.TeacherRegCode, &scheduleTimeStr, &s.Description); err != nil {
			return nil, err
		}
		// Преобразуем строку в time.Time (формат зависит от того, как сохраняете дату)
		s.ScheduleTime, _ = time.Parse(time.RFC3339, scheduleTimeStr)
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

// GetScheduleByTeacher выполняет выборку всех записей расписания для преподавателя по его регистрационному коду.
func GetScheduleByTeacher(teacherRegCode string) ([]models.Schedule, error) {
	query := `
		SELECT id, course_id, group_name, teacher_reg_code, schedule_time, description
		FROM schedules
		WHERE teacher_reg_code = ?
		ORDER BY schedule_time
	`
	rows, err := db.DB.Query(query, teacherRegCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var scheduleTimeStr string
		if err := rows.Scan(&s.ID, &s.CourseID, &s.GroupName, &s.TeacherRegCode, &scheduleTimeStr, &s.Description); err != nil {
			return nil, err
		}
		// Преобразуем строку времени в тип time.Time
		s.ScheduleTime, err = time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

/*
// GetMaterialsByGroup возвращает список материалов для указанной группы.
func GetMaterialsByGroup(group string) ([]models.Material, error) {
	rows, err := db.DB.Query(`
        SELECT id, course_id, group_name, teacher_reg_code, title, file_url, description
        FROM materials
        WHERE group_name = ?
    `, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []models.Material
	for rows.Next() {
		var m models.Material
		if err := rows.Scan(&m.ID, &m.CourseID, &m.GroupName, &m.TeacherRegCode, &m.Title, &m.FileURL, &m.Description); err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}


// GetSchedulesByTeacher возвращает список расписания для преподавателя по его регистрационному коду.
func GetSchedulesByTeacher(teacherRegCode string) ([]models.Schedule, error) {
	rows, err := db.DB.Query(`
		SELECT id, course_id, group_name, teacher_reg_code, schedule_time, description
		FROM schedules
		WHERE teacher_reg_code = ?
		ORDER BY schedule_time
	`, teacherRegCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var scheduleTimeStr string
		if err := rows.Scan(&s.ID, &s.CourseID, &s.GroupName, &s.TeacherRegCode, &scheduleTimeStr, &s.Description); err != nil {
			return nil, err
		}
		// Преобразуем строку в time.Time, формат зависит от того, как записаны даты
		s.ScheduleTime, err = time.Parse(time.RFC3339, scheduleTimeStr)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}
*/

// GetAllCourses возвращает список всех курсов из базы данных
func GetAllCourses() ([]models.Course, error) {
	rows, err := db.DB.Query(`
		SELECT id, name FROM courses
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		if err := rows.Scan(&course.ID, &course.Name); err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}
	return courses, rows.Err()
}
