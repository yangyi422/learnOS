package httpapi

import (
	"io/fs"
	"net/http"
	"strings"

	"learnos/internal/config"
	"learnos/internal/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg config.Config, handler *Handler, webFS fs.FS) *gin.Engine {
	if cfg.Production() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Keep health checks public so Docker and reverse proxies can probe the app.
	router.GET("/healthz", handler.Health)
	router.GET("/health", handler.Health)

	// Local development is loopback-only and intentionally skips browser auth.
	// Every other environment keeps the single-user authentication middleware.
	if !cfg.Development() {
		router.Use(middleware.BasicAuth(cfg.Username, cfg.PasswordHash))
	}

	api := router.Group("/api/v1")
	{
		api.GET("/courses", handler.ListCourses)
		api.DELETE("/courses/:id", handler.DeleteCourse)
		api.POST("/domains/drafts", handler.CreateDomainDraft)
		api.GET("/domains/drafts/:draftId", handler.GetDomainDraft)
		api.POST("/domains/drafts/:draftId/regenerate-skeleton", handler.RegenerateDomainSkeleton)
		api.POST("/domains/drafts/:draftId/expand-starter", handler.ExpandDomainStarter)
		api.POST("/domains/drafts/:draftId/generate-initial-world", handler.GenerateDomainInitialWorld)
		api.POST("/domains/drafts/:draftId/apply", handler.ApplyDomainDraft)
		api.GET("/courses/:id/current-lesson", handler.GetCurrentLesson)
		api.POST("/courses/:id/current-lesson", handler.SetCurrentLesson)
		api.GET("/courses/:id/next-lesson", handler.GetNextLesson)
		api.GET("/courses/:id/lessons/:lessonId/learning", handler.GetLessonForLearning)
		api.POST("/courses/:id/answers", handler.SubmitAnswer)
		api.POST("/courses/:id/lessons/:lessonId/answers", handler.SubmitLessonAnswer)
		api.GET("/courses/:id/learning-turns", handler.ListLearningTurns)
		api.GET("/courses/:id/knowledge-graph", handler.GetKnowledgeGraph)
		api.GET("/courses/:id/lessons/:lessonId/relations", handler.GetLessonRelations)
		api.GET("/courses/:id/cognitive-states", handler.GetCourseCognitiveStates)
		api.GET("/courses/:id/lessons/:lessonId/cognitive-state", handler.GetLessonCognitiveState)
		api.POST("/courses/:id/lessons/:lessonId/challenges", handler.CreateChallenge)
		api.POST("/courses/:id/challenges/:challengeId/answers", handler.AnswerChallenge)
		api.GET("/courses/:id/misconception-network", handler.GetMisconceptionNetwork)
		api.GET("/courses/:id/lessons/:lessonId/misconceptions", handler.ListLessonMisconceptions)
		api.POST("/courses/:id/misconceptions/:misconceptionId/review", handler.ReviewMisconception)
		api.GET("/exploration/radar", handler.GetExplorationRadar)
		api.POST("/exploration/unfamiliar", handler.FindUnfamiliarKnowledge)
		api.POST("/exploration/directions/:directionId/save", handler.SaveExplorationDirection)
		api.POST("/exploration/directions/:directionId/dismiss", handler.DismissExplorationDirection)
		api.POST("/exploration/directions/:directionId/open", handler.OpenExplorationDirection)
		api.POST("/exploration/directions/:directionId/questions", handler.AddExplorationQuestion)
		api.POST("/exploration/directions/:directionId/questions/undo", handler.UndoExplorationQuestion)
		api.GET("/exploration/questions", handler.ListExplorationQuestions)
		api.POST("/exploration/questions/:questionId/start", handler.StartExplorationQuestion)
		api.POST("/exploration/questions/:questionId/later", handler.LaterExplorationQuestion)
		api.POST("/exploration/questions/:questionId/resolve", handler.ResolveExplorationQuestion)
		api.POST("/exploration/questions/:questionId/reopen", handler.ReopenExplorationQuestion)
		api.POST("/exploration/questions/:questionId/priority", handler.SetExplorationQuestionPriority)
		api.POST("/exploration/questions/:questionId/archive", handler.ArchiveExplorationQuestion)
		api.GET("/exploration/history", handler.GetExplorationHistory)
		api.GET("/courses/:id/curriculum", handler.GetCurriculum)
		api.GET("/courses/:id/curriculum/coverage", handler.GetCurriculumCoverage)
		api.POST("/courses/:id/curriculum/units/:blueprintUnitId/expand", handler.ExpandBlueprintUnit)
		api.POST("/courses/:id/curriculum/drafts", handler.CreateCurriculumDraft)
		api.GET("/courses/:id/curriculum/drafts", handler.ListCurriculumDrafts)
		api.GET("/courses/:id/curriculum/drafts/:draftId", handler.GetCurriculumDraft)
		api.POST("/courses/:id/curriculum/drafts/:draftId/apply", handler.ApplyCurriculumDraft)
		api.POST("/courses/:id/curriculum/drafts/:draftId/reject", handler.RejectCurriculumDraft)
		api.GET("/sources", handler.ListSources)
		api.POST("/sources", handler.CreateSource)
		api.GET("/sources/:sourceId", handler.GetSource)
		api.PATCH("/sources/:sourceId", handler.UpdateSource)
		api.POST("/sources/:sourceId/evidence", handler.CreateSourceEvidence)
		api.GET("/sources/:sourceId/evidence", handler.ListSourceEvidence)
		api.GET("/sources/:sourceId/grounding-links", handler.ListSourceGroundingLinks)
		api.PATCH("/evidence/:evidenceId", handler.UpdateSourceEvidence)
		api.POST("/grounding/links", handler.CreateGroundingLink)
		api.GET("/grounding/links", handler.ListGroundingLinks)
		api.POST("/grounding/links/:linkId/review", handler.ReviewGroundingLink)
		api.POST("/grounding/links/:linkId/reject", handler.RejectGroundingLink)
		api.GET("/sources/:sourceId/credibility", handler.ListSourceCredibility)
		api.POST("/sources/:sourceId/credibility", handler.CreateSourceCredibility)
		api.POST("/sources/:sourceId/credibility/:assessmentId/review", handler.ReviewSourceCredibility)
		api.GET("/grounding/targets/:targetType/:targetId", handler.GetGroundingTarget)
		api.GET("/courses/:id/grounding/coverage", handler.GetGroundingCoverage)
		api.POST("/system/backups", handler.CreateBackup)
		api.GET("/system/backups", handler.ListBackups)
		api.POST("/system/backups/restore", handler.RequestBackupRestore)
		api.POST("/system/export", handler.ExportData)
		api.GET("/system/diagnostics", handler.GetDiagnostics)
		api.GET("/system/consistency", handler.GetConsistency)
		api.GET("/system/ai-config", handler.GetAIConfiguration)
		api.PATCH("/system/ai-config", handler.UpdateAIConfiguration)
		api.POST("/system/ai-config/test", handler.TestAIConnection)
	}

	fileServer := http.FileServer(http.FS(webFS))
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(webFS, path); err != nil {
			// Serve the SPA entry through the directory root. Passing
			// /index.html to http.FileServer makes it redirect to ./; on a
			// history-mode route that relative redirect points back to the
			// deep link and loops forever on browser refresh.
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return router
}
