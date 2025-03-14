package db

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dbFile string) {
	var err error
	DB, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Panicf("Ошибка открытия SQLite: %v", err)
	}
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	createTables()
}
func createTables() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1) Таблица users
	_, err := DB.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            telegram_id INTEGER,
            role TEXT,
            name TEXT,
            faculty TEXT,
            group_name TEXT,
            password TEXT,
            registration_code TEXT UNIQUE
        );
    `)
	if err != nil {
		log.Panicf("Ошибка создания таблицы users: %v", err)
	}

	// 2) Таблица faculty_groups
	_, err = DB.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS faculty_groups (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            faculty TEXT,
            group_name TEXT
        );
    `)
	if err != nil {
		log.Panicf("Ошибка создания таблицы faculty_groups: %v", err)
	}

	// 3) Таблица courses
	_, err = DB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS courses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL
		);
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы courses: %v", err)
	}

	// 4) Таблица teacher_course_groups
	_, err = DB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS teacher_course_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			teacher_reg_code TEXT NOT NULL,
			course_id INTEGER NOT NULL,
			group_name TEXT NOT NULL,
			FOREIGN KEY(course_id) REFERENCES courses(id)
		);
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы teacher_course_groups: %v", err)
	}

	// Таблица для расписания
	_, err = DB.ExecContext(ctx, `
	    CREATE TABLE IF NOT EXISTS schedules (
	        id INTEGER PRIMARY KEY AUTOINCREMENT,
	        course_id INTEGER NOT NULL,
	        group_name TEXT NOT NULL,
	        teacher_reg_code TEXT NOT NULL,
	        schedule_time DATETIME NOT NULL,
	        description TEXT,
	        FOREIGN KEY(course_id) REFERENCES courses(id)
	    );
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы schedules: %v", err)
	}

	// Таблица для материалов
	_, err = DB.ExecContext(ctx, `
	    CREATE TABLE IF NOT EXISTS materials (
	        id INTEGER PRIMARY KEY AUTOINCREMENT,
	        course_id INTEGER NOT NULL,
	        group_name TEXT NOT NULL,
	        teacher_reg_code TEXT NOT NULL,
	        title TEXT NOT NULL,
	        file_url TEXT,
	        description TEXT,
	        FOREIGN KEY(course_id) REFERENCES courses(id)
	    );
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы materials: %v", err)
	}

	seedStudents()
	seedTeachers()
	seedFaculties()
	seedCourses()
	seedSchedules()
	seedTeacherCourseGroups()
}

func seedStudents() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'student'`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы users для студентов: %v", err)
	}
	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO users (telegram_id, role, name, faculty, group_name, password, registration_code)
			VALUES
			 (0, 'student', 'Иван Иванов',       'Факультет Информатики', 'AA-25-07', '', 'ST-456'),
			 (0, 'student', 'Светлана Соколова', 'Факультет Информатики', 'AA-25-08', '', 'ST-457'),
			 (0, 'student', 'Мария Смирнова',    'Факультет Физики',      'CC-15-01', '', 'ST-459'),
			 (0, 'student', 'Сергей Иванов',     'Факультет Информатики', 'AA-25-07', '', 'ST-458'),
			 (0, 'student', 'Дмитрий Соколов',   'Факультет Физики',      'CC-15-01', '', 'ST-460'),
			 (0, 'student', 'Анна Кузнецова',    'Факультет Экономики',   'EE-20-01', '', 'ST-461')
			 ON CONFLICT(registration_code) DO UPDATE SET telegram_id=0;
		`)
		if err != nil {
			log.Panicf("Ошибка вставки студентов: %v", err)
		}
		log.Println("Дефолтные студенты добавлены в таблицу users.")
	}
}

func seedTeachers() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'teacher'`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы users для преподавателей: %v", err)
	}
	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO users (telegram_id, role, name, faculty, group_name, password, registration_code)
			VALUES
			 (0, 'teacher', 'Петр Петров',     'Факультет Механики',    '', '', 'TR-345'),
			 (0, 'teacher', 'Алексей Козлов',  'Факультет Механики',    '', '', 'TR-346'),
			 (0, 'teacher', 'Ольга Новикова',  'Факультет Механики',    '', '', 'TR-347'),
			 (0, 'teacher', 'Иван Смирнов',    'Факультет Информатики', '', '', 'TR-348'),
			 (0, 'teacher', 'Екатерина Иванова','Факультет Информатики', '', '', 'TR-349'),
			 (0, 'teacher', 'Николай Петров',  'Факультет Экономики',   '', '', 'TR-350'),
			 (0, 'teacher', 'Мария Козлова',   'Факультет Физики',      '', '', 'TR-351')
			 ON CONFLICT(registration_code) DO UPDATE SET telegram_id=0;
		`)
		if err != nil {
			log.Panicf("Ошибка вставки преподавателей: %v", err)
		}
		log.Println("Дефолтные преподаватели добавлены в таблицу users.")
	}
}

