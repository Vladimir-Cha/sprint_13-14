package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Vladimir-Cha/sprint_13-14/project/project/tasks"
)

func sendError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	response := map[string]string{"error": message}
	json.NewEncoder(w).Encode(response)
}

func (h *Handlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close() // Закрываем тело запроса
	w.Header().Set("Content-Type", "application/json")

	// Получаем JSON из тела запроса
	var task tasks.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Недопустимый JSON")
		return
	}

	// Проверяем обязательное поле title
	if task.Title == "" {
		sendError(w, http.StatusBadRequest, "Не указан заголовок задачи")
		return
	}

	// Получаем текущую дату
	now := time.Now().Truncate(24 * time.Hour) // Убираем время

	// Обрабатываем поле date
	if task.Date == "today" {
		task.Date = now.Format(dateFormat)
	} else if task.Date == "" {
		task.Date = now.Format(dateFormat)
	} else {
		// Проверяем формат даты
		_, err := time.Parse(dateFormat, task.Date)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Неверный формат даты")
			return
		}

		// Если дата задачи меньше текущей, корректируем её
		taskDate, err := time.Parse(dateFormat, task.Date)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Ошибка парсинга даты задачи")
			return
		}

		if taskDate.Before(now) {
			if task.Repeat != "" {
				// Если правило повторения указано, вычисляем следующую дату
				nextDate, err := calculateNextDateFromRule(task.Date, task.Repeat, now)
				if err != nil {
					sendError(w, http.StatusInternalServerError, "Ошибка вычисления следующей даты")
					return
				}
				task.Date = nextDate
			} else {
				// Если правило повторения не указано, используем текущую дату
				task.Date = now.Format(dateFormat)
			}
		}
	}

	// Проверяем правило повторения
	if task.Repeat != "" {
		if !isValidRepeatRule(task.Repeat) {
			sendError(w, http.StatusBadRequest, "Неподдерживаемое правило повторения")
			return
		}
	}

	// Подключаемся к базе данных

	// Вставляем задачу в базу данных
	res, err := h.db.Exec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Ошибка базы данных")
		return
	}

	// Получаем ID созданной задачи
	id, err := res.LastInsertId()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Ошибка получения ID задачи")
		return
	}

	// Возвращаем JSON с ID задачи
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	response := map[string]string{"id": strconv.FormatInt(id, 10)} // Преобразуем ID в строку
	jsonResponse, _ := json.Marshal(response)
	w.Write(jsonResponse)
}

// DeleteTask удаляет задачу
func (h *Handlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Получаем ID задачи из параметра запроса
	id := r.URL.Query().Get("id")
	if id == "" {
		sendError(w, http.StatusBadRequest, "Не указан идентификатор задачи")
		return
	}

	// Преобразуем ID в int
	taskID, err := strconv.Atoi(id)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Неверный формат идентификатора")
		return
	}

	// Подключаемся к базе данных

	// Удаляем задачу
	_, err = h.db.Exec("DELETE FROM scheduler WHERE id = ?", taskID)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Ошибка при удалении задачи")
		return
	}

	// Возвращаем пустой JSON
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// GetTasks возвращает список всех задач
func (h *Handlers) GetTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Получаем параметр поиска из запроса
	search := r.URL.Query().Get("search")

	// Подключаемся к базе данных

	var tasksList []tasks.Task
	var err error

	// Проверяем, является ли поисковой запрос датой в формате 02.01.2006
	if search != "" {
		if date, err := time.Parse("02.01.2006", search); err == nil {
			// Если это дата, ищем задачи на эту дату
			formattedDate := date.Format(dateFormat)
			err = h.db.Select(&tasksList, "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? ORDER BY date LIMIT 50", formattedDate)
		} else {
			// Если запрос не является датой, то ищем задачи по заголовку или комментарию
			searchPattern := "%" + search + "%"
			err = h.db.Select(&tasksList, "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT 50", searchPattern, searchPattern)
		}
	} else {
		// Если поисковой запрос не указан, возвращаем все задачи
		err = h.db.Select(&tasksList, "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT 50")
	}

	// Обрабатываем ошибки
	if err != nil {
		sendError(w, http.StatusBadRequest, "Ошибка базы данных")
		return
	}

	// Если tasksList равен nil, создаем пустой слайс
	if tasksList == nil {
		tasksList = []tasks.Task{}
	}

	// Формируем JSON-ответ
	response := map[string][]tasks.Task{"tasks": tasksList}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Ошибка при формировании JSON")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// UpdateTask обновляет существующую задачу
