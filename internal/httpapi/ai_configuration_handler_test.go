package httpapi

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/ai"
	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"
)

func TestAIConfigurationAPIDoesNotReturnAPIKey(t *testing.T) {
	const secret = "secret-not-for-response"
	var frameworkLog bytes.Buffer
	var applicationLog bytes.Buffer
	previousGinWriter := gin.DefaultWriter
	previousLogWriter := log.Writer()
	gin.DefaultWriter = &frameworkLog
	log.SetOutput(&applicationLog)
	t.Cleanup(func() {
		gin.DefaultWriter = previousGinWriter
		log.SetOutput(previousLogWriter)
	})

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.AIConfiguration{}, &model.AIEvaluationRun{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	runtime := ai.NewRuntimeProvider(ai.NewMockProvider())
	aiService := service.NewAIConfigurationService(repository.NewAIConfigurationRepository(db), config.Config{DeepSeekBaseURL: ai.DefaultBaseURL, DeepSeekModel: ai.DefaultModel}, runtime)
	router := NewRouter(config.Config{}, NewHandler(nil, nil, aiService), fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}})

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/system/ai-config", strings.NewReader(`{"provider":"deepseek","api_key":"`+secret+`"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("update configuration failed: %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), secret) {
		t.Fatal("API key leaked in update response")
	}

	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, httptest.NewRequest(http.MethodGet, "/api/v1/system/ai-config", nil))
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), `"api_key_configured":true`) {
		t.Fatalf("unexpected configuration response: %d %s", getResponse.Code, getResponse.Body.String())
	}
	if strings.Contains(getResponse.Body.String(), secret) {
		t.Fatal("API key leaked in configuration response")
	}
	if strings.Contains(frameworkLog.String(), secret) || strings.Contains(applicationLog.String(), secret) {
		t.Fatalf("API key leaked in logs: gin=%q application=%q", frameworkLog.String(), applicationLog.String())
	}
}
