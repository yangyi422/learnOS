package service

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newToolStateServiceTest(t *testing.T) *ToolStateService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.ToolState{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return NewToolStateService(repository.NewToolStateRepository(db))
}

func TestToolStateSaveCreatesThenBumpsRevision(t *testing.T) {
	svc := newToolStateServiceTest(t)
	ctx := context.Background()

	first, err := svc.Save(ctx, 7, model.ToolKeyFEFWPlanner, json.RawMessage(`{"versions":1,"plans":{"k":{"c":{"status":"target"}}}}`), 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if first.Revision != 1 {
		t.Fatalf("first revision = %d, want 1", first.Revision)
	}

	second, err := svc.Save(ctx, 7, model.ToolKeyFEFWPlanner, json.RawMessage(`{"plans":{}}`), 1)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if second.Revision != 2 {
		t.Fatalf("second revision = %d, want 2", second.Revision)
	}

	// Stale writers must not overwrite the newer document.
	if _, err := svc.Save(ctx, 7, model.ToolKeyFEFWPlanner, json.RawMessage(`{"plans":{}}`), 1); !errors.Is(err, repository.ErrToolStateConflict) {
		t.Fatalf("stale write error = %v, want conflict", err)
	}

	// The document is stored compacted and readable back as JSON.
	got, err := svc.Get(ctx, 7, model.ToolKeyFEFWPlanner)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Revision != 2 || len(got.State) == 0 || got.UpdatedAt == "" {
		t.Fatalf("unexpected view: %+v", got)
	}
}

func TestToolStateIsScopedPerUser(t *testing.T) {
	svc := newToolStateServiceTest(t)
	ctx := context.Background()
	if _, err := svc.Save(ctx, 1, model.ToolKeyFEFWPlanner, json.RawMessage(`{"owner":1}`), 0); err != nil {
		t.Fatalf("save user 1: %v", err)
	}
	if _, err := svc.Get(ctx, 2, model.ToolKeyFEFWPlanner); !errors.Is(err, repository.ErrToolStateNotFound) {
		t.Fatalf("other user error = %v, want not found", err)
	}
}

func TestToolStateRejectsUnknownToolsAndBadDocuments(t *testing.T) {
	svc := newToolStateServiceTest(t)
	ctx := context.Background()

	if _, err := svc.Get(ctx, 1, "not-a-tool"); !errors.Is(err, ErrUnknownTool) {
		t.Fatalf("unknown tool error = %v", err)
	}
	for _, bad := range []string{``, `not json`, `[]`, `"text"`, `null`} {
		if _, err := svc.Save(ctx, 1, model.ToolKeyFEFWPlanner, json.RawMessage(bad), 0); !errors.Is(err, ErrInvalidToolState) {
			t.Fatalf("document %q error = %v, want invalid", bad, err)
		}
	}
	tooBig := `{"note":"` + strings.Repeat("x", maxToolStateBytes) + `"}`
	if _, err := svc.Save(ctx, 1, model.ToolKeyFEFWPlanner, json.RawMessage(tooBig), 0); !errors.Is(err, ErrInvalidToolState) {
		t.Fatalf("oversized document error = %v, want invalid", err)
	}
}
