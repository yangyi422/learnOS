package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"learnos/internal/model"
)

func TestGetCourseCognitiveStatesReturnsEveryLessonAsUnseen(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/1/cognitive-states", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			CourseID uint `json:"course_id"`
			States   []struct {
				LessonID     uint   `json:"lesson_id"`
				CurrentLevel string `json:"current_level"`
				Status       string `json:"status"`
				Evidence     int    `json:"evidence_count"`
			} `json:"states"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode cognitive states: %v", err)
	}
	if payload.Data.CourseID != app.course.ID || len(payload.Data.States) != 8 {
		t.Fatalf("unexpected cognitive state list: %+v", payload.Data)
	}
	for _, state := range payload.Data.States {
		if state.CurrentLevel != model.CognitiveLevelUnseen || state.Status != model.CognitiveStatusUnknown || state.Evidence != 0 {
			t.Fatalf("expected unseen state, got %+v", state)
		}
	}
}

func TestCognitiveStateAPIsExposeAnswerStateAndEvidence(t *testing.T) {
	app := newTestLearningApp(t)
	answerResponse := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerBody(app.lesson.ID, "口渴不是唯一的补水依据。"))
	if answerResponse.Code != http.StatusOK {
		t.Fatalf("submit answer: expected 200, got %d: %s", answerResponse.Code, answerResponse.Body.String())
	}
	var answerPayload struct {
		Data struct {
			DemonstratedLevel        string     `json:"demonstrated_level"`
			UserUnderstandingSummary string     `json:"user_understanding_summary"`
			CognitiveEvidence        []struct{} `json:"cognitive_evidence"`
			CognitiveState           struct {
				CurrentLevel string `json:"current_level"`
				Status       string `json:"status"`
			} `json:"cognitive_state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(answerResponse.Body.Bytes(), &answerPayload); err != nil {
		t.Fatalf("decode answer: %v", err)
	}
	if answerPayload.Data.DemonstratedLevel != model.CognitiveLevelUnderstand || answerPayload.Data.UserUnderstandingSummary == "" || len(answerPayload.Data.CognitiveEvidence) != 2 || answerPayload.Data.CognitiveState.CurrentLevel != model.CognitiveLevelUnderstand || answerPayload.Data.CognitiveState.Status != model.CognitiveStatusStable {
		t.Fatalf("unexpected answer cognitive data: %+v", answerPayload.Data)
	}
	if strings.Contains(answerResponse.Body.String(), "raw_response") {
		t.Fatalf("raw AI response leaked: %s", answerResponse.Body.String())
	}

	detailResponse := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/1/lessons/"+strconv.FormatUint(uint64(app.lesson.ID), 10)+"/cognitive-state", "")
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("get cognitive detail: expected 200, got %d: %s", detailResponse.Code, detailResponse.Body.String())
	}
	var detailPayload struct {
		Data struct {
			State struct {
				CurrentLevel string `json:"current_level"`
				Status       string `json:"status"`
			} `json:"state"`
			Evidence []struct{} `json:"evidence"`
			Timeline []struct{} `json:"timeline"`
		} `json:"data"`
	}
	if err := json.Unmarshal(detailResponse.Body.Bytes(), &detailPayload); err != nil {
		t.Fatalf("decode cognitive detail: %v", err)
	}
	if detailPayload.Data.State.CurrentLevel != model.CognitiveLevelUnderstand || detailPayload.Data.State.Status != model.CognitiveStatusStable || len(detailPayload.Data.Evidence) != 2 || len(detailPayload.Data.Timeline) != 1 {
		t.Fatalf("unexpected cognitive detail: %+v", detailPayload.Data)
	}
}

func TestGetCognitiveStateRejectsLessonFromAnotherCourse(t *testing.T) {
	app := newTestLearningApp(t)
	otherLesson := model.Lesson{CourseID: app.course.ID + 1, UnitID: 999, Title: "另一门课程知识", CoreQuestion: "问题"}
	if err := app.db.Create(&otherLesson).Error; err != nil {
		t.Fatalf("create other lesson: %v", err)
	}
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/1/lessons/"+strconv.FormatUint(uint64(otherLesson.ID), 10)+"/cognitive-state", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-course lesson, got %d: %s", response.Code, response.Body.String())
	}
}
