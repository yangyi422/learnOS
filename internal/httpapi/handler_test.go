package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"learnos/internal/ai"
	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testLearningApp struct {
	db      *gorm.DB
	router  *gin.Engine
	service *service.CourseService
	course  model.Course
	lesson  model.Lesson
}

func newTestLearningApp(t *testing.T) testLearningApp {
	return newTestLearningAppWithProvider(t, nil)
}

func newTestLearningAppWithProvider(t *testing.T, provider ai.AIProvider) testLearningApp {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Course{}, &model.CourseUnit{}, &model.Lesson{}, &model.LessonRelation{}, &model.LearningTurn{},
		&model.MasteryRecord{}, &model.Misconception{}, &model.AIEvaluationRun{}, &model.CognitiveState{}, &model.CognitiveEvidence{}, &model.CognitiveStateEvent{},
		&model.AssessmentChallenge{}, &model.ChallengeAttempt{}, &model.MisconceptionEvent{}, &model.MisconceptionPatternLink{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	courseRepository := repository.NewCourseRepository(db)
	learningRepository := repository.NewLearningRepository(db)
	courseService := service.NewCourseService(courseRepository, learningRepository, provider)
	knowledgeGraphRepository := repository.NewKnowledgeGraphRepository(db)
	knowledgeGraphService := service.NewKnowledgeGraphService(courseRepository, knowledgeGraphRepository)
	cognitiveRepository := repository.NewCognitiveRepository(db)
	cognitiveStateService := service.NewCognitiveStateService(courseRepository, knowledgeGraphRepository, cognitiveRepository)
	courseService.SetCognitiveStateService(cognitiveStateService)
	misconceptionRepository := repository.NewMisconceptionRepository(db)
	misconceptionService := service.NewMisconceptionService(courseRepository, knowledgeGraphRepository, misconceptionRepository)
	challengeRepository := repository.NewChallengeRepository(db)
	challengeProvider, ok := provider.(ai.ChallengeProvider)
	if !ok && provider == nil {
		challengeProvider = ai.NewMockProvider()
	}
	challengeService := service.NewChallengeService(courseRepository, learningRepository, knowledgeGraphRepository, challengeRepository, misconceptionRepository, cognitiveStateService, challengeProvider)
	if err := courseService.SeedStarterCourse(context.Background()); err != nil {
		t.Fatalf("seed test course: %v", err)
	}
	var course model.Course
	if err := db.Where("name = ?", "营养学").First(&course).Error; err != nil {
		t.Fatalf("find test course: %v", err)
	}
	var lesson model.Lesson
	if err := db.First(&lesson, *course.CurrentLessonID).Error; err != nil {
		t.Fatalf("find test lesson: %v", err)
	}

	handler := NewHandler(courseService, knowledgeGraphService, cognitiveStateService, challengeService, misconceptionService)
	webFS := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}
	router := NewRouter(config.Config{}, handler, webFS)
	return testLearningApp{db: db, router: router, service: courseService, course: course, lesson: lesson}
}

func TestChallengeAPIAnswerIsSingleUse(t *testing.T) {
	app := newTestLearningApp(t)
	answer := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerBody(app.lesson.ID, "口渴只是信号，还要结合环境和身体状态。"))
	if answer.Code != http.StatusOK {
		t.Fatalf("seed answer failed: %d %s", answer.Code, answer.Body.String())
	}
	generated := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/lessons/"+strconv.Itoa(int(app.lesson.ID))+"/challenges", `{"challenge_type":"transfer"}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate challenge failed: %d %s", generated.Code, generated.Body.String())
	}
	var payload struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(generated.Body.Bytes(), &payload); err != nil || payload.Data.ID == 0 {
		t.Fatalf("decode challenge: %v %s", err, generated.Body.String())
	}
	path := "/api/v1/courses/1/challenges/" + strconv.Itoa(int(payload.Data.ID)) + "/answers"
	completed := requestJSON(t, app.router, http.MethodPost, path, `{"answer":"高温散步后即使不口渴，也要结合环境和身体状态判断。"}`)
	if completed.Code != http.StatusOK {
		t.Fatalf("answer challenge failed: %d %s", completed.Code, completed.Body.String())
	}
	repeated := requestJSON(t, app.router, http.MethodPost, path, `{"answer":"再次提交"}`)
	if repeated.Code != http.StatusConflict {
		t.Fatalf("repeated challenge should be rejected, got %d %s", repeated.Code, repeated.Body.String())
	}
}

