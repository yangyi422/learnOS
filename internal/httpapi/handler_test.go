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
		&model.ExplorationDirection{}, &model.ExplorationQuestion{},
		&model.CurriculumBlueprint{}, &model.CurriculumBlueprintUnit{}, &model.CurriculumBlueprintLesson{}, &model.CurriculumBlueprintRelation{}, &model.CurriculumDraft{},
		&model.KnowledgeSource{}, &model.SourceEvidence{}, &model.GroundingLink{}, &model.SourceCredibilityAssessment{}, &model.GroundingReviewEvent{},
		&model.ExplorationDirection{}, &model.ExplorationQuestion{}, &model.DomainInitializationDraft{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	courseRepository := repository.NewCourseRepository(db)
	learningRepository := repository.NewLearningRepository(db)
	courseService := service.NewCourseService(courseRepository, learningRepository, provider)
	knowledgeGraphRepository := repository.NewKnowledgeGraphRepository(db)
	curriculumRepository := repository.NewCurriculumRepository(db)
	knowledgeGraphService := service.NewKnowledgeGraphService(courseRepository, knowledgeGraphRepository, curriculumRepository)
	cognitiveRepository := repository.NewCognitiveRepository(db)
	cognitiveStateService := service.NewCognitiveStateService(courseRepository, knowledgeGraphRepository, cognitiveRepository)
	courseService.SetCognitiveStateService(cognitiveStateService)
	nextLessonService := service.NewNextLessonService(courseRepository, knowledgeGraphRepository, curriculumRepository, cognitiveRepository, learningRepository)
	misconceptionRepository := repository.NewMisconceptionRepository(db)
	misconceptionService := service.NewMisconceptionService(courseRepository, knowledgeGraphRepository, misconceptionRepository)
	explorationRepository := repository.NewExplorationRepository(db)
	explorationService := service.NewExplorationService(courseRepository, knowledgeGraphRepository, learningRepository, cognitiveRepository, misconceptionRepository, explorationRepository)
	if explorationProvider, ok := provider.(ai.ExplorationProvider); ok {
		explorationService.SetExplorationProvider(explorationProvider)
	}
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

	handler := NewHandler(courseService, knowledgeGraphService, cognitiveStateService, challengeService, misconceptionService, explorationService, nextLessonService)
	webFS := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}
	router := NewRouter(config.Config{}, handler, webFS)
	return testLearningApp{db: db, router: router, service: courseService, course: course, lesson: lesson}
}

