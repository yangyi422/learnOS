package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/model"
)

func TestLearningDataExportIncludesFactsAndNeverIncludesAPIKey(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(config.Config{DatabasePath: filepath.Join(dir, "export.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 3, AppVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	secret := "sk-export-must-never-appear"
	if err := db.Create(&model.AIConfiguration{Provider: "deepseek", APIKey: secret, BaseURL: "https://example.test", Model: "test-model"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Course{Name: "Exported course", Goal: "verify export", Status: model.CourseStatusLearning}).Error; err != nil {
		t.Fatal(err)
	}
	project := model.Project{UserID: 7, Title: "Exported project", Status: "active"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ProjectTask{ProjectID: project.ID, Title: "Exported task", Status: "next", Priority: "normal"}).Error; err != nil {
		t.Fatal(err)
	}

	service := NewExportService(db)
	jsonData, err := service.JSON(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	markdownData, err := service.Markdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for format, data := range map[string][]byte{"json": jsonData, "markdown": markdownData} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("%s export leaked API key", format)
		}
		if !strings.Contains(string(data), "Exported course") {
			t.Fatalf("%s export omitted course facts", format)
		}
		if !strings.Contains(string(data), "Exported project") || !strings.Contains(string(data), "Exported task") {
			t.Fatalf("%s export omitted project facts", format)
		}
	}
}