func requestJSON(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func answerBody(lessonID uint, answer string) string {
	payload, _ := json.Marshal(map[string]interface{}{"lesson_id": lessonID, "answer": answer})
	return string(payload)
}

func TestGetCurrentLesson(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/1/current-lesson", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			Course struct {
				Name string `json:"name"`
			} `json:"course"`
			Unit struct {
				Title string `json:"title"`
			} `json:"unit"`
			Lesson struct {
				ID     uint   `json:"id"`
				Title  string `json:"title"`
				Status string `json:"status"`
			} `json:"lesson"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Course.Name != "营养学" || payload.Data.Unit.Title != "饮水信号与日常判断" || payload.Data.Lesson.ID != app.lesson.ID || payload.Data.Lesson.Status != "learning" {
		t.Fatalf("unexpected current lesson response: %+v", payload.Data)
	}
}

func TestGetCurrentLessonCourseNotFound(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/999/current-lesson", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestGetKnowledgeGraph(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/1/knowledge-graph", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			Nodes []struct {
				Title string `json:"title"`
			} `json:"nodes"`
			Edges []struct{} `json:"edges"`
			Stats struct {
				NodeCount int `json:"node_count"`
				EdgeCount int `json:"edge_count"`
			} `json:"stats"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode graph response: %v", err)
	}
	if len(payload.Data.Nodes) != 8 || len(payload.Data.Edges) != 14 || payload.Data.Stats.NodeCount != 8 || payload.Data.Stats.EdgeCount != 14 {
		t.Fatalf("unexpected graph response: nodes=%d edges=%d stats=%+v", len(payload.Data.Nodes), len(payload.Data.Edges), payload.Data.Stats)
	}
}

