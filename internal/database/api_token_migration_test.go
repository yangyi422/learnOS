package database

import (
	"path/filepath"
	"testing"

	"learnos/internal/config"
	"learnos/internal/model"
)

// Schema 33 adds api_tokens: hashed, scoped, revocable capture credentials.
func TestAPITokenMigrationCreatesTableIdempotently(t *testing.T) {
	cfg := config.Config{
		DatabasePath: filepath.Join(t.TempDir(), "api-token.db"),
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
		if !db.Migrator().HasTable(&model.APIToken{}) {
			t.Fatal("api_tokens table missing")
		}
		for _, column := range []string{"user_id", "name", "token_hash", "prefix", "scopes", "expires_at", "revoked_at", "last_used_at"} {
			if !db.Migrator().HasColumn(&model.APIToken{}, column) {
				t.Fatalf("api_tokens.%s missing", column)
			}
		}
		if attempt == 0 {
			if err := db.Create(&model.APIToken{UserID: 1, Name: "t", TokenHash: "hash-1", Prefix: "los_pat_ab12", Scopes: "capture:inbox"}).Error; err != nil {
				t.Fatalf("insert: %v", err)
			}
			if err := db.Create(&model.APIToken{UserID: 1, Name: "t2", TokenHash: "hash-1", Prefix: "los_pat_cd34", Scopes: "capture:inbox"}).Error; err == nil {
				t.Fatal("duplicate token_hash was accepted")
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
