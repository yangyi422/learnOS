package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	courseService := service.NewCourseService(courseRepository)
	if err := courseService.SeedStarterCourse(context.Background()); err != nil {
		return nil, fmt.Errorf("seed starter course: %w", err)
	}

	dist, err := webassets.Dist()
	if err != nil {
		return nil, fmt.Errorf("load embedded web assets: %w", err)
	}

	handler := httpapi.NewHandler(courseService)
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
