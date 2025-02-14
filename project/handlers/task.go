package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"project/dbopen"
	"project/tasks"
	"strconv"
	"time"
)

// CreateTask создает новую задачу
func CreateTask(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		response := map[string]string{"error": "Недопустимый метод запроса"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Получаем JSON из тела запроса
	var task tasks.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Недопустимый JSON"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Проверяем обязательное поле title
	if task.Title == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Не указан заголовок задачи"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Получаем текущую дату
	now := time.Now()

	// Обрабатываем поле date
	if task.Date == "today" {
		task.Date = now.Format("20060102") // Заменяем "today" на текущую дату
	} else if task.Date == "" {
		// Если дата не указана, используем текущую дату
		task.Date = now.Format("20060102")
	} else {
		// Проверяем формат даты
		taskDate, err := time.Parse("20060102", task.Date)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			response := map[string]string{"error": "Неверный формат даты"}
			jsonResponse, _ := json.Marshal(response)
			w.Write(jsonResponse)
			return
		}

		// Если дата задачи меньше текущей, корректируем её
		if taskDate.Before(now) {
			if task.Repeat != "" {
				// Если правило повторения указано, вычисляем следующую дату
				nextDate, err := calculateNextDateFromRule(task.Date, task.Repeat, now)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					response := map[string]string{"error": "Ошибка вычисления следующей даты"}
					jsonResponse, _ := json.Marshal(response)
					w.Write(jsonResponse)
					return
				}
				task.Date = nextDate
			} else {
				// Если правило повторения не указано, используем текущую дату
				task.Date = now.Format("20060102")
			}
		}
	}

	// Проверяем правило повторения
	if task.Repeat != "" {
		if !isValidRepeatRule(task.Repeat) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			response := map[string]string{"error": "Неподдерживаемое правило повторения"}
			jsonResponse, _ := json.Marshal(response)
			w.Write(jsonResponse)
			return
		}
	}

	// Подключаемся к базе данных
	db := dbopen.DB()
	defer db.Close()

	// Вставляем задачу в базу данных
	res, err := db.Exec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]string{"error": "Ошибка базы данных"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Получаем ID созданной задачи
	id, err := res.LastInsertId()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]string{"error": "Ошибка получения ID задачи"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Возвращаем JSON с ID задачи
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	response := map[string]int64{"id": id}
	jsonResponse, _ := json.Marshal(response)
	w.Write(jsonResponse)
}

// DeleteTask удаляет задачу
func DeleteTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Получаем ID задачи из параметра запроса
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Не указан идентификатор задачи"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Преобразуем ID в int
	taskID, err := strconv.Atoi(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Неверный формат идентификатора"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Подключаемся к базе данных
	db := dbopen.DB()
	defer db.Close()

	// Удаляем задачу
	_, err = db.Exec("DELETE FROM scheduler WHERE id = ?", taskID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]string{"error": "Ошибка при удалении задачи"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Возвращаем пустой JSON
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// GetTasks возвращает список всех задач
func GetTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Получаем параметр поиска из запроса
	search := r.URL.Query().Get("search")

	// Подключаемся к базе данных
	db := dbopen.DB()
	defer db.Close()

	var tasksList []tasks.Task
	var err error

	// Проверяем, является ли поисковой запрос датой в формате 02.01.2006
	if search != "" {
		if date, err := time.Parse("02.01.2006", search); err == nil {
			// Если это дата, ищем задачи на эту дату
			formattedDate := date.Format("20060102")
			err = db.Select(&tasksList, "SELECT * FROM scheduler WHERE date = ? ORDER BY date LIMIT 50", formattedDate)
		} else {
			// Если запрос не является датой, то ищем задачи по заголовку или комментарию
			searchPattern := "%" + search + "%"
			err = db.Select(&tasksList, "SELECT * FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT 50", searchPattern, searchPattern)
		}
	} else {
		// Если поисковой запрос не указан, возвращаем все задачи
		err = db.Select(&tasksList, "SELECT * FROM scheduler ORDER BY date LIMIT 50")
	}

	// Обрабатываем ошибки
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]string{"error": "Ошибка базы данных"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
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
		w.WriteHeader(http.StatusInternalServerError)
		errorResponse := map[string]string{"error": "Ошибка при формировании JSON"}
		jsonErrorResponse, _ := json.Marshal(errorResponse)
		w.Write(jsonErrorResponse)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// UpdateTask обновляет существующую задачу
func UpdateTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Ошибка чтения тела запроса"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}
	log.Printf("Полученное тело запроса: %s", string(body))

	// Получаем JSON из тела запроса
	var task tasks.Task
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&task)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Недопустимый JSON"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Проверяем, что ID задачи указан
	if task.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Не указан идентификатор задачи"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Проверяем обязательное поле title
	if task.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Не указан заголовок задачи"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Проверяем формат даты
	if task.Date != "" {
		_, err := time.Parse("20060102", task.Date)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := map[string]string{"error": "Неверный формат даты"}
			jsonResponse, _ := json.Marshal(response)
			w.Write(jsonResponse)
			return
		}
	}

	// Подключаемся к базе данных
	db := dbopen.DB()
	defer db.Close()

	// Обновляем задачу в базе данных
	_, err = db.Exec("UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?",
		task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]string{"error": "Ошибка базы данных"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Возвращаем пустой JSON в случае успеха
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// GetTaskByID получает задачу по id
func GetTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Получаем ID задачи из параметра запроса
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Не указан идентификатор"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Преобразуем ID в int
	taskID, err := strconv.Atoi(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Неверный формат идентификатора"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Подключаемся к базе данных
	db := dbopen.DB()
	defer db.Close()

	// Ищем задачу по ID
	var task tasks.Task
	err = db.Get(&task, "SELECT * FROM scheduler WHERE id = ?", taskID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		response := map[string]string{"error": "Задача не найдена"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Возвращаем задачу в формате JSON
	jsonResponse, err := json.Marshal(task)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]string{"error": "Ошибка при формировании JSON"}
		jsonErrorResponse, _ := json.Marshal(response)
		w.Write(jsonErrorResponse)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// MarkTaskAsDone отмечает задачу как выполненную
func MarkTaskAsDone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Ищем задачу по ID
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Не указан идентификатор задачи"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}
	// Преобразуем ID в int
	taskID, err := strconv.Atoi(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Неверный формат идентификатора"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}
	// Подключаемся к базе данных
	db := dbopen.DB()
	defer db.Close()

	// Возвращаем ответ в формате JSON
	var task tasks.Task
	err = db.Get(&task, "SELECT * FROM scheduler WHERE id = ?", taskID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		response := map[string]string{"error": "Задача не найдена"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	if task.Repeat == "" {
		_, err = db.Exec("DELETE FROM scheduler WHERE id = ?", taskID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := map[string]string{"error": "Ошибка при удалении задачи"}
			jsonResponse, _ := json.Marshal(response)
			w.Write(jsonResponse)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
		return
	}

	// Вычисляем следующую дату для повторяющейся задачи
	now := time.Now()
	nextDate, err := NextDate(taskID, now)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]string{"error": "Ошибка при вычислении следующей даты"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}
	// Обновляем дату задачи в базе данных
	_, err = db.Exec("UPDATE scheduler SET date = ? WHERE id = ?", nextDate, taskID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]string{"error": "Ошибка при обновлении задачи"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
