package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"learnos/internal/config"
)

const testPasswordHash = "$2y$12$UX/AaWiy1a2gIVI8T7.SIe3hfh9Yt7/JzFcHvq/UxmLz/B3QeZPL."

func TestRouterAuthenticationDependsOnEnvironment(t *testing.T) {
	tests := []struct {
		name         string
		environment  string
		authenticate bool
		wantStatus   int
	}{
		{name: "development skips auth", environment: "development", wantStatus: http.StatusOK},
		{name: "production requires auth", environment: "production", wantStatus: http.StatusUnauthorized},
		{name: "production accepts credentials", environment: "production", authenticate: true, wantStatus: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := NewRouter(
				config.Config{Environment: test.environment, Username: "admin", PasswordHash: testPasswordHash},
				NewHandler(nil, nil),
				fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}},
			)
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.authenticate {
				request.SetBasicAuth("admin", "change-me")
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
		})
	}
}