func TestGetKnowledgeGraphCourseNotFound(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/999/knowledge-graph", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestGetLessonRelationsAndRejectsLessonFromOtherCourse(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/1/lessons/"+strconv.FormatUint(uint64(app.lesson.ID), 10)+"/relations", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			Lesson struct {
				Title string `json:"title"`
			} `json:"lesson"`
			Prerequisites []struct{} `json:"prerequisites"`
			NextLessons   []struct{} `json:"next_lessons"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode relations response: %v", err)
	}
	if payload.Data.Lesson.Title != "口渴是否是可靠的饮水依据" || len(payload.Data.Prerequisites) != 1 || len(payload.Data.NextLessons) != 2 {
		t.Fatalf("unexpected lesson relations: %+v", payload.Data)
	}
	otherLesson := model.Lesson{CourseID: app.course.ID + 1, UnitID: 999, Title: "其他课程知识", CoreQuestion: "问题"}
	if err := app.db.Create(&otherLesson).Error; err != nil {
		t.Fatalf("create other lesson: %v", err)
	}
	response = requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/1/lessons/"+strconv.FormatUint(uint64(otherLesson.ID), 10)+"/relations", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected cross-course lesson to return 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestSeedStarterCourseIsIdempotent(t *testing.T) {
	app := newTestLearningApp(t)
	originalLessonID := app.lesson.ID
	if err := app.service.SeedStarterCourse(context.Background()); err != nil {
		t.Fatalf("seed starter course twice: %v", err)
	}
	var courses, units, lessons, relations int64
	if err := app.db.Model(&model.Course{}).Count(&courses).Error; err != nil {
		t.Fatalf("count courses: %v", err)
	}
	if err := app.db.Model(&model.CourseUnit{}).Count(&units).Error; err != nil {
		t.Fatalf("count units: %v", err)
	}
	if err := app.db.Model(&model.Lesson{}).Count(&lessons).Error; err != nil {
		t.Fatalf("count lessons: %v", err)
	}
	if err := app.db.Model(&model.LessonRelation{}).Count(&relations).Error; err != nil {
		t.Fatalf("count lesson relations: %v", err)
	}
	if courses != 1 || units != 3 || lessons != 8 || relations != 14 {
		t.Fatalf("expected one seeded course graph, got courses=%d units=%d lessons=%d relations=%d", courses, units, lessons, relations)
	}
	var lesson model.Lesson
	if err := app.db.Where("title = ?", "口渴是否是可靠的饮水依据").First(&lesson).Error; err != nil {
		t.Fatalf("find existing lesson: %v", err)
	}
	if lesson.ID != originalLessonID {
		t.Fatalf("existing lesson id changed from %d to %d", originalLessonID, lesson.ID)
	}
}

func TestSeedStarterCoursePreservesLearningHistory(t *testing.T) {
	app := newTestLearningApp(t)
	turn := model.LearningTurn{
		CourseID: app.course.ID, UnitID: *app.course.CurrentUnitID, LessonID: app.lesson.ID,
		Question: "问题", UserAnswer: "回答", Result: "mostly_correct",
	}
	mastery := model.MasteryRecord{CourseID: app.course.ID, LessonID: app.lesson.ID, MasteryScore: 0.75, AnswerCount: 1}
	misconception := model.Misconception{CourseID: app.course.ID, LessonID: app.lesson.ID, OriginalUnderstanding: "原理解", CorrectUnderstanding: "正确理解", Status: "active"}
	run := model.AIEvaluationRun{CourseID: app.course.ID, LessonID: app.lesson.ID, Provider: "mock", Model: "mock", PromptVersion: "test", Status: model.AIEvaluationRunStatusSuccess}
	for _, record := range []interface{}{&turn, &mastery, &misconception, &run} {
		if err := app.db.Create(record).Error; err != nil {
			t.Fatalf("create history record: %v", err)
		}
	}
	if err := app.service.SeedStarterCourse(context.Background()); err != nil {
		t.Fatalf("seed starter course: %v", err)
	}
	var turns, masteries, misconceptions, runs int64
	app.db.Model(&model.LearningTurn{}).Count(&turns)
	app.db.Model(&model.MasteryRecord{}).Count(&masteries)
	app.db.Model(&model.Misconception{}).Count(&misconceptions)
	app.db.Model(&model.AIEvaluationRun{}).Count(&runs)
	if turns != 1 || masteries != 1 || misconceptions != 1 || runs != 1 {
		t.Fatalf("seed changed history counts: turns=%d mastery=%d misconceptions=%d runs=%d", turns, masteries, misconceptions, runs)
	}
}

func TestSubmitAnswerValidation(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerBody(app.lesson.ID, "   "))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestSubmitAnswerPersistsTurnAndMastery(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerBody(app.lesson.ID, "口渴不是唯一的补水依据。"))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var turns int64
	if err := app.db.Model(&model.LearningTurn{}).Where("course_id = ?", app.course.ID).Count(&turns).Error; err != nil {
		t.Fatalf("count learning turns: %v", err)
	}
	if turns != 1 {
		t.Fatalf("expected one learning turn, got %d", turns)
	}
	var mastery model.MasteryRecord
	if err := app.db.Where("lesson_id = ?", app.lesson.ID).First(&mastery).Error; err != nil {
		t.Fatalf("find mastery record: %v", err)
	}
	if mastery.AnswerCount != 1 || mastery.MasteryScore != 0.75 || !mastery.NeedsReview {
		t.Fatalf("unexpected mastery record: %+v", mastery)
	}
}

func TestSubmitAnswerReturnsStructuredEvaluationWithoutRawResponse(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerBody(app.lesson.ID, "口渴不是唯一的补水依据。"))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			Explanation      string `json:"explanation"`
			EvaluationSource string `json:"evaluation_source"`
			Provider         string `json:"provider"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Explanation == "" || payload.Data.EvaluationSource != "mock" || payload.Data.Provider != "mock" {
		t.Fatalf("unexpected structured response: %+v", payload.Data)
	}
	if strings.Contains(response.Body.String(), "raw_response") || strings.Contains(response.Body.String(), "api_key") {
		t.Fatalf("sensitive evaluation fields leaked: %s", response.Body.String())
	}
}

func TestSubmitAnswerAITimeoutReturnsSafeError(t *testing.T) {
	provider := &handlerTestProvider{
		meta: ai.ProviderMeta{Provider: "deepseek", Model: "deepseek-v4-flash", PromptVersion: ai.PromptVersion},
		err:  ai.ErrTimeout,
	}
	app := newTestLearningAppWithProvider(t, provider)
	response := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerBody(app.lesson.ID, "回答保留测试"))
	if response.Code != http.StatusGatewayTimeout || response.Body.String() != `{"error":"AI_TIMEOUT"}` {
		t.Fatalf("unexpected timeout response: status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "Authorization") || strings.Contains(response.Body.String(), "api_key") {
		t.Fatalf("sensitive data leaked in timeout response: %s", response.Body.String())
	}
}

type handlerTestProvider struct {
	result ai.EvaluationResult
	meta   ai.ProviderMeta
	err    error
}

