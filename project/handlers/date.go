package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Vladimir-Cha/sprint_13-14/project/project/dbopen"
	"github.com/Vladimir-Cha/sprint_13-14/project/project/tasks"

	"github.com/jmoiron/sqlx"
)

const dateFormat = "20060102"

var db *sqlx.DB

func InitDB() {
	db = dbopen.DB()
}

// GetNextDate возвращает следующую дату выполнения задачи
func GetNextDate(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры из запроса
	date := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")
	nowStr := r.URL.Query().Get("now")

	// Парсим текущую дату
	now, err := time.Parse(dateFormat, nowStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]string{"error": "неверный 'now' параметр"}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
		return
	}

	//fmt.Printf("GetNextDate: date=%s, repeat=%s, now=%s\n", date, repeat, now.Format(dateFormat))

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

	// Чтение текущей задачи из базы данных
	var task tasks.Task
	err := db.Get(&task, `SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?`, taskID)
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
	taskDate, err := time.Parse(dateFormat, date)
	if err != nil {
		return "", fmt.Errorf("неверный формат даты")
	}

	switch {
	case strings.HasPrefix(repeat, "d "): // Повторение через N дней
		days, err := strconv.Atoi(repeat[2:])
		if err != nil {
			return "", fmt.Errorf("недопустимое значение дней")
		}
		if days < 1 || days > 400 {
			return "", fmt.Errorf("количество дней должно быть от 1 до 400")
		}

		// Вычисляем следующую дату
		nextDate := taskDate
		for {
			nextDate = nextDate.AddDate(0, 0, days)
			if nextDate.After(now) {
				break
			}
		}
		return nextDate.Format(dateFormat), nil

	case repeat == "y": // Ежегодное повторение
		nextDate := taskDate
		for {
			nextDate = nextDate.AddDate(1, 0, 0)
			// Если дата - 29 февраля, а следующий год не високосный, переносим на 1 марта
			if nextDate.Month() == time.February && nextDate.Day() == 29 && IsLeapYear(nextDate.Year()) {
				nextDate = nextDate.AddDate(0, 0, 1) // Переносим на 1 марта
			}
			if nextDate.After(now) {
				break
			}
		}
		return nextDate.Format(dateFormat), nil

	case strings.HasPrefix(repeat, "w "): // Повторение по дням недели
		daysOfWeek := strings.Split(repeat[2:], ",")
		return calculateNextWeekDate(taskDate, daysOfWeek, now)

	case strings.HasPrefix(repeat, "m "): // Повторение по дням месяца
		parts := strings.Split(repeat[2:], " ")
		daysOfMonth := strings.Split(parts[0], ",")
		var months []string
		if len(parts) > 1 {
			months = strings.Split(parts[1], ",")
		}
		return calculateNextMonthDate(taskDate, daysOfMonth, months, now)

	default:
		return "", fmt.Errorf("некорректное правило повторения")
	}
}

// Проверка на високосный год
func IsLeapYear(year int) bool {
	return (year%4 == 0 && year%100 != 0) || year%400 == 0
}

// calculateNextWeekDate вычисляет следующую дату по дням недели
func calculateNextWeekDate(taskDate time.Time, daysOfWeek []string, now time.Time) (string, error) {
	for {
		taskDate = taskDate.AddDate(0, 0, 1) // Переходим к следующему дню
		currentWeekday := int(taskDate.Weekday())
		if currentWeekday == 0 {
			currentWeekday = 7 // Воскресенье
		}

		for _, dayStr := range daysOfWeek {
			day, err := strconv.Atoi(dayStr)
			if err != nil || day < 1 || day > 7 {
				return "", fmt.Errorf("неверный день недели")
			}

			if currentWeekday == day && (taskDate.After(now) || taskDate.Equal(now)) {
				return taskDate.Format(dateFormat), nil
			}
		}
	}
}

// calculateNextMonthDate вычисляет следующую дату по дням месяца
func calculateNextMonthDate(taskDate time.Time, daysOfMonth []string, months []string, now time.Time) (string, error) {
	for {
		taskDate = taskDate.AddDate(0, 0, 1)
		day := taskDate.Day()
		month := int(taskDate.Month())

		// Проверяем, подходит ли день месяца
		dayMatch := false
		for _, dayStr := range daysOfMonth {
			if dayStr == "-1" {
				if day == lastDayOfMonth(taskDate) {
					dayMatch = true
					break
				}
			} else if dayStr == "-2" {
				if day == lastDayOfMonth(taskDate)-1 {
					dayMatch = true
					break
				}
			} else {
				dayInt, err := strconv.Atoi(dayStr)
				if err == nil && dayInt == day {
					dayMatch = true
					break
				}
			}
		}

		// Проверяем, подходит ли месяц
		monthMatch := true
		if len(months) > 0 {
			monthMatch = false
			for _, monthStr := range months {
				monthInt, err := strconv.Atoi(monthStr)
				if err == nil && monthInt == month {
					monthMatch = true
					break
				}
			}
		}

		// Если день и месяц подходят, и дата после текущей, возвращаем её
		if dayMatch && monthMatch && (taskDate.After(now) || taskDate.Equal(now)) {
			return taskDate.Format(dateFormat), nil
		}
	}
}

// lastDayOfMonth возвращает последний день месяца
func lastDayOfMonth(date time.Time) int {
	return time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
