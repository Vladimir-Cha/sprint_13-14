package handlers

import (
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

type Handlers struct {
	db *sqlx.DB
}

func NewHandlers(db *sqlx.DB) *Handlers {
	return &Handlers{db: db}
}

func (h *Handlers) InitAuth(pass string) {
	password = pass
}

// RegisterHandlers регистрирует все обработчики API
func (h *Handlers) RegisterHandlers(r *mux.Router) {
	r.HandleFunc("/api/tasks", h.GetTasks).Methods("GET")
	r.HandleFunc("/api/task", h.GetTaskByID).Methods("GET")
	r.HandleFunc("/api/task", h.UpdateTask).Methods("PUT")
	r.HandleFunc("/api/task", h.DeleteTask).Methods("DELETE")
	r.HandleFunc("/api/nextdate", h.GetNextDate).Methods("GET")
	r.HandleFunc("/api/task", h.CreateTask).Methods("POST")
	r.HandleFunc("/api/task/done", h.MarkTaskAsDone).Methods("POST")

	r.HandleFunc("/api/signin", h.SignInHandler).Methods("POST")
	r.HandleFunc("/api/task", h.AuthMiddleware(h.CreateTask)).Methods("POST")
	r.HandleFunc("/api/task", h.AuthMiddleware(h.GetTaskByID)).Methods("GET")
	r.HandleFunc("/api/task", h.AuthMiddleware(h.UpdateTask)).Methods("PUT")
	r.HandleFunc("/api/task", h.AuthMiddleware(h.DeleteTask)).Methods("DELETE")
	r.HandleFunc("/api/tasks", h.AuthMiddleware(h.GetTasks)).Methods("GET")
	r.HandleFunc("/api/task/done", h.AuthMiddleware(h.MarkTaskAsDone)).Methods("POST")
}
