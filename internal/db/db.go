package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dbFile string) {
	var err error
	DB, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Panicf("Ошибка открытия SQLite: %v", err)
	}
	createTables()
}

func createTables() {
	// Таблица users
	_, err := DB.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id INTEGER,           -- 0, если не привязан
		role TEXT,
		name TEXT,
		group_name TEXT,
		password TEXT,
		registration_code TEXT UNIQUE  -- код (ST-456 и т.д.) уникален
	);
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы users: %v", err)
	}

	// Таблица faculty_groups
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS faculty_groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		faculty TEXT,
		group_name TEXT
	);
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы faculty_groups: %v", err)
	}

	seedUsers()
	seedFaculties()
}

// seedUsers вставляет «предустановленных» участников (ST-456 …) если таблица пуста
func seedUsers() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		log.Panicf("Ошибка при проверке таблицы users: %v", err)
	}
	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO users (telegram_id, role, name, group_name, password, registration_code)
			VALUES
			 (0, 'student', 'Иван Иванов',       'AA-25-07', '', 'ST-456'),
			 (0, 'teacher', 'Петр Петров',       'BB-10-07', '', 'TR-345'),
			 (0, 'student', 'Светлана Соколова', 'AA-25-08', '', 'ST-457'),
			 (0, 'student', 'Мария Смирнова',    'CC-15-01', '', 'ST-459'),
			 (0, 'teacher', 'Алексей Козлов',    'BB-10-08', '', 'TR-346'),
			 (0, 'admin',   'Елена Васильева',   'CC-15-02', '', 'AD-314'),
			 (0, 'student', 'Сергей Иванов',     'AA-25-07', '', 'ST-458'),
			 (0, 'teacher', 'Ольга Новикова',    'BB-10-07', '', 'TR-347'),
			 (0, 'student', 'Дмитрий Соколов',   'CC-15-01', '', 'ST-460'),
			 (0, 'student', 'Анна Кузнецова',    'EE-20-01', '', 'ST-461')
			 ON CONFLICT(registration_code) DO UPDATE SET telegram_id=0;
		`)
		if err != nil {
			log.Panicf("Ошибка вставки seed-пользователей: %v", err)
		}
		log.Println("Дефолтные пользователи добавлены в таблицу users.")
	}
}

// seedFaculties вставляет список факультетов и групп, если faculty_groups пуста
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
