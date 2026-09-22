package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"learnos/internal/ai"
	"learnos/internal/auth"
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/httpapi"
	"learnos/internal/repository"
	"learnos/internal/service"
	"learnos/internal/webassets"

	"gorm.io/gorm"
)

type App struct {
	cfg              config.Config
	db               *gorm.DB
	server           *http.Server
	restoreRequested chan struct{}
}

var ErrRestartAfterRestore = errors.New("restart after database restore request")

func New(cfg config.Config) (*App, error) {
	if restored, restoreErr := service.ApplyPendingRestore(cfg); restoreErr != nil {
		return nil, fmt.Errorf("apply pending database restore: %w", restoreErr)
	} else if restored {
		fmt.Printf("restored LearnOS database from confirmed backup request\n")
	}
	db, err := database.Open(cfg)
	if err != nil {
		return nil, err
	}
	userRepository := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepository)
	bootstrapContext := context.Background()
	if cfg.Username != "" && cfg.PasswordHash != "" {
		bootstrapUser, err := userService.EnsureBootstrap(context.Background(), cfg.Username, cfg.PasswordHash)
		if err != nil {
			_ = closeDatabase(db)
			return nil, fmt.Errorf("initialize bootstrap user: %w", err)
		}
		bootstrapContext = auth.WithPrincipal(bootstrapContext, auth.Principal{UserID: bootstrapUser.ID, Username: bootstrapUser.Username, Role: string(bootstrapUser.Role)})
	}

	courseRepository := repository.NewCourseRepository(db)
	learningRepository := repository.NewLearningRepository(db)
	knowledgeGraphRepository := repository.NewKnowledgeGraphRepository(db)
	cognitiveRepository := repository.NewCognitiveRepository(db)
	challengeRepository := repository.NewChallengeRepository(db)
	misconceptionRepository := repository.NewMisconceptionRepository(db)
	explorationRepository := repository.NewExplorationRepository(db)
	curriculumRepository := repository.NewCurriculumRepository(db)
	groundingRepository := repository.NewGroundingRepository(db)
	domainInitializationRepository := repository.NewDomainInitializationRepository(db)
	provider := ai.AIProvider(ai.NewMockProvider())
	if cfg.AIProvider == "deepseek" {
		provider = ai.NewDeepSeekProvider(
			&http.Client{},
			cfg.DeepSeekBaseURL,
			cfg.DeepSeekAPIKey,
			cfg.DeepSeekModel,
			time.Duration(cfg.AITimeoutSeconds)*time.Second,
			time.Duration(cfg.ChallengeGenerationTimeoutSeconds)*time.Second,
			time.Duration(cfg.CurriculumDraftTimeoutSeconds)*time.Second,
			time.Duration(cfg.DomainSkeletonTimeoutSeconds)*time.Second,
			time.Duration(cfg.DomainStarterBlueprintTimeoutSeconds)*time.Second,
			time.Duration(cfg.DomainInitialWorldTimeoutSeconds)*time.Second,
		)
	}
	runtimeProvider := ai.NewRuntimeProvider(provider)
	aiConfigurationRepository := repository.NewAIConfigurationRepository(db)
	aiConfigurationService := service.NewAIConfigurationService(aiConfigurationRepository, cfg, runtimeProvider)
	if err := aiConfigurationService.LoadPersisted(context.Background()); err != nil {
		_ = closeDatabase(db)
		return nil, fmt.Errorf("load AI configuration: %w", err)
	}
	courseService := service.NewCourseService(courseRepository, learningRepository, runtimeProvider)
	knowledgeGraphService := service.NewKnowledgeGraphService(courseRepository, knowledgeGraphRepository, curriculumRepository)
	cognitiveStateService := service.NewCognitiveStateService(courseRepository, knowledgeGraphRepository, cognitiveRepository)
	courseService.SetCognitiveStateService(cognitiveStateService)
	misconceptionService := service.NewMisconceptionService(courseRepository, knowledgeGraphRepository, misconceptionRepository, learningRepository)
	explorationService := service.NewExplorationService(courseRepository, knowledgeGraphRepository, learningRepository, cognitiveRepository, misconceptionRepository, explorationRepository)
	explorationService.SetExplorationProvider(runtimeProvider)
	challengeProvider := ai.ChallengeProvider(runtimeProvider)
	challengeService := service.NewChallengeService(courseRepository, learningRepository, knowledgeGraphRepository, challengeRepository, misconceptionRepository, cognitiveStateService, challengeProvider)
	curriculumService := service.NewCurriculumService(courseRepository, curriculumRepository)
	curriculumService.SetCurriculumDraftProvider(runtimeProvider)
	curriculumService.SetDomainInitializationProvider(runtimeProvider)
	nextLessonService := service.NewNextLessonService(courseRepository, knowledgeGraphRepository, curriculumRepository, cognitiveRepository, learningRepository)
	groundingService := service.NewGroundingService(groundingRepository, curriculumRepository, courseRepository)
	domainInitializationService := service.NewDomainInitializationService(domainInitializationRepository)
	domainInitializationService.SetProvider(runtimeProvider)
	backupService := service.NewBackupService(db, cfg)
	restoreRequested := make(chan struct{}, 1)
	backupService.SetRestartSignal(restoreRequested)
	databaseHealthService := service.NewDatabaseHealthService(db, cfg, backupService)
	databaseHealthService.SetAIConfigurationService(aiConfigurationService)
	exportService := service.NewExportService(db)
	consistencyService := service.NewConsistencyService(db)
	if cfg.DemoSeedEnabled {
		if err := courseService.SeedStarterCourse(bootstrapContext); err != nil {
			_ = closeDatabase(db)
			return nil, fmt.Errorf("seed demo world: %w", err)
		}
	}

	dist, err := webassets.Dist()
	if err != nil {
		_ = closeDatabase(db)
		return nil, fmt.Errorf("load embedded web assets: %w", err)
	}

	handler := httpapi.NewHandler(courseService, knowledgeGraphService, cognitiveStateService, challengeService, misconceptionService, explorationService, curriculumService, groundingService, backupService, databaseHealthService, exportService, consistencyService, domainInitializationService, aiConfigurationService, nextLessonService, userService)
	router := httpapi.NewRouter(cfg, handler, dist, userService)

	return &App{
		cfg:              cfg,
		db:               db,
		restoreRequested: restoreRequested,
		server: &http.Server{
			Addr:              cfg.Address,
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      180 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
	}, nil
}

func (a *App) Run() error {
	fmt.Printf("%s listening on %s\n", a.cfg.AppName, a.cfg.Address)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)
	serverErr := make(chan error, 1)
	go func() { serverErr <- a.server.ListenAndServe() }()
	select {
	case err := <-serverErr:
		closeErr := closeDatabase(a.db)
		if errors.Is(err, http.ErrServerClosed) {
			return closeErr
		}
		return errors.Join(err, closeErr)
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownErr := a.server.Shutdown(ctx)
		closeErr := closeDatabase(a.db)
		return errors.Join(shutdownErr, closeErr)
	case <-a.restoreRequested:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownErr := a.server.Shutdown(ctx)
		closeErr := closeDatabase(a.db)
		return errors.Join(ErrRestartAfterRestore, shutdownErr, closeErr)
	}
}

func closeDatabase(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
