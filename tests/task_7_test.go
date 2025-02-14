package tests

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func notFoundTask(t *testing.T, id int) {
	body, err := requestJSON("api/task?id="+strconv.Itoa(id), nil, http.MethodGet) //
	assert.NoError(t, err)
	t.Logf("Отправляемые данные: %+v", id)
	var m map[string]any
	err = json.Unmarshal(body, &m)
	assert.NoError(t, err)
	_, ok := m["error"]
	assert.True(t, ok)
}

func TestDone(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	now := time.Now()
	idInt := addTask(t, task{
		date:  now.Format(`20060102`),
		title: "Свести баланс",
	})

	//Преобразуем id из string в int
	id, err := strconv.Atoi(idInt)
	assert.NoError(t, err)

	ret, err := postJSON("api/task/done?id="+strconv.Itoa(id), nil, http.MethodPost)
	assert.NoError(t, err)
	assert.Empty(t, ret)
	notFoundTask(t, id)

	idInt = addTask(t, task{
		title:  "Проверить работу /api/task/done",
		repeat: "d 3",
	})

	//Преобразуем id из string в int
	id, err = strconv.Atoi(idInt)
	assert.NoError(t, err)

	for i := 0; i < 3; i++ {
		ret, err := postJSON("api/task/done?id="+strconv.Itoa(id), nil, http.MethodPost)
		assert.NoError(t, err)
		assert.Empty(t, ret)

		var task Task
		err = db.Get(&task, `SELECT * FROM scheduler WHERE id=?`, id)
		assert.NoError(t, err)
		now = now.AddDate(0, 0, 3)
		assert.Equal(t, task.Date, now.Format(`20060102`))
	}
}

func TestDelTask(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	idInt := addTask(t, task{
		title:  "Временная задача",
		repeat: "d 3",
	})

	//Преобразуем id из string в int
	id, err := strconv.Atoi(idInt)
	assert.NoError(t, err)

	ret, err := postJSON("api/task?id="+strconv.Itoa(id), nil, http.MethodDelete)
	assert.NoError(t, err)
	assert.Empty(t, ret)

	notFoundTask(t, id)

	ret, err = postJSON("api/task", nil, http.MethodDelete)
	assert.NoError(t, err)
	assert.NotEmpty(t, ret)
	ret, err = postJSON("api/task?id=wjhgese", nil, http.MethodDelete)
	assert.NoError(t, err)
	assert.NotEmpty(t, ret)
}
