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
	seedMaterials() // Добавляем функцию для генерации материалов
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

// Генерация курсов
func seedCourses() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM courses`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы courses: %v", err)
	}
	if count == 0 {
		courseNames := []string{
			"Математика", "Программирование", "Физика", "Алгоритмы и структуры данных",
			"Теория вероятностей", "Линейная алгебра", "Электродинамика", "Квантовая механика",
			"Базы данных", "Сети и телекоммуникации", "Механика", "Оптика",
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

		// Генерация связей: каждый преподаватель ведет 3–5 курсов для групп своего факультета
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

// Генерация расписания
// Генерация расписания
// Обновлённая функция генерации расписания с дополнительными полями
func seedSchedules() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM schedules`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы schedules: %v", err)
	}
	if count == 0 {
		// Получаем все связи преподавателей, курсов и групп
		rows, err := DB.Query(`SELECT teacher_reg_code, course_id, group_name FROM teacher_course_groups`)
		if err != nil {
			log.Panicf("Ошибка получения teacher_course_groups: %v", err)
		}
		defer rows.Close()

		var teacherCourseGroups []struct {
			teacherRegCode string
			courseID       int64
			groupName      string
		}
		for rows.Next() {
			var tcg struct {
				teacherRegCode string
				courseID       int64
				groupName      string
			}
			if err := rows.Scan(&tcg.teacherRegCode, &tcg.courseID, &tcg.groupName); err != nil {
				log.Panicf("Ошибка сканирования teacher_course_groups: %v", err)
			}
			teacherCourseGroups = append(teacherCourseGroups, tcg)
		}

		// Группируем связи по группам, чтобы ограничить количество занятий в день
		groupSchedules := make(map[string][]struct {
			teacherRegCode string
			courseID       int64
		})
		for _, tcg := range teacherCourseGroups {
			groupSchedules[tcg.groupName] = append(groupSchedules[tcg.groupName], struct {
				teacherRegCode string
				courseID       int64
			}{tcg.teacherRegCode, tcg.courseID})
		}

		// Доступные временные слоты в день (например, 9:00, 11:00, 13:00, 15:00)
		timeSlots := []int{9, 11, 13, 15}

		// Дополнительные данные для расписания
		auditoryOptions := []string{"101", "102", "103", "104", "201", "202"}
		lessonTypes := []string{"Лекция", "Практика", "Лабораторная", "Семинар"}
		// Варианты длительностей занятий (в минутах)
		lessonDurations := []int{60, 90, 120}

		// Генерация расписания на 3 недели, 4 рабочих дня в неделю, 3–4 занятия в день
		startDate := time.Date(2025, 3, 17, 0, 0, 0, 0, time.UTC) // Начало с понедельника

		for groupName, tcgs := range groupSchedules {
			for week := 0; week < 3; week++ {
				for day := 0; day < 4; day++ { // Пн–Чт
					// Случайно выбираем 3–4 занятия из доступных для группы
					classesPerDay := rand.Intn(2) + 3 // 3–4 занятия в день
					if classesPerDay > len(tcgs) {
						classesPerDay = len(tcgs) // Не можем выбрать больше, чем есть связей
					}

					// Перемешиваем список связей, чтобы выбрать случайные занятия
					shuffledTcgs := make([]struct {
						teacherRegCode string
						courseID       int64
					}, len(tcgs))
					copy(shuffledTcgs, tcgs)
					rand.Shuffle(len(shuffledTcgs), func(i, j int) {
						shuffledTcgs[i], shuffledTcgs[j] = shuffledTcgs[j], shuffledTcgs[i]
					})

					// Выбираем случайные временные слоты без пересечений
					usedSlots := make(map[int]bool)
					for i := 0; i < classesPerDay; i++ {
						// Если все слоты заняты, прерываем
						if len(usedSlots) >= len(timeSlots) {
							break
						}

						// Выбираем случайный незанятый слот
						var slot int
						for {
							slot = timeSlots[rand.Intn(len(timeSlots))]
							if !usedSlots[slot] {
								usedSlots[slot] = true
								break
							}
						}

						// Время начала занятия
						scheduleTime := startDate.AddDate(0, 0, week*7+day).Add(time.Duration(slot) * time.Hour)
						description := fmt.Sprintf("Занятие по курсу для группы %s", groupName)
						auditory := auditoryOptions[rand.Intn(len(auditoryOptions))]
						lessonType := lessonTypes[rand.Intn(len(lessonTypes))]
						duration := lessonDurations[rand.Intn(len(lessonDurations))] // Выбираем случайную длительность

						_, err := DB.Exec(`
                            INSERT INTO schedules (course_id, group_name, teacher_reg_code, schedule_time, description, auditory, lesson_type, duration)
                            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                        `,
							shuffledTcgs[i].courseID,
							groupName,
							shuffledTcgs[i].teacherRegCode,
							scheduleTime.Format(time.RFC3339),
							description,
							auditory,
							lessonType,
							duration,
						)
						if err != nil {
							log.Panicf("Ошибка вставки расписания: %v", err)
						}
					}
				}
			}
		}
		log.Println("Дефолтное расписание добавлено в таблицу schedules.")
	}
}

// Генерация материалов
// Функция для генерации материалов

// Функция для генерации материалов
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
			"Практическое задание",
			"Лабораторная работа",
			"Тест",
			"Дополнительный материал",
		}

		// Генерация одного материала для каждого курса и группы
		for _, cg := range courseGroups {
			// Выбираем случайный тип материала
			materialType := materialTypes[rand.Intn(len(materialTypes))]
			title := fmt.Sprintf("%s для группы %s", materialType, cg.groupName)
			description := fmt.Sprintf("Описание материала для группы %s", cg.groupName)
			fileURL := fmt.Sprintf("https://example.com/materials/%d/%s", cg.courseID, title)

			// Вставляем один материал в таблицу materials, включая teacher_reg_code
			_, err := DB.Exec(`
                INSERT INTO materials (course_id, group_name, teacher_reg_code, title, description, file_url)
                VALUES (?, ?, ?, ?, ?, ?)
            `, cg.courseID, cg.groupName, cg.teacherRegCode, title, description, fileURL)
			if err != nil {
				log.Printf("Ошибка вставки материала для курса %d и группы %s: %v", cg.courseID, cg.groupName, err)
				continue // Пропускаем ошибку и продолжаем вставку других материалов
			}
		}

		log.Println("Дефолтные материалы добавлены в таблицу materials (по одному на курс для каждой группы).")
	}
}
