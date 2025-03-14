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
	// Настраиваем пул соединений (опционально)
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)

	createTables()
}

func createTables() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1) Таблица users (c полем faculty и group_name)
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

	// 2) Таблица faculty_groups (как было)
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

	// 3) Таблица subjects
	_, err = DB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS subjects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT
		);
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы subjects: %v", err)
	}

	// 4) Таблица teacher_subjects (связь «преподаватель - предмет»)
	_, err = DB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS teacher_subjects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			teacher_id INTEGER NOT NULL,
			subject_id INTEGER NOT NULL,
			FOREIGN KEY(teacher_id) REFERENCES users(id),
			FOREIGN KEY(subject_id) REFERENCES subjects(id)
		);
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы teacher_subjects: %v", err)
	}

	// 5) Таблица teacher_groups (связь «преподаватель - группа»)
	_, err = DB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS teacher_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			teacher_id INTEGER NOT NULL,
			group_name TEXT NOT NULL,
			FOREIGN KEY(teacher_id) REFERENCES users(id)
		);
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы teacher_groups: %v", err)
	}

	seedUsers()
	seedFaculties()
	seedSubjects()
	seedTeacherSubjects()
	seedTeacherGroups()
}

// seedUsers – уже есть, поправьте, если нужно
func seedUsers() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы users: %v", err)
	}
	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO users (telegram_id, role, name, faculty, group_name, password, registration_code)
			VALUES
			 (0, 'student', 'Иван Иванов',       'Факультет Информатики', 'AA-25-07', '', 'ST-456'),
			 (0, 'teacher', 'Петр Петров',       'Факультет Механики',    'BB-10-07', '', 'TR-345'),
			 (0, 'student', 'Светлана Соколова', 'Факультет Информатики', 'AA-25-08', '', 'ST-457'),
			 (0, 'student', 'Мария Смирнова',    'Факультет Физики',      'CC-15-01', '', 'ST-459'),
			 (0, 'teacher', 'Алексей Козлов',    'Факультет Механики',    'BB-10-08', '', 'TR-346'),
			 (0, 'admin',   'Елена Васильева',   'Факультет Физики',      'CC-15-02', '', 'AD-314'),
			 (0, 'student', 'Сергей Иванов',     'Факультет Информатики', 'AA-25-07', '', 'ST-458'),
			 (0, 'teacher', 'Ольга Новикова',    'Факультет Механики',    'BB-10-07', '', 'TR-347'),
			 (0, 'student', 'Дмитрий Соколов',   'Факультет Физики',      'CC-15-01', '', 'ST-460'),
			 (0, 'student', 'Анна Кузнецова',    'Факультет Экономики',   'EE-20-01', '', 'ST-461')
			 ON CONFLICT(registration_code) DO UPDATE SET telegram_id=0;
		`)
		if err != nil {
			log.Panicf("Ошибка вставки seed-пользователей: %v", err)
		}
		log.Println("Дефолтные пользователи добавлены в таблицу users.")
	}
}

// seedFaculties – уже есть, пусть остаётся
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

// seedSubjects – новая функция
func seedSubjects() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM subjects`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы subjects: %v", err)
	}
	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO subjects (name, description)
			VALUES
			 ('Математика', 'Базовый курс математики'),
			 ('Программирование', 'Основы языков программирования'),
			 ('Физика', 'Теоретическая физика'),
			 ('Экономика', 'Микро- и макроэкономика')
		`)
		if err != nil {
			log.Panicf("Ошибка вставки предметов: %v", err)
		}
		log.Println("Дефолтные предметы добавлены в таблицу subjects.")
	}
}

// seedTeacherSubjects – новая функция
func seedTeacherSubjects() {
	// Для упрощения: проверяем, есть ли уже данные (можно и не проверять).
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM teacher_subjects`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке teacher_subjects: %v", err)
	}
	if count == 0 {
		// Пример: предположим, что:
		//  - Петр Петров (id=2) ведёт Математику (id=1) и Физику (id=3).
		//  - Алексей Козлов (id=5) ведёт Программирование (id=2).
		// Нужно сверять ID пользователей и предметов с реальными данными в БД.
		_, err = DB.Exec(`
			INSERT INTO teacher_subjects (teacher_id, subject_id)
			VALUES
			 (2, 1),
			 (2, 3),
			 (5, 2)
		`)
		if err != nil {
			log.Panicf("Ошибка вставки teacher_subjects: %v", err)
		}
		log.Println("Связи преподавателей и предметов добавлены в teacher_subjects.")
	}
}

// seedTeacherGroups – новая функция
func seedTeacherGroups() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM teacher_groups`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке teacher_groups: %v", err)
	}
	if count == 0 {
		// Пример: Петр Петров (id=2) ведёт BB-10-07 и BB-10-08,
		//         Алексей Козлов (id=5) ведёт BB-10-08 и CC-15-01
		_, err = DB.Exec(`
			INSERT INTO teacher_groups (teacher_id, group_name)
			VALUES
			 (2, 'BB-10-07'),
			 (2, 'BB-10-08'),
			 (5, 'BB-10-08'),
			 (5, 'CC-15-01')
		`)
		if err != nil {
			log.Panicf("Ошибка вставки teacher_groups: %v", err)
		}
		log.Println("Связи преподавателей и групп добавлены в teacher_groups.")
	}
}
