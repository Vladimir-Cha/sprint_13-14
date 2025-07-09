package dbopen

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// DB возвращает подключение к базе данных
func DB() *sqlx.DB {
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = "./data/scheduler.db"
	}

	db, err := sqlx.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal("Ошибка при подключении к базе данных:", err)
	}
	return db
}