func seedFaculties() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM faculty_groups`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке faculty_groups: %v", err)
	}
	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO faculty_groups (faculty, group_name)
			VALUES
			 ('Факультет Информатики', 'AA-25-07'),
			 ('Факультет Информатики', 'AA-25-08'),
			 ('Факультет Информатики', 'AA-25-09'),
			 ('Факультет Механики',    'BB-10-07'),
			 ('Факультет Механики',    'BB-10-08'),
			 ('Факультет Физики',      'CC-15-01'),
			 ('Факультет Физики',      'CC-15-02'),
			 ('Факультет Экономики',   'EE-20-01'),
			 ('Факультет Экономики',   'EE-20-02')
		`)
		if err != nil {
			log.Panicf("Ошибка вставки факультетов: %v", err)
		}
		log.Println("Дефолтные факультеты и группы добавлены в faculty_groups.")
	}
}

func seedCourses() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM courses`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы courses: %v", err)
	}
	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO courses (name)
			VALUES
				('Математика'),
				('Программирование'),
				('Физика'),
				('Экономика')
		`)
		if err != nil {
			log.Panicf("Ошибка вставки курсов: %v", err)
		}
		log.Println("Дефолтные курсы добавлены в таблицу courses.")
	}
}

func seedTeacherCourseGroups() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM teacher_course_groups`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке teacher_course_groups: %v", err)
	}
	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO teacher_course_groups (teacher_reg_code, course_id, group_name)
			VALUES
			-- Преподаватели из Факультета Механики
			('TR-345', 1, 'BB-10-07'),
			('TR-345', 1, 'BB-10-08'),
			('TR-346', 2, 'BB-10-08'),
			('TR-346', 2, 'CC-15-01'),
			('TR-347', 3, 'BB-10-07'),
			-- Преподаватели из Факультета Информатики
			('TR-348', 1, 'AA-25-07'),
			('TR-348', 1, 'AA-25-08'),
			('TR-348', 1, 'AA-25-09'),
			('TR-349', 2, 'AA-25-07'),
			-- Преподаватели из Факультета Экономики
			('TR-350', 4, 'EE-20-01'),
			('TR-350', 4, 'EE-20-02'),
			-- Преподаватели из Факультета Физики
			('TR-351', 3, 'CC-15-01'),
			('TR-351', 3, 'CC-15-02')
		`)
		if err != nil {
			log.Panicf("Ошибка вставки teacher_course_groups: %v", err)
		}
		log.Println("Связи преподавателей, курсов и групп добавлены в teacher_course_groups.")
	}
}

func seedSchedules() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM schedules`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы schedules: %v", err)
	}
	if count == 0 {
		// Добавляем дефолтное расписание.
		// Примеры занятий по разным дням:
		// Курс 1 (Математика) для групп 'BB-10-07' и 'BB-10-08'
		// Курс 2 (Программирование) для групп 'BB-10-08', 'CC-15-01' и 'AA-25-07'
		// Курс 3 (Физика) для групп 'BB-10-07' и 'CC-15-01'
		// Курс 4 (Экономика) для групп 'EE-20-01' и 'EE-20-02'
		_, err = DB.Exec(`
			INSERT INTO schedules (course_id, group_name, teacher_reg_code, schedule_time, description)
			VALUES
			  -- Математика занятия
			  (1, 'BB-10-07', 'TR-345', '2025-03-20T10:00:00Z', 'Лекция по математическому анализу'),
			  (1, 'BB-10-08', 'TR-345', '2025-03-23T09:00:00Z', 'Семинар по математике'),
			  (1, 'BB-10-07', 'TR-348', '2025-03-24T10:00:00Z', 'Дополнительное занятие по математике'),
			  
			  -- Программирование занятия
			  (2, 'BB-10-08', 'TR-346', '2025-03-21T12:00:00Z', 'Практическое занятие по программированию'),
			  (2, 'CC-15-01', 'TR-346', '2025-03-24T11:00:00Z', 'Лекция по программированию'),
			  (2, 'AA-25-07', 'TR-349', '2025-03-25T12:00:00Z', 'Семинар по программированию'),
			  
			  -- Физика занятия
			  (3, 'BB-10-07', 'TR-347', '2025-03-22T14:00:00Z', 'Лабораторная работа по физике'),
			  (3, 'BB-10-07', 'TR-347', '2025-03-25T15:00:00Z', 'Практическая работа по физике'),
			  (3, 'CC-15-01', 'TR-351', '2025-03-26T14:00:00Z', 'Лекция по физике'),
			  
			  -- Экономика занятия
			  (4, 'EE-20-01', 'TR-350', '2025-03-20T16:00:00Z', 'Лекция по экономике'),
			  (4, 'EE-20-02', 'TR-350', '2025-03-23T16:00:00Z', 'Семинар по экономике'),
			  (4, 'EE-20-01', 'TR-350', '2025-03-27T16:00:00Z', 'Практическое занятие по экономике')
		`)
		if err != nil {
			log.Panicf("Ошибка вставки расписания: %v", err)
		}
		log.Println("Дефолтное расписание добавлено в таблицу schedules.")
	}
}
