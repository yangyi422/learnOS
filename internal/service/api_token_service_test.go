package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAPITokenTest(t *testing.T) (*APITokenService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "token.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.APIToken{}, &model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user := model.User{Username: "owner", DisplayName: "Owner", PasswordHash: "x", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return NewAPITokenService(repository.NewAPITokenRepository(db), repository.NewUserRepository(db)), db
}

func TestAPITokenCreateReturnsSecretOnceAndAuthenticatesScoped(t *testing.T) {
	svc, db := newAPITokenTest(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, 1, "uTools 采集", 30)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(created.Token) < 20 || created.Token[:8] != "los_pat_" {
		t.Fatalf("unexpected token: %q", created.Token)
	}
	if created.Prefix == created.Token {
		t.Fatal("prefix must not be the whole secret")
	}

	// The secret is stored hashed only.
	var stored model.APIToken
	if err := db.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.TokenHash == created.Token || len(stored.TokenHash) != 64 {
		t.Fatalf("token stored unhashed: %q", stored.TokenHash)
	}

	principal, err := svc.AuthenticateToken(ctx, created.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal.UserID != 1 || principal.Role == string(model.UserRoleAdmin) {
		t.Fatalf("token principal must be the owner but never admin: %+v", principal)
	}
	if !principal.Allows(model.TokenScopeCaptureInbox) || principal.Allows("admin") {
		t.Fatalf("unexpected scopes: %+v", principal.Scopes)
	}

	// Listing exposes metadata only.
	views, err := svc.List(ctx, 1)
	if err != nil || len(views) != 1 {
		t.Fatalf("list: %+v %v", views, err)
	}
	if views[0].Prefix != created.Prefix || views[0].Expired || views[0].RevokedAt != nil {
		t.Fatalf("unexpected view: %+v", views[0])
	}
}

func TestAPITokenRejectionPaths(t *testing.T) {
	svc, db := newAPITokenTest(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, 1, "", 0) // default ttl
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Unknown and malformed credentials are rejected the same way.
	for _, bad := range []string{"", "los_pat_deadbeef", "Bearer x", "plain"} {
		if _, err := svc.AuthenticateToken(ctx, bad); !errors.Is(err, ErrInvalidAPIToken) {
			t.Fatalf("credential %q error = %v", bad, err)
		}
	}

	// Revoked tokens stop working immediately.
	if err := svc.Revoke(ctx, 1, created.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := svc.AuthenticateToken(ctx, created.Token); !errors.Is(err, ErrInvalidAPIToken) {
		t.Fatalf("revoked token error = %v", err)
	}
	if err := svc.Revoke(ctx, 1, created.ID); !errors.Is(err, repository.ErrAPITokenNotFound) {
		t.Fatalf("second revoke error = %v", err)
	}

	// Expired tokens stop working too.
	expired, err := svc.Create(ctx, 1, "old", 1)
	if err != nil {
		t.Fatalf("create expired: %v", err)
	}
	if err := db.Model(&model.APIToken{}).Where("id = ?", expired.ID).
		Update("expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AuthenticateToken(ctx, expired.Token); !errors.Is(err, ErrInvalidAPIToken) {
		t.Fatalf("expired token error = %v", err)
	}

	// A blocked owner invalidates their tokens without touching the row.
	blocked, err := svc.Create(ctx, 1, "blocked owner", 30)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.User{}).Where("id = 1").Update("status", model.UserStatusBlocked).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AuthenticateToken(ctx, blocked.Token); !errors.Is(err, ErrInvalidAPIToken) {
		t.Fatalf("blocked owner error = %v", err)
	}
}

func TestAPITokenCreateValidation(t *testing.T) {
	svc, _ := newAPITokenTest(t)
	ctx := context.Background()
	if _, err := svc.Create(ctx, 1, "x", -1); !errors.Is(err, ErrInvalidTokenInput) {
		t.Fatalf("negative ttl error = %v", err)
	}
	if _, err := svc.Create(ctx, 1, "x", maxTokenDays+1); !errors.Is(err, ErrInvalidTokenInput) {
		t.Fatalf("oversized ttl error = %v", err)
	}
}