func (h *Handlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Set("Content-Type", "application/json")

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Ошибка чтения тела запроса")
		return
	}

	// Парсим JSON
	var task tasks.Task
	err = json.Unmarshal(body, &task)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Недопустимый JSON")
		return
	}

	// Проверяем, что ID задачи указан
	if task.ID == "" {
		sendError(w, http.StatusBadRequest, "Не указан идентификатор задачи")
		return
	}

	// Проверяем, что ID задачи является числом
	taskID, err := strconv.Atoi(task.ID)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Неверный формат идентификатора задачи")
		return
	}

	// Проверяем обязательное поле title
	if task.Title == "" {
		sendError(w, http.StatusBadRequest, "Не указан заголовок задачи")
		return
	}

	// Проверяем формат даты
	if task.Date != "" {
		_, err := time.Parse(dateFormat, task.Date)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Неверный формат даты")
			return
		}
	}

	// Проверяем правило повторения
	if task.Repeat != "" && !isValidRepeatRule(task.Repeat) {
		sendError(w, http.StatusBadRequest, "Неподдерживаемое правило повторения")
		return
	}

	// Подключаемся к базе данных

	// Проверяем, существует ли задача с таким ID
	var existingTask tasks.Task
	err = h.db.Get(&existingTask, "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", taskID)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Задача не найдена")
		return
	}

	// Обновляем задачу в базе данных
	_, err = h.db.Exec("UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?",
		task.Date, task.Title, task.Comment, task.Repeat, taskID)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Ошибка базы данных")
		return
	}

	// Возвращаем пустой JSON в случае успеха
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// GetTaskByID получает задачу по id
func (h *Handlers) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Получаем ID задачи из параметра запроса
	id := r.URL.Query().Get("id")
	if id == "" {
		sendError(w, http.StatusBadRequest, "Не указан идентификатор")
		return
	}

	// Преобразуем ID в int
	taskID, err := strconv.Atoi(id)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Неверный формат идентификатора")
		return
	}

	// Ищем задачу по ID
	var task tasks.Task
	err = h.db.Get(&task, "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", taskID)
	if err != nil {
		sendError(w, http.StatusNotFound, "Задача не найдена")
		return
	}

	// Возвращаем задачу в формате JSON
	jsonResponse, err := json.Marshal(task)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Ошибка при формировании JSON")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// MarkTaskAsDone отмечает задачу как выполненную
func (h *Handlers) MarkTaskAsDone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Ищем задачу по ID
	id := r.URL.Query().Get("id")
	if id == "" {
		sendError(w, http.StatusBadRequest, "Не указан идентификатор задачи")
		return
	}
	// Преобразуем ID в int
	taskID, err := strconv.Atoi(id)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Неверный формат идентификатора")
		return
	}
	// Подключаемся к базе данных

	// Возвращаем ответ в формате JSON
	var task tasks.Task
	err = h.db.Get(&task, "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", taskID)
	if err != nil {
		sendError(w, http.StatusNotFound, "Задача не найдена")
		return
	}

	if task.Repeat == "" {
		_, err = h.db.Exec("DELETE FROM scheduler WHERE id = ?", taskID)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Ошибка при удалении задачи")
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
		return
	}

	// Вычисляем следующую дату для повторяющейся задачи
	now := time.Now()
	nextDate, err := h.NextDate(taskID, now)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]string{"error": "Ошибка при вычислении следующей даты"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}
	// Обновляем дату задачи в базе данных
	_, err = h.db.Exec("UPDATE scheduler SET date = ? WHERE id = ?", nextDate, taskID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Ошибка при обновлении задачи")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
