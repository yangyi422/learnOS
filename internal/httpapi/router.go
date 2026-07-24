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

	// Protect the API and the SPA with the same single-user credentials.
	router.Use(middleware.BasicAuth(cfg.Username, cfg.PasswordHash))

	api := router.Group("/api/v1")
	{
		api.GET("/courses", handler.ListCourses)
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
			c.Request.URL.Path = "/index.html"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return router
}