func TestSPARouteDeepLinkServesIndexWithoutRedirect(t *testing.T) {
	app := newTestLearningApp(t)
	request := httptest.NewRequest(http.MethodGet, "/courses/1/map", nil)
	recorder := httptest.NewRecorder()
	app.router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("deep SPA route status = %d, want 200; headers=%v body=%s", recorder.Code, recorder.Header(), recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "" {
		t.Fatalf("deep SPA route unexpectedly redirected to %q", location)
	}
	if recorder.Body.String() != "ok" {
		t.Fatalf("deep SPA route body = %q, want index content", recorder.Body.String())
	}
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

func TestLearningLoopWriteAPIsAreIdempotent(t *testing.T) {
	app := newTestLearningApp(t)
	answerPayload := `{"lesson_id":` + strconv.Itoa(int(app.lesson.ID)) + `,"answer":"口渴只是信号，还要结合环境和身体状态。","idempotency_key":"http-answer-1"}`
	firstAnswer := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerPayload)
	secondAnswer := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", answerPayload)
	if firstAnswer.Code != http.StatusOK || secondAnswer.Code != http.StatusOK {
		t.Fatalf("idempotent answer failed: first=%d %s second=%d %s", firstAnswer.Code, firstAnswer.Body.String(), secondAnswer.Code, secondAnswer.Body.String())
	}
	var firstAnswerBody, secondAnswerBody struct {
		Data struct {
			TurnID uint `json:"turn_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(firstAnswer.Body.Bytes(), &firstAnswerBody); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(secondAnswer.Body.Bytes(), &secondAnswerBody); err != nil {
		t.Fatal(err)
	}
	if firstAnswerBody.Data.TurnID == 0 || firstAnswerBody.Data.TurnID != secondAnswerBody.Data.TurnID {
		t.Fatalf("answer replay returned different turns: first=%+v second=%+v", firstAnswerBody, secondAnswerBody)
	}

	challengePath := "/api/v1/courses/1/lessons/" + strconv.Itoa(int(app.lesson.ID)) + "/challenges"
	challengePayload := `{"challenge_type":"transfer","idempotency_key":"http-challenge-1"}`
	firstChallenge := requestJSON(t, app.router, http.MethodPost, challengePath, challengePayload)
	secondChallenge := requestJSON(t, app.router, http.MethodPost, challengePath, challengePayload)
	if firstChallenge.Code != http.StatusOK || secondChallenge.Code != http.StatusOK {
		t.Fatalf("idempotent challenge generation failed: first=%d %s second=%d %s", firstChallenge.Code, firstChallenge.Body.String(), secondChallenge.Code, secondChallenge.Body.String())
	}
	var firstChallengeBody, secondChallengeBody struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(firstChallenge.Body.Bytes(), &firstChallengeBody)
	_ = json.Unmarshal(secondChallenge.Body.Bytes(), &secondChallengeBody)
	if firstChallengeBody.Data.ID == 0 || firstChallengeBody.Data.ID != secondChallengeBody.Data.ID {
		t.Fatalf("challenge replay returned different rows: first=%+v second=%+v", firstChallengeBody, secondChallengeBody)
	}

	challengeAnswerPath := "/api/v1/courses/1/challenges/" + strconv.Itoa(int(firstChallengeBody.Data.ID)) + "/answers"
	challengeAnswerPayload := `{"answer":"高温散步后即使不口渴，也要结合环境和身体状态判断。","idempotency_key":"http-challenge-answer-1"}`
	firstAttempt := requestJSON(t, app.router, http.MethodPost, challengeAnswerPath, challengeAnswerPayload)
	secondAttempt := requestJSON(t, app.router, http.MethodPost, challengeAnswerPath, challengeAnswerPayload)
	if firstAttempt.Code != http.StatusOK || secondAttempt.Code != http.StatusOK {
		t.Fatalf("idempotent challenge answer failed: first=%d %s second=%d %s", firstAttempt.Code, firstAttempt.Body.String(), secondAttempt.Code, secondAttempt.Body.String())
	}
	var turns, challenges, attempts int64
	app.db.Model(&model.LearningTurn{}).Count(&turns)
	app.db.Model(&model.AssessmentChallenge{}).Count(&challenges)
	app.db.Model(&model.ChallengeAttempt{}).Count(&attempts)
	if turns != 2 || challenges != 1 || attempts != 1 {
		t.Fatalf("idempotent HTTP writes duplicated data: turns=%d challenges=%d attempts=%d", turns, challenges, attempts)
	}

	conflict := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/1/answers", `{"lesson_id":`+strconv.Itoa(int(app.lesson.ID))+`,"answer":"不同回答","idempotency_key":"http-answer-1"}`)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("idempotency key reuse with a different payload = %d %s, want 409", conflict.Code, conflict.Body.String())
	}
}

func TestExplorationOpenDoesNotMoveCourseMainline(t *testing.T) {
	app := newTestLearningApp(t)
	var before model.Course
	if err := app.db.First(&before, app.course.ID).Error; err != nil {
		t.Fatalf("load course before exploration: %v", err)
	}
	var turnsBefore, evidenceBefore int64
	app.db.Model(&model.LearningTurn{}).Where("course_id = ?", app.course.ID).Count(&turnsBefore)
	app.db.Model(&model.CognitiveEvidence{}).Where("course_id = ?", app.course.ID).Count(&evidenceBefore)

	radars := requestJSON(t, app.router, http.MethodGet, "/api/v1/exploration/radar?course_id="+strconv.Itoa(int(app.course.ID))+"&lesson_id=0", "")
	if radars.Code != http.StatusOK {
		t.Fatalf("get exploration radar failed: %d %s", radars.Code, radars.Body.String())
	}
	var radarPayload struct {
		Data struct {
			Directions []struct {
				ID             uint   `json:"id"`
				TargetCourseID uint   `json:"target_course_id"`
				DirectionType  string `json:"direction_type"`
			} `json:"directions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(radars.Body.Bytes(), &radarPayload); err != nil || len(radarPayload.Data.Directions) == 0 {
		t.Fatalf("expected exploration direction: %v %s", err, radars.Body.String())
	}
	hasCrossDomain := false
	for _, item := range radarPayload.Data.Directions {
		if item.DirectionType == "cross_domain" {
			hasCrossDomain = true
		}
	}
	if !hasCrossDomain {
		t.Fatalf("expected a curated cross-domain direction: %s", radars.Body.String())
	}
	direction := radarPayload.Data.Directions[0]
	opened := requestJSON(t, app.router, http.MethodPost, "/api/v1/exploration/directions/"+strconv.Itoa(int(direction.ID))+"/open", `{"course_id":1}`)
	if opened.Code != http.StatusOK {
		t.Fatalf("open exploration direction failed: %d %s", opened.Code, opened.Body.String())
	}
	var after model.Course
	if err := app.db.First(&after, app.course.ID).Error; err != nil {
		t.Fatalf("load course after exploration: %v", err)
	}
	if before.CurrentLessonID == nil || after.CurrentLessonID == nil || *before.CurrentLessonID != *after.CurrentLessonID {
		t.Fatalf("exploration changed current lesson: before=%v after=%v", before.CurrentLessonID, after.CurrentLessonID)
	}
	var turnsAfter, evidenceAfter int64
	app.db.Model(&model.LearningTurn{}).Where("course_id = ?", app.course.ID).Count(&turnsAfter)
	app.db.Model(&model.CognitiveEvidence{}).Where("course_id = ?", app.course.ID).Count(&evidenceAfter)
	if turnsBefore != turnsAfter || evidenceBefore != evidenceAfter {
		t.Fatalf("opening exploration created learning records: turns %d/%d evidence %d/%d", turnsBefore, turnsAfter, evidenceBefore, evidenceAfter)
	}
	question := requestJSON(t, app.router, http.MethodPost, "/api/v1/exploration/directions/"+strconv.Itoa(int(direction.ID))+"/questions", `{"course_id":1}`)
	if question.Code != http.StatusOK {
		t.Fatalf("create exploration question failed: %d %s", question.Code, question.Body.String())
	}
	duplicateQuestion := requestJSON(t, app.router, http.MethodPost, "/api/v1/exploration/directions/"+strconv.Itoa(int(direction.ID))+"/questions", `{"course_id":1}`)
	if duplicateQuestion.Code != http.StatusOK {
		t.Fatalf("repeat exploration question failed: %d %s", duplicateQuestion.Code, duplicateQuestion.Body.String())
	}
	var firstQuestion, secondQuestion struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(question.Body.Bytes(), &firstQuestion); err != nil {
		t.Fatalf("decode first exploration question: %v", err)
	}
	if err := json.Unmarshal(duplicateQuestion.Body.Bytes(), &secondQuestion); err != nil {
		t.Fatalf("decode duplicate exploration question: %v", err)
	}
	if firstQuestion.Data.ID == 0 || firstQuestion.Data.ID != secondQuestion.Data.ID {
		t.Fatalf("repeated generation created a duplicate question: %d/%d", firstQuestion.Data.ID, secondQuestion.Data.ID)
	}
}

func TestExplorationRecommendationOwnershipMatchesReferencedLessons(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/exploration/radar?course_id="+strconv.Itoa(int(app.course.ID))+"&lesson_id=0", "")
	if response.Code != http.StatusOK {
		t.Fatalf("get exploration radar failed: %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			Directions []service.ExplorationDirectionView `json:"directions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || len(payload.Data.Directions) == 0 {
		t.Fatalf("decode exploration radar: %v %s", err, response.Body.String())
	}
	for _, direction := range payload.Data.Directions {
		var source model.Lesson
		if err := app.db.First(&source, direction.SourceLessonID).Error; err != nil {
			t.Fatalf("load source lesson %d: %v", direction.SourceLessonID, err)
		}
		if source.CourseID != direction.SourceCourseID ||
			direction.SourceDomainID != source.CourseID ||
			direction.SourceLesson.CourseID != source.CourseID ||
			direction.SourceLesson.ID != source.ID ||
			direction.SourceCourse.ID != source.CourseID ||
			direction.SourceDomain.ID != source.CourseID {
			t.Fatalf("candidate source ownership mismatch: direction=%+v lesson=%+v", direction, source)
		}
		if direction.ContextLessonID != nil {
			var contextLesson model.Lesson
			if err := app.db.First(&contextLesson, *direction.ContextLessonID).Error; err != nil {
				t.Fatalf("load context lesson %d: %v", *direction.ContextLessonID, err)
			}
			if contextLesson.CourseID != direction.ContextCourseID || direction.ContextCourseID != direction.CourseID {
				t.Fatalf("context ownership mismatch: direction=%+v lesson=%+v", direction, contextLesson)
			}
		}

		var target model.Lesson
		if err := app.db.First(&target, direction.TargetLessonID).Error; err != nil {
			t.Fatalf("load target lesson %d: %v", direction.TargetLessonID, err)
		}
		var targetCourse model.Course
		if err := app.db.First(&targetCourse, target.CourseID).Error; err != nil {
			t.Fatalf("load target course %d: %v", target.CourseID, err)
		}
		if target.CourseID != direction.TargetCourseID ||
			direction.TargetLesson.CourseID != target.CourseID ||
			direction.TargetLesson.ID != target.ID ||
			direction.TargetCourse.ID != targetCourse.ID ||
			direction.TargetCourse.Name != targetCourse.Name {
			t.Fatalf("target ownership mismatch: direction=%+v lesson=%+v course=%+v", direction, target, targetCourse)
		}
	}
}

func TestUnfamiliarExplorationSkipsMostRecentLesson(t *testing.T) {
	app := newTestLearningApp(t)
	var before model.Course
	if err := app.db.First(&before, app.course.ID).Error; err != nil {
		t.Fatalf("load course before unfamiliar exploration: %v", err)
	}
	first := requestJSON(t, app.router, http.MethodPost, "/api/v1/exploration/unfamiliar", `{"course_id":1}`)
	second := requestJSON(t, app.router, http.MethodPost, "/api/v1/exploration/unfamiliar", `{"course_id":1}`)
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("unfamiliar exploration failed: %d %d; %s %s", first.Code, second.Code, first.Body.String(), second.Body.String())
	}
	var firstPayload, secondPayload struct {
		Data struct {
			TargetLessonID uint   `json:"target_lesson_id"`
			DirectionType  string `json:"direction_type"`
		} `json:"data"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first unfamiliar direction: %v", err)
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondPayload); err != nil {
		t.Fatalf("decode second unfamiliar direction: %v", err)
	}
	if firstPayload.Data.DirectionType != "unknown" || secondPayload.Data.DirectionType != "unknown" {
		t.Fatalf("unfamiliar endpoint returned non-unknown directions: %s %s", first.Body.String(), second.Body.String())
	}
	if firstPayload.Data.TargetLessonID == 0 || firstPayload.Data.TargetLessonID == secondPayload.Data.TargetLessonID {
		t.Fatalf("unfamiliar endpoint repeated the most recent lesson: %d/%d", firstPayload.Data.TargetLessonID, secondPayload.Data.TargetLessonID)
	}
	var after model.Course
	if err := app.db.First(&after, app.course.ID).Error; err != nil {
		t.Fatalf("load course after unfamiliar exploration: %v", err)
	}
	if before.CurrentLessonID == nil || after.CurrentLessonID == nil || *before.CurrentLessonID != *after.CurrentLessonID {
		t.Fatalf("unfamiliar exploration changed current lesson: before=%v after=%v", before.CurrentLessonID, after.CurrentLessonID)
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

func TestCourseListCurrentLessonMatchesCurrentLessonEndpoint(t *testing.T) {
	app := newTestLearningApp(t)
	listResponse := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses", "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list courses failed: %d %s", listResponse.Code, listResponse.Body.String())
	}
	var listPayload struct {
		Data []model.Course `json:"data"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode course list: %v", err)
	}
	var listed *model.Course
	for index := range listPayload.Data {
		if listPayload.Data[index].ID == app.course.ID {
			listed = &listPayload.Data[index]
			break
		}
	}
	if listed == nil || listed.CurrentLessonID == nil {
		t.Fatalf("course list omitted current lesson pointer: %s", listResponse.Body.String())
	}

	lessonResponse := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/"+strconv.Itoa(int(app.course.ID))+"/current-lesson", "")
	if lessonResponse.Code != http.StatusOK {
		t.Fatalf("get current lesson failed: %d %s", lessonResponse.Code, lessonResponse.Body.String())
	}
	var lessonPayload struct {
		Data struct {
			Lesson model.Lesson `json:"lesson"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lessonResponse.Body.Bytes(), &lessonPayload); err != nil {
		t.Fatalf("decode current lesson: %v", err)
	}
	if *listed.CurrentLessonID != lessonPayload.Data.Lesson.ID || lessonPayload.Data.Lesson.ID != app.lesson.ID {
		t.Fatalf("current lesson mismatch: list=%d endpoint=%d seeded=%d", *listed.CurrentLessonID, lessonPayload.Data.Lesson.ID, app.lesson.ID)
	}
}

func TestExplicitCurrentLessonSwitchAPI(t *testing.T) {
	app := newTestLearningApp(t)
	var target model.Lesson
	if err := app.db.Where("course_id = ? AND id <> ?", app.course.ID, app.lesson.ID).Order("id ASC").First(&target).Error; err != nil {
		t.Fatalf("find target lesson: %v", err)
	}
	response := requestJSON(t, app.router, http.MethodPost, "/api/v1/courses/"+strconv.Itoa(int(app.course.ID))+"/current-lesson", `{"lesson_id":`+strconv.Itoa(int(target.ID))+`}`)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), target.Title) {
		t.Fatalf("switch current lesson failed: %d %s", response.Code, response.Body.String())
	}
	var course model.Course
	if err := app.db.First(&course, app.course.ID).Error; err != nil || course.CurrentLessonID == nil || *course.CurrentLessonID != target.ID {
		t.Fatalf("current lesson pointer not persisted: %+v err=%v", course, err)
	}
	var turns int64
	app.db.Model(&model.LearningTurn{}).Where("course_id = ?", app.course.ID).Count(&turns)
	if turns != 0 {
		t.Fatalf("switch created learning turns: %d", turns)
	}
}

func TestNextLessonAPIReturnsRuleRecommendation(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodGet, "/api/v1/courses/"+strconv.Itoa(int(app.course.ID))+"/next-lesson", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"recommended"`) {
		t.Fatalf("next lesson API failed: %d %s", response.Code, response.Body.String())
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
				Title    string `json:"title"`
				NodeID   string `json:"node_id"`
				NodeType string `json:"node_type"`
				Current  bool   `json:"is_current"`
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
	formalNodes := 0
	blueprintNodes := 0
	currentNodes := 0
	for _, node := range payload.Data.Nodes {
		if node.NodeType == "lesson" && strings.HasPrefix(node.NodeID, "lesson:") {
			formalNodes++
		}
		if node.NodeType == "blueprint" && strings.HasPrefix(node.NodeID, "blueprint:") {
			blueprintNodes++
		}
		if node.Current {
			currentNodes++
		}
	}
	if formalNodes < 8 || blueprintNodes == 0 || currentNodes != 1 || len(payload.Data.Edges) < 14 || payload.Data.Stats.NodeCount != len(payload.Data.Nodes) || payload.Data.Stats.EdgeCount != len(payload.Data.Edges) {
		t.Fatalf("unexpected graph response: formal_nodes=%d blueprint_nodes=%d current_nodes=%d nodes=%d edges=%d stats=%+v", formalNodes, blueprintNodes, currentNodes, len(payload.Data.Nodes), len(payload.Data.Edges), payload.Data.Stats)
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
	if courses != 3 || units != 5 || lessons != 14 || relations != 19 {
		t.Fatalf("expected three seeded course graphs, got courses=%d units=%d lessons=%d relations=%d", courses, units, lessons, relations)
	}
	var lesson model.Lesson
	if err := app.db.Where("title = ?", "口渴是否是可靠的饮水依据").First(&lesson).Error; err != nil {
		t.Fatalf("find existing lesson: %v", err)
	}
	if lesson.ID != originalLessonID {
		t.Fatalf("existing lesson id changed from %d to %d", originalLessonID, lesson.ID)
	}
	for _, item := range []struct {
		name  string
		model interface{}
	}{
		{name: "learning turns", model: &model.LearningTurn{}},
		{name: "mastery records", model: &model.MasteryRecord{}},
		{name: "misconceptions", model: &model.Misconception{}},
		{name: "cognitive states", model: &model.CognitiveState{}},
		{name: "cognitive evidence", model: &model.CognitiveEvidence{}},
		{name: "cognitive events", model: &model.CognitiveStateEvent{}},
		{name: "assessment challenges", model: &model.AssessmentChallenge{}},
		{name: "challenge attempts", model: &model.ChallengeAttempt{}},
	} {
		var count int64
		if err := app.db.Model(item.model).Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", item.name, err)
		}
		if count != 0 {
			t.Fatalf("seed created %s: %d", item.name, count)
		}
	}
}

func TestDeleteCourseAPI(t *testing.T) {
	app := newTestLearningApp(t)
	response := requestJSON(t, app.router, http.MethodDelete, "/api/v1/courses/"+strconv.Itoa(int(app.course.ID)), "")
	if response.Code != http.StatusOK {
		t.Fatalf("delete course failed: %d %s", response.Code, response.Body.String())
	}
	var count int64
	if err := app.db.Model(&model.Course{}).Where("id = ?", app.course.ID).Count(&count).Error; err != nil {
		t.Fatalf("count deleted course: %v", err)
	}
	if count != 0 {
		t.Fatal("delete API left the course in the database")
	}
}

func TestNutritionLessonsHaveCompleteStaticLearningFields(t *testing.T) {
	app := newTestLearningApp(t)
	var lessons []model.Lesson
	if err := app.db.Where("course_id = ?", app.course.ID).Order("id ASC").Find(&lessons).Error; err != nil {
		t.Fatalf("find nutrition lessons: %v", err)
	}
	if len(lessons) != 8 {
		t.Fatalf("expected eight nutrition lessons, got %d", len(lessons))
	}
	ids := make(map[string]uint, len(lessons))
	for _, lesson := range lessons {
		ids[lesson.Title] = lesson.ID
		if lesson.CoreQuestion == "" || lesson.ExpectedUnderstanding == "" || lesson.AssessmentTargetLevel == "" || lesson.ContentRole == "" || lesson.DepthLevel < 1 || lesson.Status == "" {
			t.Fatalf("incomplete nutrition lesson: %+v", lesson)
		}
	}
	var existingLesson model.Lesson
	if err := app.db.Where("course_id = ? AND title = ?", app.course.ID, "口渴是否是可靠的饮水依据").First(&existingLesson).Error; err != nil {
		t.Fatalf("find existing nutrition lesson: %v", err)
	}
	if existingLesson.CoreQuestion != "只要不口渴，是否说明身体不缺水？" {
		t.Fatalf("existing nutrition lesson was not preserved: %+v", existingLesson)
	}
	originalIDs := map[string]uint{}
	for title, id := range ids {
		originalIDs[title] = id
	}
	if err := app.service.SeedStarterCourse(context.Background()); err != nil {
		t.Fatalf("seed nutrition lessons twice: %v", err)
	}
	for title, originalID := range originalIDs {
		var lesson model.Lesson
		if err := app.db.Where("course_id = ? AND title = ?", app.course.ID, title).First(&lesson).Error; err != nil {
			t.Fatalf("find lesson %q after reseed: %v", title, err)
		}
		if lesson.ID != originalID {
			t.Fatalf("lesson %q id changed from %d to %d", title, originalID, lesson.ID)
		}
	}
}

func TestSeedCreatesLogicAndPsychologyKnowledgeIslands(t *testing.T) {
	app := newTestLearningApp(t)
	for _, courseDefinition := range []struct {
		name          string
		unitTitle     string
		lessonList    []string
		relationCount int64
	}{
		{name: "逻辑与科学思维", unitTitle: "日常推理基础", lessonList: []string{"单因素解释的陷阱", "相关不等于因果", "如何判断一条证据有多可靠"}, relationCount: 3},
		{name: "心理学", unitTitle: "认知与行为基础", lessonList: []string{"确认偏误", "情绪如何影响判断", "习惯为什么会自动发生"}, relationCount: 2},
	} {
		var course model.Course
		if err := app.db.Where("name = ?", courseDefinition.name).First(&course).Error; err != nil {
			t.Fatalf("find seeded course %q: %v", courseDefinition.name, err)
		}
		if course.Progress != 0 || course.Status != model.CourseStatusLearning || course.CurrentLessonID == nil {
			t.Fatalf("course with generated content must be learning for %q: %+v", courseDefinition.name, course)
		}
		var unit model.CourseUnit
		if err := app.db.Where("course_id = ? AND title = ?", course.ID, courseDefinition.unitTitle).First(&unit).Error; err != nil {
			t.Fatalf("find unit for %q: %v", courseDefinition.name, err)
		}
		var lessons []model.Lesson
		if err := app.db.Where("course_id = ?", course.ID).Order("sort_order ASC").Find(&lessons).Error; err != nil {
			t.Fatalf("find lessons for %q: %v", courseDefinition.name, err)
		}
		if len(lessons) != len(courseDefinition.lessonList) {
			t.Fatalf("unexpected lesson count for %q: got %d", courseDefinition.name, len(lessons))
		}
		for index, title := range courseDefinition.lessonList {
			lesson := lessons[index]
			if lesson.UnitID != unit.ID || lesson.Title != title || lesson.CoreQuestion == "" || lesson.ExpectedUnderstanding == "" || lesson.AssessmentTargetLevel == "" || lesson.ContentRole == "" || lesson.DepthLevel == 0 {
				t.Fatalf("incomplete seeded lesson for %q: %+v", courseDefinition.name, lesson)
			}
		}
		var relations int64
		if err := app.db.Model(&model.LessonRelation{}).Where("course_id = ?", course.ID).Count(&relations).Error; err != nil {
			t.Fatalf("count relations for %q: %v", courseDefinition.name, err)
		}
		if relations != courseDefinition.relationCount {
			t.Fatalf("unexpected relation count for %q: got %d", courseDefinition.name, relations)
		}
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
	if response.Code != http.StatusGatewayTimeout || !strings.Contains(response.Body.String(), `"code":"AI_TIMEOUT"`) || !strings.Contains(response.Body.String(), `"retryable":true`) {
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
	if response.Code != http.StatusGatewayTimeout || !strings.Contains(response.Body.String(), `"code":"AI_TIMEOUT"`) || !strings.Contains(response.Body.String(), `"retryable":true`) {
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
