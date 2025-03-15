package handlers

import (
	"education/internal/db"
	"education/internal/models"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// GetMaterialsByTeacher возвращает материалы, загруженные преподавателем.
func GetMaterialsByTeacher(teacherRegCode string) ([]models.Material, error) {
	rows, err := db.DB.Query(`
		SELECT id, course_id, group_name, teacher_reg_code, title, file_url, description
		FROM materials
		WHERE teacher_reg_code = ?
	`, teacherRegCode)
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

// GetMaterialsByGroup возвращает материалы для указанной группы (уже есть в коде, но включим для полноты).
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

// FormatMaterials форматирует материалы для удобного отображения.
func FormatMaterials(materials []models.Material, mode string, user *models.User) (string, error) {
	if len(materials) == 0 {
		return "Материалы не найдены.", nil
	}

	// Получаем дополнительную информацию о курсах
	courseMap := make(map[int64]string) // courseID -> courseName
	courses, err := db.DB.Query(`SELECT id, name FROM courses`)
	if err != nil {
		return "", err
	}
	defer courses.Close()
	for courses.Next() {
		var id int64
		var name string
		if err := courses.Scan(&id, &name); err != nil {
			return "", err
		}
		courseMap[id] = name
	}

	// Формируем текст сообщения
	msgText := "Доступные материалы:\n\n"
	for _, m := range materials {
		courseName := courseMap[m.CourseID]
		msgText += fmt.Sprintf("📘 %s:\n", courseName)
		msgText += fmt.Sprintf("  • %s\n", m.Title)
		if m.Description != "" {
			msgText += fmt.Sprintf("    Описание: %s\n", m.Description)
		}
		if m.FileURL != "" {
			msgText += fmt.Sprintf("    Ссылка: %s\n", m.FileURL)
		}
		if mode == "teacher" {
			msgText += fmt.Sprintf("    Группа: %s\n", m.GroupName)
		}
		msgText += "\n"
	}

	return msgText, nil
}

// ShowMaterials отправляет материалы пользователю в Telegram.
func ShowMaterials(chatID int64, bot *tgbotapi.BotAPI, user *models.User) error {
	var materials []models.Material
	var err error
	var mode string

	if user.Role == "teacher" {
		mode = "teacher"
		materials, err = GetMaterialsByTeacher(user.RegistrationCode)
	} else if user.Role == "student" {
		mode = "student"
		materials, err = GetMaterialsByGroup(user.Group)
	} else {
		msg := tgbotapi.NewMessage(chatID, "⚠️ Роль пользователя не определена.")
		return sendAndTrackMessage(bot, msg)
	}

	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка при получении материалов.")
		return sendAndTrackMessage(bot, msg)
	}

	msgText, err := FormatMaterials(materials, mode, user)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "⚠️ Ошибка при форматировании материалов.")
		return sendAndTrackMessage(bot, msg)
	}

	msg := tgbotapi.NewMessage(chatID, msgText)
	return sendAndTrackMessage(bot, msg)
}