type challengeTimeoutHandlerProvider struct {
	mock           *ai.MockProvider
	challengeCalls int
}

func (p *challengeTimeoutHandlerProvider) EvaluateLessonAnswer(ctx context.Context, request ai.EvaluationRequest) (ai.EvaluationResult, ai.ProviderMeta, error) {
	return p.mock.EvaluateLessonAnswer(ctx, request)
}

func (p *challengeTimeoutHandlerProvider) GenerateChallenge(ctx context.Context, request ai.ChallengeGenerationRequest) (ai.ChallengeGenerationResult, ai.ProviderMeta, error) {
	return p.mock.GenerateChallenge(ctx, request)
}

func (p *challengeTimeoutHandlerProvider) EvaluateChallenge(context.Context, ai.ChallengeEvaluationRequest) (ai.ChallengeEvaluationResult, ai.ProviderMeta, error) {
	p.challengeCalls++
	return ai.ChallengeEvaluationResult{}, ai.ProviderMeta{Provider: "deepseek", Model: "test", PromptVersion: ai.ChallengeEvaluatorPromptVersion, AttemptCount: 2}, ai.ErrTimeout
}

func (p *handlerTestProvider) EvaluateLessonAnswer(context.Context, ai.EvaluationRequest) (ai.EvaluationResult, ai.ProviderMeta, error) {
	return p.result, p.meta, p.err
}

func TestChallengeAnswerProviderTimeoutReturnsAIError(t *testing.T) {
	provider := &challengeTimeoutHandlerProvider{mock: ai.NewMockProvider()}
	app := newTestLearningAppWithProvider(t, provider)
	seed := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerBody(app.lesson.ID, "口渴只是信号，还要结合环境和身体状态。"))
	if seed.Code != http.StatusOK {
		t.Fatalf("seed answer failed: %d %s", seed.Code, seed.Body.String())
	}
	challengePath := "/api/v1/courses/1/lessons/" + strconv.Itoa(int(app.lesson.ID)) + "/challenges"
	generated := requestJSON(t, app.router, http.MethodPost, challengePath, `{"challenge_type":"transfer"}`)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate challenge failed: %d %s", generated.Code, generated.Body.String())
	}
	var payload struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(generated.Body.Bytes(), &payload); err != nil || payload.Data.ID == 0 {
		t.Fatalf("decode challenge: %v %s", err, generated.Body.String())
	}
	answerPath := "/api/v1/courses/1/challenges/" + strconv.Itoa(int(payload.Data.ID)) + "/answers"
	response := requestJSON(t, app.router, http.MethodPost, answerPath, `{"answer":"这是一个有判断过程的正常回答"}`)
	if response.Code != http.StatusGatewayTimeout || response.Body.String() != `{"error":"AI_TIMEOUT"}` {
		t.Fatalf("unexpected challenge timeout response: status=%d body=%s", response.Code, response.Body.String())
	}
	if provider.challengeCalls != 1 {
		t.Fatalf("substantive timeout answer should call provider once, calls=%d", provider.challengeCalls)
	}
}

func TestSubmitAnswerRejectsLessonFromAnotherCourse(t *testing.T) {
	app := newTestLearningApp(t)
	otherCourse := model.Course{Name: "另一门课程", Status: model.CourseStatusLearning}
	if err := app.db.Create(&otherCourse).Error; err != nil {
		t.Fatalf("create other course: %v", err)
	}
	otherLesson := model.Lesson{CourseID: otherCourse.ID, UnitID: *app.course.CurrentUnitID, Title: "其他知识点", CoreQuestion: "问题", Status: model.LessonStatusLearning}
	if err := app.db.Create(&otherLesson).Error; err != nil {
		t.Fatalf("create other lesson: %v", err)
	}

	body := `{"lesson_id":` + strconv.FormatUint(uint64(otherLesson.ID), 10) + `,"answer":"这是一个足够长的回答"}`
	response := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", body)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestLearningTurnsAreReturnedNewestFirst(t *testing.T) {
	app := newTestLearningApp(t)
	for _, answer := range []string{"第一个回答", "第二个回答"} {
		response := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerBody(app.lesson.ID, answer))
		if response.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
		}
	}
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/1/learning-turns?limit=10", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data []struct {
			UserAnswer string `json:"user_answer"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data) != 2 || payload.Data[0].UserAnswer != "第二个回答" {
		t.Fatalf("expected newest answer first, got %+v", payload.Data)
	}
}
