package db

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// Списки для генерации реалистичных данных
var firstNames = []string{
	"Иван", "Алексей", "Дмитрий", "Сергей", "Андрей", "Павел", "Николай", "Михаил", "Владимир", "Евгений",
	"Екатерина", "Мария", "Анна", "Ольга", "Татьяна", "Наталья", "Елена", "Светлана", "Юлия", "Ирина",
}

var lastNames = []string{
	"Иванов", "Петров", "Сидоров", "Смирнов", "Кузнецов", "Васильев", "Попов", "Соколов", "Михайлов", "Новиков",
	"Федоров", "Морозов", "Волков", "Алексеев", "Лебедев", "Семенов", "Егоров", "Павлов", "Козлов", "Степанов",
}

var faculties = []string{
	"Факультет Информатики",
	"Факультет Физики",
}

var prefixes = []string{"АА", "ББ"}
var startYears = []int{23, 24, 25}

const subgroupCount = 1 // Уменьшаем до 1 подгруппы на факультет и год, итого 3 группы на факультет

// SeedData заполняет базу данных тестовыми данными.
func SeedData() {
	seedFacultiesAndGroups()
	seedCourses()
	seedStudents()
	seedTeachers()
	seedTeacherCourseGroups()
	seedSchedules()
	seedMaterials() // Генерация материалов
}

// Генерация групп в формате АА-21-01, АА-21-02 и т.д.
func generateGroupNames(prefix string, startYear int, subgroupCount int) []string {
	var groups []string
	for i := 1; i <= subgroupCount; i++ {
		group := fmt.Sprintf("%s-%02d-%02d", prefix, startYear, i)
		groups = append(groups, group)
	}
	return groups
}

// Генерация факультетов и групп
func seedFacultiesAndGroups() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM faculty_groups`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке faculty_groups: %v", err)
	}
	if count == 0 {
		// Генерация групп для каждого факультета
		for fIdx, faculty := range faculties {
			prefix := prefixes[fIdx] // Каждому факультету соответствует уникальный префикс
			for _, year := range startYears {
				groups := generateGroupNames(prefix, year, subgroupCount)
				for _, group := range groups {
					_, err := DB.Exec(`
						INSERT INTO faculty_groups (faculty, group_name)
						VALUES (?, ?)
					`, faculty, group)
					if err != nil {
						log.Panicf("Ошибка вставки факультета и группы: %v", err)
					}
				}
			}
		}
		log.Println("Дефолтные факультеты и группы добавлены в faculty_groups.")
	}
}

// Генерация курсов с более короткими и разнообразными названиями
func seedCourses() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM courses`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы courses: %v", err)
	}
	if count == 0 {
		courseNames := []string{
			"Матем", "Прог", "Физ", "Алгос", "Вероят", "Линейка", "Электро", "Квант", "База", "Сети", "Мех", "Опт", "Стат", "Дискр", "Комп",
		}
		for _, courseName := range courseNames {
			_, err := DB.Exec(`
				INSERT INTO courses (name)
				VALUES (?)
			`, courseName)
			if err != nil {
				log.Panicf("Ошибка вставки курса: %v", err)
			}
		}
		log.Println("Дефолтные курсы добавлены в таблицу courses.")
	}
}

// Генерация студентов
func seedStudents() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'student'`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы users для студентов: %v", err)
	}
	if count == 0 {
		// Получаем все группы из faculty_groups
		rows, err := DB.Query(`SELECT faculty, group_name FROM faculty_groups`)
		if err != nil {
			log.Panicf("Ошибка получения групп: %v", err)
		}
		defer rows.Close()

		var facultyGroups []struct {
			faculty   string
			groupName string
		}
		for rows.Next() {
			var fg struct {
				faculty   string
				groupName string
			}
			if err := rows.Scan(&fg.faculty, &fg.groupName); err != nil {
				log.Panicf("Ошибка сканирования групп: %v", err)
			}
			facultyGroups = append(facultyGroups, fg)
		}

		// Генерация 60 студентов (по 10 на каждую группу)
		for _, fg := range facultyGroups {
			for i := 0; i < 10; i++ {
				name := fmt.Sprintf("%s %s", firstNames[rand.Intn(len(firstNames))], lastNames[rand.Intn(len(lastNames))])
				_, err := DB.Exec(`
					INSERT INTO users (telegram_id, role, name, faculty, group_name, password, registration_code)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, rand.Int63(), "student", name, fg.faculty, fg.groupName, "", fmt.Sprintf("ST-%s-%d", fg.groupName, i+1))
				if err != nil {
					log.Panicf("Ошибка вставки студента: %v", err)
				}
			}
		}
		log.Println("Дефолтные студенты добавлены в таблицу users.")
	}
}

