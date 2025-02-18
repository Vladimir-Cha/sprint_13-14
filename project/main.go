package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Vladimir-Cha/sprint_13-14/project/project/dbcreate"
	"github.com/Vladimir-Cha/sprint_13-14/project/project/handlers"

	"github.com/Vladimir-Cha/sprint_13-14/project/project/dbopen"
	"github.com/gorilla/mux"
)

func main() {
	dbcreate.DataFile() //Функцию тянем из database.go. Открываем файл scheduler.db. Если файла нет, то создаем в текущей папке ../data/scheduler.db
	db := dbopen.DB()
	defer db.Close()

	handlers.InitDB()
	handlers.InitAuth(os.Getenv("TODO_PASSWORD"))

	r := mux.NewRouter()

	// Регистрируем обработчики
	handlers.RegisterHandlers(r)

	// Подключение локальной папки
	webDir := "../web"
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(webDir)))

	// Запуск сервера
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = ":7540"
	}

	log.Printf("Сервер запущен на порту %s\n", port)

	// Обработка сигналов для остановки сервера
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Fatal(http.ListenAndServe(port, r))
	}()

	<-stop
	log.Println("Сервер остановлен")
}
