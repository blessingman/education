package handlers

import (
	"database/sql"
	"education/internal/db"
	"education/internal/models"
	"fmt"
)

// FindVerifiedParticipant ищет верифицированного участника в памяти (verifiedParticipants)
// по (faculty, group, pass). Возвращает *VerifiedParticipant, если нашёл, иначе nil.
func FindVerifiedParticipantInDB(faculty, group, pass string) (*models.User, bool) {
	row := db.DB.QueryRow(`
        SELECT id, telegram_id, role, name, group_name, password, registration_code
        FROM users
        WHERE registration_code = ?
          AND group_name = ?
          AND telegram_id = 0
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
