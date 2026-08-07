package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"learnos/internal/ai"
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/httpapi"
	"learnos/internal/repository"
	"learnos/internal/service"
	"learnos/internal/webassets"
)

type App struct {
	cfg    config.Config
	server *http.Server
}

func New(cfg config.Config) (*App, error) {
	db, err := database.Open(cfg)
	if err != nil {
		return nil, err
	}

	courseRepository := repository.NewCourseRepository(db)
	learningRepository := repository.NewLearningRepository(db)
	knowledgeGraphRepository := repository.NewKnowledgeGraphRepository(db)
	cognitiveRepository := repository.NewCognitiveRepository(db)
	challengeRepository := repository.NewChallengeRepository(db)
	misconceptionRepository := repository.NewMisconceptionRepository(db)
	provider := ai.AIProvider(ai.NewMockProvider())
	if cfg.AIProvider == "deepseek" {
		provider = ai.NewDeepSeekProvider(&http.Client{}, cfg.DeepSeekBaseURL, cfg.DeepSeekAPIKey, cfg.DeepSeekModel, time.Duration(cfg.AITimeoutSeconds)*time.Second)
	}
	courseService := service.NewCourseService(courseRepository, learningRepository, provider)
	knowledgeGraphService := service.NewKnowledgeGraphService(courseRepository, knowledgeGraphRepository)
	cognitiveStateService := service.NewCognitiveStateService(courseRepository, knowledgeGraphRepository, cognitiveRepository)
	courseService.SetCognitiveStateService(cognitiveStateService)
	misconceptionService := service.NewMisconceptionService(courseRepository, knowledgeGraphRepository, misconceptionRepository)
	challengeProvider, _ := provider.(ai.ChallengeProvider)
	challengeService := service.NewChallengeService(courseRepository, learningRepository, knowledgeGraphRepository, challengeRepository, misconceptionRepository, cognitiveStateService, challengeProvider)
	if err := courseService.SeedStarterCourse(context.Background()); err != nil {
		return nil, fmt.Errorf("seed starter course: %w", err)
	}

	dist, err := webassets.Dist()
	if err != nil {
		return nil, fmt.Errorf("load embedded web assets: %w", err)
	}

	handler := httpapi.NewHandler(courseService, knowledgeGraphService, cognitiveStateService, challengeService, misconceptionService)
	router := httpapi.NewRouter(cfg, handler, dist)

	return &App{
		cfg: cfg,
		server: &http.Server{
			Addr:              cfg.Address,
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
	}, nil
}

func (a *App) Run() error {
	fmt.Printf("%s listening on %s\n", a.cfg.AppName, a.cfg.Address)
	return a.server.ListenAndServe()
}
