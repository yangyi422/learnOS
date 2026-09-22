package service

import (
	"context"
	"path/filepath"
	"testing"

	"learnos/internal/model"
	"learnos/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserServiceTest(t *testing.T) *UserService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "users.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Session{}, &model.Course{}, &model.DomainInitializationDraft{}); err != nil {
		t.Fatal(err)
	}
	return NewUserService(repository.NewUserRepository(db))
}

func TestUserServiceBootstrapLoginAndAdminCreatedUser(t *testing.T) {
	service := newUserServiceTest(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("admin-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := service.EnsureBootstrap(context.Background(), "admin", string(hash))
	if err != nil {
		t.Fatal(err)
	}
	if admin.Role != model.UserRoleAdmin {
		t.Fatalf("role = %q", admin.Role)
	}
	token, loggedIn, err := service.Login(context.Background(), "admin", "admin-password")
	if err != nil || token == "" || loggedIn.ID != admin.ID {
		t.Fatalf("login = %q %+v %v", token, loggedIn, err)
	}
	created, err := service.Create(context.Background(), CreateUserRequest{Username: "learner", DisplayName: "Learner", Password: "learner-password"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Role != model.UserRoleUser {
		t.Fatalf("created role = %q", created.Role)
	}
	principal, err := service.AuthenticateSession(context.Background(), token)
	if err != nil || principal.UserID != admin.ID || principal.Role != string(model.UserRoleAdmin) {
		t.Fatalf("principal = %+v %v", principal, err)
	}
}

func TestUserServiceBlocksLogin(t *testing.T) {
	service := newUserServiceTest(t)
	user, err := service.Create(context.Background(), CreateUserRequest{Username: "learner", DisplayName: "Learner", Password: "password-123"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetStatus(context.Background(), user.ID, model.UserStatusBlocked); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Login(context.Background(), "learner", "password-123"); err != ErrUserBlocked {
		t.Fatalf("login error = %v", err)
	}
}
