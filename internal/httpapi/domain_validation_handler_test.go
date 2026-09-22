package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"learnos/internal/service"

	"github.com/gin-gonic/gin"
)

func TestDomainInputErrorUsesReadableStructuredResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	(&Handler{}).respondServiceError(context, service.ErrDomainInputInvalid)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	var response struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			Retryable bool   `json:"retryable"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != "DOMAIN_INPUT_INVALID" || response.Error.Message == "" || response.Error.Retryable {
		t.Fatalf("unexpected domain validation response: %+v", response.Error)
	}
}
