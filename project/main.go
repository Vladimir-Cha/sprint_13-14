package main

import (
	"log"
	"net/http"
	"os"
	"project/dbcreate"

	"project/handlers"

	"github.com/gorilla/mux"
)

func main() {
	dbcreate.DataFile() //Функцию тянем из database.go. Открываем файл scheduler.db. Если файла нет, то создаем в текущей папке ../data/scheduler.db

	r := mux.NewRouter()

	// Регистрируем обработчики
	handlers.RegisterHandlers(r)

	// Подключение локальной папки
	webDir := "../web"
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(webDir)))

	// Запуск сервера
	port := os.Getenv("needPort")
	if port == "" {
		port = ":7540"
	}

	log.Printf("Сервер запущен на порту %s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
