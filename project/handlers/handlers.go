package handlers

import (
	"github.com/gorilla/mux"
)

// RegisterHandlers регистрирует все обработчики API
func RegisterHandlers(r *mux.Router) {
	r.HandleFunc("/api/tasks", GetTasks).Methods("GET")
	r.HandleFunc("/api/task", GetTaskByID).Methods("GET")
	r.HandleFunc("/api/task", UpdateTask).Methods("PUT")
	r.HandleFunc("/api/task", DeleteTask).Methods("DELETE")
	r.HandleFunc("/api/nextdate", GetNextDate).Methods("GET")
	r.HandleFunc("/api/task", CreateTask).Methods("POST")
	r.HandleFunc("/api/task/done", MarkTaskAsDone).Methods("POST")

	r.HandleFunc("/api/signin", SignInHandler).Methods("POST")
	r.HandleFunc("/api/task", AuthMiddleware(CreateTask)).Methods("POST")
	r.HandleFunc("/api/task", AuthMiddleware(GetTaskByID)).Methods("GET")
	r.HandleFunc("/api/task", AuthMiddleware(UpdateTask)).Methods("PUT")
	r.HandleFunc("/api/task", AuthMiddleware(DeleteTask)).Methods("DELETE")
	r.HandleFunc("/api/tasks", AuthMiddleware(GetTasks)).Methods("GET")
	r.HandleFunc("/api/task/done", AuthMiddleware(MarkTaskAsDone)).Methods("POST")
}
