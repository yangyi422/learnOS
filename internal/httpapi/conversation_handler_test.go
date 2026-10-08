package httpapi

import (
	"net/http"
	"strconv"
	"testing"

	"learnos/internal/model"
)

func TestConversationAPIRestoreRetryAndSkip(t *testing.T) {
	app := newTestLearningApp(t)
	prefix := "/api/v1/courses/" + strconv.Itoa(int(app.course.ID)) + "/lessons/" + strconv.Itoa(int(app.lesson.ID))
	response := requestJSON(t, app.router, http.MethodPost, prefix+"/conversation", `{"message":"请换一种解释","idempotency_key":"api-chat"}`)
	if response.Code != http.StatusOK {
		t.Fatal(response.Code, response.Body.String())
	}
	var turns []model.LearningTurn
	app.db.Where("lesson_id = ?", app.lesson.ID).Find(&turns)
	if len(turns) != 1 {
		t.Fatal("message not saved")
	}
	retry := requestJSON(t, app.router, http.MethodPost, prefix+"/conversation/"+strconv.Itoa(int(turns[0].ID))+"/retry", "")
	if retry.Code != http.StatusOK {
		t.Fatal(retry.Code, retry.Body.String())
	}
	restored := requestJSON(t, app.router, http.MethodGet, prefix+"/conversation", "")
	if restored.Code != http.StatusOK {
		t.Fatal(restored.Code)
	}
	complete := requestJSON(t, app.router, http.MethodPost, prefix+"/advance", `{"action":"complete"}`)
	if complete.Code == http.StatusOK {
		t.Fatal("mock fabricated completion")
	}
	skip := requestJSON(t, app.router, http.MethodPost, prefix+"/advance", `{"action":"skip"}`)
	if skip.Code != http.StatusOK {
		t.Fatal(skip.Code, skip.Body.String())
	}
	replay := requestJSON(t, app.router, http.MethodPost, prefix+"/advance", `{"action":"skip"}`)
	if replay.Code != http.StatusOK {
		t.Fatal(replay.Code)
	}
	var lesson model.Lesson
	app.db.First(&lesson, app.lesson.ID)
	if lesson.Status != model.LessonStatusSkipped {
		t.Fatal(lesson.Status)
	}
}