// Генерация преподавателей
func seedTeachers() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'teacher'`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы users для преподавателей: %v", err)
	}
	if count == 0 {
		// Генерация 3 преподавателей на факультет
		for fIdx, faculty := range faculties {
			for tIdx := 0; tIdx < 3; tIdx++ {
				name := fmt.Sprintf("%s %s", firstNames[rand.Intn(len(firstNames))], lastNames[rand.Intn(len(lastNames))])
				_, err := DB.Exec(`
					INSERT INTO users (telegram_id, role, name, faculty, group_name, password, registration_code)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, rand.Int63(), "teacher", name, faculty, "", "", fmt.Sprintf("TR-%d-%d", fIdx+1, tIdx+1))
				if err != nil {
					log.Panicf("Ошибка вставки преподавателя: %v", err)
				}
			}
		}
		log.Println("Дефолтные преподаватели добавлены в таблицу users.")
	}
}

// Генерация связей преподавателей, курсов и групп
func seedTeacherCourseGroups() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM teacher_course_groups`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке teacher_course_groups: %v", err)
	}
	if count == 0 {
		// Получаем всех преподавателей
		rows, err := DB.Query(`SELECT faculty, registration_code FROM users WHERE role = 'teacher'`)
		if err != nil {
			log.Panicf("Ошибка получения преподавателей: %v", err)
		}
		defer rows.Close()

		var teachers []struct {
			faculty string
			regCode string
		}
		for rows.Next() {
			var t struct {
				faculty string
				regCode string
			}
			if err := rows.Scan(&t.faculty, &t.regCode); err != nil {
				log.Panicf("Ошибка сканирования преподавателей: %v", err)
			}
			teachers = append(teachers, t)
		}

		// Получаем все курсы
		courseRows, err := DB.Query(`SELECT id, name FROM courses`)
		if err != nil {
			log.Panicf("Ошибка получения курсов: %v", err)
		}
		defer courseRows.Close()

		var courses []struct {
			id   int64
			name string
		}
		for courseRows.Next() {
			var c struct {
				id   int64
				name string
			}
			if err := courseRows.Scan(&c.id, &c.name); err != nil {
				log.Panicf("Ошибка сканирования курсов: %v", err)
			}
			courses = append(courses, c)
		}

		// Получаем все группы
		rows, err = DB.Query(`SELECT faculty, group_name FROM faculty_groups`)
		if err != nil {
			log.Panicf("Ошибка получения групп: %v", err)
		}
		defer rows.Close()

		var facultyGroups []struct {
			faculty   string
			groupName string
		}
		for rows.Next() {
			var fg struct {
				faculty   string
				groupName string
			}
			if err := rows.Scan(&fg.faculty, &fg.groupName); err != nil {
				log.Panicf("Ошибка сканирования групп: %v", err)
			}
			facultyGroups = append(facultyGroups, fg)
		}

		// Генерация связей: каждый преподаватель ведёт 3–5 курсов для групп своего факультета
		for _, teacher := range teachers {
			// Выбираем группы только с того же факультета, что и преподаватель
			var teacherGroups []string
			for _, fg := range facultyGroups {
				if fg.faculty == teacher.faculty {
					teacherGroups = append(teacherGroups, fg.groupName)
				}
			}
			if len(teacherGroups) == 0 {
				continue
			}

			// Выбираем случайные 3–5 курсов для преподавателя
			courseCount := rand.Intn(3) + 3 // От 3 до 5 курсов
			selectedCourses := make(map[int64]bool)
			for len(selectedCourses) < courseCount {
				course := courses[rand.Intn(len(courses))]
				selectedCourses[course.id] = true
			}

			// Привязываем преподавателя ко всем группам его факультета для каждого курса
			for courseID := range selectedCourses {
				for _, group := range teacherGroups {
					_, err := DB.Exec(`
						INSERT INTO teacher_course_groups (teacher_reg_code, course_id, group_name)
						VALUES (?, ?, ?)
					`, teacher.regCode, courseID, group)
					if err != nil {
						log.Panicf("Ошибка вставки teacher_course_groups: %v", err)
					}
				}
			}
		}
		log.Println("Связи преподавателей, курсов и групп добавлены в teacher_course_groups.")
	}
}

// Генерация расписания для преподавателей на 3 месяца.
// Для каждого преподавателя в рабочие дни (понедельник-пятница) генерируется от 3 до 7 пар.
// При этом выбранные временные слоты сортируются, и для каждой пары длительность подбирается так,
// чтобы конец текущей пары не пересекался со следующим (для последней пары считается, что день заканчивается в 17:00).
func seedSchedules() {
	// Очищаем таблицу schedules (TRUNCATE не поддерживается в SQLite)
	_, err := DB.Exec(`DELETE FROM schedules`)
	if err != nil {
		log.Panicf("Не удалось очистить таблицу schedules: %v", err)
	}
	log.Println("Таблица schedules очищена. Генерация нового расписания...")

	// Получаем все связи преподавателей, курсов и групп
	rows, err := DB.Query(`SELECT teacher_reg_code, course_id, group_name FROM teacher_course_groups`)
	if err != nil {
		log.Panicf("Ошибка получения teacher_course_groups: %v", err)
	}
	defer rows.Close()

	// Группируем назначения (привязки) по преподавателю
	teacherSchedules := make(map[string][]struct {
		teacherRegCode string
		courseID       int64
		groupName      string
	})
	for rows.Next() {
		var tcg struct {
			teacherRegCode string
			courseID       int64
			groupName      string
		}
		if err := rows.Scan(&tcg.teacherRegCode, &tcg.courseID, &tcg.groupName); err != nil {
			log.Panicf("Ошибка сканирования teacher_course_groups: %v", err)
		}
		teacherSchedules[tcg.teacherRegCode] = append(teacherSchedules[tcg.teacherRegCode], tcg)
	}

	// Фиксированные временные слоты (с перерывами):
	//  1-я пара: 08:00 – 09:30
	//  2-я пара: 09:45 – 11:15 (перерыв 15 минут)
	//  3-я пара: 11:45 – 13:15 (перерыв 30 минут)
	lessonSlots := []struct {
		hour     int
		minute   int
		duration int // длительность в минутах
	}{
		{8, 0, 90},   // первая пара
		{9, 45, 90},  // вторая пара
		{11, 45, 90}, // третья пара
	}

	// Диапазон дат (3 месяца начиная с 17 марта 2025)
	startDate := time.Date(2025, 3, 17, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 3, 0)

	// Справочные данные
	auditoryOptions := []string{"101", "102", "103", "104", "201", "202"}
	lessonTypes := []string{"Лекция", "Практика", "Лабораторная", "Семинар"}

	// Карты для отслеживания занятости:
	// - scheduledGroups: "groupName|время" → bool
	// - scheduledTeachers: "teacherRegCode|время" → bool
	scheduledGroups := make(map[string]bool)
	scheduledTeachers := make(map[string]bool)

	for teacherRegCode, assignments := range teacherSchedules {
		if len(assignments) == 0 {
			continue
		}

		// Перебираем дни в заданном диапазоне
		for day := startDate; day.Before(endDate); day = day.AddDate(0, 0, 1) {
			// Пропускаем выходные (суббота и воскресенье)
			if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
				continue
			}

			// Для каждого фиксированного временного слота
			for _, slot := range lessonSlots {
				lessonStart := time.Date(day.Year(), day.Month(), day.Day(), slot.hour, slot.minute, 0, 0, time.UTC)
				utcTime := lessonStart.UTC().Format(time.RFC3339)

				// Перемешиваем назначения, чтобы распределение было случайным
				rand.Shuffle(len(assignments), func(i, j int) {
					assignments[i], assignments[j] = assignments[j], assignments[i]
				})

				slotAssigned := false
				for _, assignment := range assignments {
					keyGroup := fmt.Sprintf("%s|%s", assignment.groupName, utcTime)
					keyTeacher := fmt.Sprintf("%s|%s", teacherRegCode, utcTime)

					if !scheduledGroups[keyGroup] && !scheduledTeachers[keyTeacher] {
						description := fmt.Sprintf("Занятие по курсу для группы %s", assignment.groupName)
						auditory := auditoryOptions[rand.Intn(len(auditoryOptions))]
						lessonType := lessonTypes[rand.Intn(len(lessonTypes))]

						_, err := DB.Exec(`
							INSERT INTO schedules (course_id, group_name, teacher_reg_code, schedule_time, description, auditory, lesson_type, duration)
							VALUES (?, ?, ?, ?, ?, ?, ?, ?)
						`,
							assignment.courseID,
							assignment.groupName,
							teacherRegCode,
							utcTime,
							description,
							auditory,
							lessonType,
							slot.duration,
						)
						if err != nil {
							log.Panicf("Ошибка вставки расписания: %v", err)
						}

						// Помечаем, что группа и преподаватель заняты в этот момент
						scheduledGroups[keyGroup] = true
						scheduledTeachers[keyTeacher] = true

						slotAssigned = true
						break
					}
				}

				if !slotAssigned {
					log.Printf("Слот %s (преподаватель %s) остался свободным — нет доступных групп или возникло пересечение.",
						lessonStart.Format("2006-01-02 15:04"), teacherRegCode)
				}
			}
		}
	}

	log.Println("Генерация расписания завершена. Дубликаты отсутствуют.")
}

// Генерация материалов
func seedMaterials() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM materials`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы materials: %v", err)
	}
	if count == 0 {
		// Получаем все связи курсов, групп и преподавателей из teacher_course_groups
		rows, err := DB.Query(`
            SELECT DISTINCT course_id, group_name, teacher_reg_code 
            FROM teacher_course_groups
        `)
		if err != nil {
			log.Panicf("Ошибка получения teacher_course_groups: %v", err)
		}
		defer rows.Close()

		var courseGroups []struct {
			courseID       int64
			groupName      string
			teacherRegCode string
		}
		for rows.Next() {
			var cg struct {
				courseID       int64
				groupName      string
				teacherRegCode string
			}
			if err := rows.Scan(&cg.courseID, &cg.groupName, &cg.teacherRegCode); err != nil {
				log.Panicf("Ошибка сканирования teacher_course_groups: %v", err)
			}
			courseGroups = append(courseGroups, cg)
		}

		// Возможные типы материалов
		materialTypes := []string{
			"Лекция",
			"Практика",
			"Лабораторка",
			"Тест",
			"Доп. материал",
		}

		// Генерация одного материала для каждого курса и группы
		for _, cg := range courseGroups {
			// Выбираем случайный тип материала
			materialType := materialTypes[rand.Intn(len(materialTypes))]
			title := fmt.Sprintf("%s для группы %s", materialType, cg.groupName)
			description := fmt.Sprintf("Описание материала для группы %s", cg.groupName)
			fileURL := fmt.Sprintf("https://example.com/materials/%d/%s", cg.courseID, title)

			_, err := DB.Exec(`
                INSERT INTO materials (course_id, group_name, teacher_reg_code, title, description, file_url)
                VALUES (?, ?, ?, ?, ?, ?)
            `, cg.courseID, cg.groupName, cg.teacherRegCode, title, description, fileURL)
			if err != nil {
				log.Printf("Ошибка вставки материала для курса %d и группы %s: %v", cg.courseID, cg.groupName, err)
				continue
			}
		}

		log.Println("Дефолтные материалы добавлены в таблицу materials (по одному на курс для каждой группы).")
	}
}
