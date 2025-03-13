package handlers

import "education/internal/db"

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
