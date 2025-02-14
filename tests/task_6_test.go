package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTask(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	now := time.Now()

	task := task{
		date:    now.Format(`20060102`),
		title:   "Созвон в 16:00",
		comment: "Обсуждение планов",
		repeat:  "d 5",
	}

	todo := addTask(t, task)

	body, err := requestJSON("api/task", nil, http.MethodGet)
	assert.NoError(t, err)
	var m map[string]interface{} //Исправлено string на interface{}, так как id имеет числовой тип.
	err = json.Unmarshal(body, &m)
	assert.NoError(t, err)

	e, ok := m["error"]
	assert.False(t, !ok || len(fmt.Sprint(e)) == 0,
		"Ожидается ошибка для вызова /api/task")

	body, err = requestJSON("api/task?id="+todo, nil, http.MethodGet)
	assert.NoError(t, err)
	err = json.Unmarshal(body, &m)
	assert.NoError(t, err)

	//Тест ожидает следующую дату (судя по полученному телу ответа "body"), но сравнивает с текущей датой.
	//Изменение должно получать дату из структуры task, добавлять из неё нужное количество дней d и сравнивать новую дату с полученной.
	daysAdd, err := strconv.Atoi(strings.TrimPrefix(task.repeat, "d "))
	assert.NoError(t, err)
	newDate := now.AddDate(0, 0, daysAdd).Format("20060102")

	assert.Equal(t, todo, fmt.Sprint(int(m["id"].(float64)))) //Добавлено преобразование ID из float64 в int.
	//assert.Equal(t, todo, m["id"])
	assert.Equal(t, newDate, m["date"])
	assert.Equal(t, task.title, m["title"])
	assert.Equal(t, task.comment, m["comment"])
	assert.Equal(t, task.repeat, m["repeat"])
	//t.Logf("Тело ответа: %s", string(body)) //Лог для тела ответа body
}

type fulltask struct {
	id string
	task
}

func TestEditTask(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	now := time.Now()

	tsk := task{
		date:    now.Format(`20060102`),
		title:   "Заказать пиццу",
		comment: "в 17:00",
		repeat:  "",
	}

	id := addTask(t, tsk)

	tbl := []fulltask{
		{"", task{"20240129", "Тест", "", ""}},
		{"abc", task{"20240129", "Тест", "", ""}},
		{"7645346343", task{"20240129", "Тест", "", ""}},
		{id, task{"20240129", "", "", ""}},
		{id, task{"20240192", "Qwerty", "", ""}},
		{id, task{"28.01.2024", "Заголовок", "", ""}},
		{id, task{"20240212", "Заголовок", "", "ooops"}},
	}
	for _, v := range tbl {
		m, err := postJSON("api/task", map[string]any{
			"id":      v.id,
			"date":    v.date,
			"title":   v.title,
			"comment": v.comment,
			"repeat":  v.repeat,
		}, http.MethodPut)
		assert.NoError(t, err)

		var errVal string
		e, ok := m["error"]
		if ok {
			errVal = fmt.Sprint(e)
		}
		assert.NotEqual(t, len(errVal), 0, "Ожидается ошибка для значения %v", v)
	}

	updateTask := func(newVals map[string]any) {
		//Преобразование id из string в int.
		if idStr, ok := newVals["id"].(string); ok {
			idInt, err := strconv.Atoi(idStr)
			if err != nil {
				t.Errorf("Ошибка преобразования id в число: %v", err)
				return
			}
			newVals["id"] = idInt
		}
		//t.Logf("Отправляемые данные: %+v", newVals) //Лог для отправляемых данных
		mupd, err := postJSON("api/task", newVals, http.MethodPut)
		assert.NoError(t, err)
		//t.Logf("Ответ сервера: %+v", mupd) //Лог для полученных данных
		e, ok := mupd["error"]
		assert.False(t, ok && fmt.Sprint(e) != "")

		var task Task
		err = db.Get(&task, `SELECT * FROM scheduler WHERE id=?`, id)
		assert.NoError(t, err)

		assert.Equal(t, id, strconv.FormatInt(task.ID, 10))
		assert.Equal(t, newVals["title"], task.Title)
		if _, is := newVals["comment"]; !is {
			newVals["comment"] = ""
		}
		if _, is := newVals["repeat"]; !is {
			newVals["repeat"] = ""
		}
		assert.Equal(t, newVals["comment"], task.Comment)
		assert.Equal(t, newVals["repeat"], task.Repeat)
		now := time.Now().Format(`20060102`)
		if task.Date < now {
			t.Errorf("Дата не может быть меньше сегодняшней")
		}
	}

	updateTask(map[string]any{
		"id":      id,
		"date":    now.Format(`20060102`),
		"title":   "Заказать хинкали",
		"comment": "в 18:00",
		"repeat":  "d 7",
	})
}
