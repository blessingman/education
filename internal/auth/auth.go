package auth

import (
	"database/sql"
	"fmt"

	"education/internal/db"
	"education/internal/models"
)

// SaveUser записывает / обновляет пользователя (по id).
func SaveUser(u *models.User) error {
	_, err := db.DB.Exec(`
		INSERT INTO users (id, telegram_id, role, name, faculty, group_name, password, registration_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			telegram_id = excluded.telegram_id,
			role = excluded.role,
			name = excluded.name,
			faculty = excluded.faculty,
			group_name = excluded.group_name,
			password = excluded.password,
			registration_code = excluded.registration_code
	`,
		u.ID,
		u.TelegramID,
		u.Role,
		u.Name,
		u.Faculty,
		u.Group,
		u.Password,
		u.RegistrationCode,
	)
	if err != nil {
		return fmt.Errorf("SaveUser: %w", err)
	}
	return nil
}

// FindUnregisteredUser ищет пользователя (telegram_id=0) по group_name / registration_code
func FindUnregisteredUser(faculty, group, pass string) (*models.User, error) {
	row := db.DB.QueryRow(`
		SELECT id, telegram_id, role, name, group_name, password, registration_code
		FROM users
		WHERE group_name = ?
		  AND registration_code = ?
		  AND telegram_id = 0
	`, group, pass)

	var u models.User
	err := row.Scan(&u.ID, &u.TelegramID, &u.Role, &u.Name, &u.Group, &u.Password, &u.RegistrationCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Проверить faculty? — если хотите сверять faculty=group_name? Или хранить faculty отдельно?
	// Это зависит от структуры
	return &u, nil
}

// GetUserByTelegramID проверяет, есть ли пользователь с данным telegram_id
func GetUserByTelegramID(telegramID int64) (*models.User, error) {
	row := db.DB.QueryRow(`
		SELECT id, telegram_id, role, name, faculty, group_name, password, registration_code
		FROM users
		WHERE telegram_id = ?
	`, telegramID)
	var u models.User
	err := row.Scan(&u.ID, &u.TelegramID, &u.Role, &u.Name, &u.Faculty, &u.Group, &u.Password, &u.RegistrationCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetUserByTelegramID: %w", err)
	}
	return &u, nil
}

// DeleteUserByTelegramID => /logout
func DeleteUserByTelegramID(telegramID int64) error {
	_, err := db.DB.Exec(`DELETE FROM users WHERE telegram_id=?`, telegramID)
	return err
}

// GetUserByRegCode ищет пользователя (telegram_id != 0 или 0) по registration_code
func GetUserByRegCode(regCode string) (*models.User, error) {
	row := db.DB.QueryRow(`
		SELECT id, telegram_id, role, name, faculty, group_name, password, registration_code
		FROM users
		WHERE registration_code = ?
	`, regCode)
	var u models.User
	err := row.Scan(&u.ID, &u.TelegramID, &u.Role, &u.Name, &u.Faculty, &u.Group, &u.Password, &u.RegistrationCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetUserByRegCode: %w", err)
	}
	return &u, nil
}

// GetUserByID обновлена аналогичным образом
func GetUserByID(id int64) (*models.User, error) {
	row := db.DB.QueryRow(`
		SELECT id, telegram_id, role, name, faculty, group_name, password, registration_code
		FROM users
		WHERE id = ?
	`, id)
	var u models.User
	err := row.Scan(&u.ID, &u.TelegramID, &u.Role, &u.Name, &u.Faculty, &u.Group, &u.Password, &u.RegistrationCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetUserByID: %w", err)
	}
	return &u, nil
}

// FindUnregisteredUser ищет запись (telegram_id=0) для faculty/group/pass
