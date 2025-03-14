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

	seedStudents()
	seedTeachers()
	seedFaculties()
	seedCourses()
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
			 (0, 'teacher', 'Ольга Новикова',  'Факультет Механики',    '', '', 'TR-347')
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
		// По скриншоту: Пётр Петров (id=6), Алексей Козлов (id=7), Ольга Новикова (id=8).
		// Courses: 1=Математика, 2=Программирование, 3=Физика, 4=Экономика (пример).
		_, err = DB.Exec(`
			INSERT INTO teacher_course_groups (teacher_reg_code, course_id, group_name)
			VALUES
			('TR-345', 1, 'BB-10-07'),
			('TR-345', 1, 'BB-10-08'),
			('TR-346', 2, 'BB-10-08'),
			('TR-346', 2, 'CC-15-01'),
			('TR-347', 3, 'BB-10-07');
		`)
		if err != nil {
			log.Panicf("Ошибка вставки teacher_course_groups: %v", err)
		}
		log.Println("Связи преподавателей, курсов и групп добавлены в teacher_course_groups.")
	}
}
