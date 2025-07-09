package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Vladimir-Cha/sprint_13-14/project/project/dbcreate"
	"github.com/Vladimir-Cha/sprint_13-14/project/project/dbopen"
	"github.com/Vladimir-Cha/sprint_13-14/project/project/handlers"
	"github.com/gorilla/mux"
)

func main() {
	dbcreate.DataFile() // Создаем файл базы данных, если его нет
	db := dbopen.DB()
	defer db.Close()

	// Инициализируем хендлеры с передачей подключения к базе данных
	h := handlers.NewHandlers(db)
	h.InitAuth(os.Getenv("TODO_PASSWORD"))

	r := mux.NewRouter()

	// Регистрируем обработчики
	h.RegisterHandlers(r)

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

	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка при запуске сервера: %v", err)
		}
	}()

	<-stop
	log.Println("Сервер остановлен")

	// Graceful shutdown
	if err := srv.Shutdown(nil); err != nil {
		log.Fatalf("Ошибка при остановке сервера: %v", err)
	}
}
