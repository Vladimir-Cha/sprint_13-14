package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"project/dbopen"
	"project/tasks"
	"strconv"
	"strings"
	"time"
)

// GetNextDate возвращает следующую дату выполнения задачи
func GetNextDate(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры из запроса
	date := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")
	nowStr := r.URL.Query().Get("now")

	// Парсим текущую дату
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "Invalid 'now' parameter"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Вычисляем следующую дату
	nextDate, err := calculateNextDateFromRule(date, repeat, now)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": err.Error()}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	// Возвращаем дату в виде строки
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}

// NextDate вычисляет следующую дату для задачи
func NextDate(taskID int, now time.Time) (string, error) {
	db := dbopen.DB()
	defer db.Close()

	// Чтение текущей задачи из базы данных
	var task tasks.Task
	err := db.Get(&task, `SELECT * FROM scheduler WHERE id=?`, taskID)
	if err != nil {
		return "", err
	}

	// Если поле повторения пустое, удаляем задачу из базы данных
	if task.Repeat == "" {
		_, err = db.Exec("DELETE FROM scheduler WHERE id=?", taskID)
		if err != nil {
			return "", err
		}
		return "Задача удалена", nil
	}

	// Вычисляем следующую дату в зависимости от правила повторения
	nextDate, err := calculateNextDateFromRule(task.Date, task.Repeat, now)
	if err != nil {
		return "", err
	}

	// Обновляем задачу в базе данных с новой датой
	_, err = db.Exec("UPDATE scheduler SET date=? WHERE id=?", nextDate, taskID)
	if err != nil {
		return "", err
	}

	return nextDate, nil
}

// calculateNextDateFromRule обрабатывает правило вычисления следующей даты
func calculateNextDateFromRule(date, repeat string, now time.Time) (string, error) {
	taskDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", fmt.Errorf("invalid date format")
	}

	switch {
	case strings.HasPrefix(repeat, "d "):
		days, err := strconv.Atoi(repeat[2:])
		if err != nil {
			return "", fmt.Errorf("invalid days value")
		}
		if days > 400 {
			return "", fmt.Errorf("days value exceeds maximum")
		}
		nextDate := taskDate.AddDate(0, 0, days)
		fmt.Printf("Task date: %s, Days to add: %d, Next date: %s\n", taskDate.Format("20060102"), days, nextDate.Format("20060102"))
		return nextDate.Format("20060102"), nil

	case repeat == "y":
		nextDate := taskDate.AddDate(1, 0, 0)
		return nextDate.Format("20060102"), nil

	case strings.HasPrefix(repeat, "w "):
		daysOfWeek := strings.Split(repeat[2:], ",")
		return calculateNextWeekDate(taskDate, daysOfWeek)

	case strings.HasPrefix(repeat, "m "):
		parts := strings.Split(repeat[2:], " ")
		daysOfMonth := strings.Split(parts[0], ",")
		var months []string
		if len(parts) > 1 {
			months = strings.Split(parts[1], ",")
		}
		return calculateNextMonthDate(taskDate, daysOfMonth, months)

	default:
		return "", fmt.Errorf("unsupported repeat rule")
	}
}

// calculateNextWeekDate вычисляет следующую дату по дням недели
func calculateNextWeekDate(taskDate time.Time, daysOfWeek []string) (string, error) {
	for _, dayStr := range daysOfWeek {
		day, err := strconv.Atoi(dayStr)
		if err != nil || day < 1 || day > 7 {
			return "", fmt.Errorf("invalid day of week")
		}

		currentWeekday := int(taskDate.Weekday())
		if currentWeekday == 0 {
			currentWeekday = 7
		}
		diff := day - currentWeekday
		if diff <= 0 {
			diff += 7
		}

		nextDate := taskDate.AddDate(0, 0, diff)
		return nextDate.Format("20060102"), nil
	}
	return "", fmt.Errorf("no valid day of week")
}

// calculateNextMonthDate вычисляет следующую дату по дням месяца
func calculateNextMonthDate(taskDate time.Time, daysOfMonth []string, months []string) (string, error) {
	for {
		taskDate = taskDate.AddDate(0, 0, 1)
		day := taskDate.Day()
		month := int(taskDate.Month())

		dayMatch := false
		for _, dayStr := range daysOfMonth {
			dayInt, err := strconv.Atoi(dayStr)
			if err != nil {
				continue
			}
			if dayInt == day || (dayStr == "-1" && day == lastDayOfMonth(taskDate)) || (dayStr == "-2" && day == lastDayOfMonth(taskDate)-1) {
				dayMatch = true
				break
			}
		}

		monthMatch := true
		if len(months) > 0 {
			monthMatch = false
			for _, monthStr := range months {
				monthInt, err := strconv.Atoi(monthStr)
				if err != nil {
					continue
				}
				if monthInt == month {
					monthMatch = true
					break
				}
			}
		}

		if dayMatch && monthMatch {
			return taskDate.Format("20060102"), nil
		}
	}
}

// lastDayOfMonth возвращает последний день месяца
func lastDayOfMonth(date time.Time) int {
	return time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
