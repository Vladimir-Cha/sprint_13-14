package dbcreate

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Функция для работы с файлом БД. Открываем файл scheduler.db. Если файла нет, то создаем в текущей папке ../data/scheduler.db
func DataFile() {
	// Получаем значение переменной окружения TODO_DBFILE
	dbFile := os.Getenv("TODO_DBFILE")

	// Если переменная окружения не задана, используем путь по умолчанию
	if dbFile == "" {
		workingDir, err := os.Getwd()
		if err != nil {
			log.Fatal("Ошибка при получении директории", err)
		}
		// Создаем путь к БД в папке data
		dbFile = filepath.Join(workingDir, "data", "scheduler.db")
		if err := os.MkdirAll(filepath.Join(workingDir, "data"), 0755); err != nil {
			log.Fatalf("Ошибка при создании директории data: %v\n", err)
		}
	}

	// Проверяем существование файла базы данных
	_, err := os.Stat(dbFile)

	var install bool
	if os.IsNotExist(err) {
		install = true
	} else if err != nil {
		log.Fatalf("Ошибка при проверке существования файла базы данных: %v\n", err)
	}

	if install {
		fmt.Println("Создание базы данных и таблицы scheduler...")
		if err := createDatabase(dbFile); err != nil {
			log.Fatalf("Ошибка при создании базы данных: %v\n", err)
		}
	}
}

func createDatabase(dbFile string) error {
	// Открываем базу данных
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return fmt.Errorf("ошибка при открытии базы данных: %w", err)
	}
	defer db.Close()

	// Создаем таблицу scheduler
	createTableSQL := `CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		title TEXT NOT NULL,
		comment TEXT,
		repeat TEXT CHECK(length(repeat) <= 128)
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка при создании таблицы: %w", err)
	}

	// Создаем индекс на поле date
	createIndexSQL := `CREATE INDEX IF NOT EXISTS idx_date ON scheduler(date);`
	_, err = db.Exec(createIndexSQL)
	if err != nil {
		return fmt.Errorf("ошибка при создании индекса: %w", err)
	}

	fmt.Println("База данных, таблица scheduler и индекс успешно созданы.")

	// Проверка наличия файла
	if _, err := os.Stat(dbFile); err == nil {
		fmt.Println("Файл базы данных успешно создан:", dbFile)
	} else {
		fmt.Println("Файл базы данных не найден:", dbFile)
	}

	return nil
}
