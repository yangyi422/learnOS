package database

import (
	"path/filepath"
	"testing"

	"learnos/internal/config"
	"learnos/internal/model"
)

// Schema 32 adds tool_states: one JSON document per user and tool. The table must
// appear on an existing database without touching business tables, and reopening
// the database must stay idempotent.
func TestToolStateMigrationCreatesTableIdempotently(t *testing.T) {
	cfg := config.Config{
		DatabasePath: filepath.Join(t.TempDir(), "tool-state.db"),
		BackupDir:    t.TempDir(),
	}
	if model.CurrentSchemaVersion != 33 {
		t.Fatalf("schema constant: %d", model.CurrentSchemaVersion)
	}

	for attempt := 0; attempt < 2; attempt++ {
		db, err := Open(cfg)
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		var metadata model.SystemMetadata
		if err := db.First(&metadata, 1).Error; err != nil {
			t.Fatal(err)
		}
		if metadata.SchemaVersion != 33 {
			t.Fatalf("schema version: %d", metadata.SchemaVersion)
		}
		if !db.Migrator().HasTable(&model.ToolState{}) {
			t.Fatal("tool_states table missing")
		}
		for _, column := range []string{"user_id", "tool_key", "state_json", "revision", "created_at", "updated_at"} {
			if !db.Migrator().HasColumn(&model.ToolState{}, column) {
				t.Fatalf("tool_states.%s missing", column)
			}
		}
		// One document per user and tool: prove the unique index on the first
		// open only, since reopening must not require an empty table.
		if attempt == 0 {
			if err := db.Create(&model.ToolState{UserID: 1, ToolKey: "fefw-planner", StateJSON: `{}`, Revision: 1}).Error; err != nil {
				t.Fatalf("insert: %v", err)
			}
			if err := db.Create(&model.ToolState{UserID: 1, ToolKey: "fefw-planner", StateJSON: `{}`, Revision: 1}).Error; err == nil {
				t.Fatal("duplicate (user_id, tool_key) was accepted")
			}
		}
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
