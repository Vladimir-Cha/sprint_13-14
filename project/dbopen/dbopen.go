package dbopen

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// DB возвращает подключение к базе данных
func DB() *sqlx.DB {
	db, err := sqlx.Open("sqlite3", "./data/scheduler.db")
	if err != nil {
		log.Fatal("Ошибка при подключении к базе данных:", err)
	}
	return db
}
