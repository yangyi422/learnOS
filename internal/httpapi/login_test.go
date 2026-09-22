package httpapi

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"testing/fstest"

	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoginDistinguishesCredentialsAndStorageFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "login.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}, &model.Session{}); err != nil {
		t.Fatal(err)
	}
	users := service.NewUserService(repository.NewUserRepository(db))
	if _, err := users.Create(context.Background(), service.CreateUserRequest{Username: "learner", DisplayName: "Learner", Password: "password-123"}); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(config.Config{Environment: "production"}, NewHandler(nil, nil, users), fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}, users)
	for _, item := range []struct {
		password string
		status   int
	}{{"wrong", 401}, {"password-123", 200}} {
		response := requestJSON(t, router, http.MethodPost, "/api/v1/auth/login", `{"username":"learner","password":"`+item.password+`"}`)
		if response.Code != item.status {
			t.Fatalf("login status=%d want=%d", response.Code, item.status)
		}
		if item.status == 200 && len(response.Result().Cookies()) == 0 {
			t.Fatal("missing session cookie")
		}
	}
	if err := db.Migrator().DropTable(&model.Session{}); err != nil {
		t.Fatal(err)
	}
	response := requestJSON(t, router, http.MethodPost, "/api/v1/auth/login", `{"username":"learner","password":"password-123"}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("storage failure status=%d", response.Code)
	}
}
