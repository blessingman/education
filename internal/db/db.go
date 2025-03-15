package db

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB - глобальная переменная для доступа к базе данных.
var DB *sql.DB

// InitDB инициализирует базу данных, создает таблицы и заполняет их тестовыми данными.
func InitDB(dbFile string) {
	var err error
	DB, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Panicf("Ошибка открытия SQLite: %v", err)
	}
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	createTables()
	SeedData() // Вызов функции генерации тестовых данных
}

// createTables создает все необходимые таблицы в базе данных.
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

	// 5) Таблица для расписания
	// 5) Таблица для расписания
	_, err = DB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schedules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			course_id INTEGER NOT NULL,
			group_name TEXT NOT NULL,
			teacher_reg_code TEXT NOT NULL,
			schedule_time DATETIME NOT NULL,
			description TEXT,
			auditory TEXT,
			lesson_type TEXT,
			duration INT,           -- <--- ВАЖНО: новая колонка
			FOREIGN KEY(course_id) REFERENCES courses(id)
		);
	`)
	if err != nil {
		log.Panicf("Ошибка создания таблицы schedules: %v", err)
	}

	// 6) Таблица для материалов
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
}
