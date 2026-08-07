package repository

import (
	"context"
	"path/filepath"
	"testing"

	"learnos/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newKnowledgeGraphRepositoryTest(t *testing.T) (*gorm.DB, *KnowledgeGraphRepository, model.Course, model.Lesson, model.Lesson) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.Course{}, &model.CourseUnit{}, &model.Lesson{}, &model.LessonRelation{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	course := model.Course{Name: "测试课程", Status: model.CourseStatusLearning}
	if err := db.Create(&course).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	unit := model.CourseUnit{CourseID: course.ID, Title: "测试模块"}
	if err := db.Create(&unit).Error; err != nil {
		t.Fatalf("create unit: %v", err)
	}
	from := model.Lesson{CourseID: course.ID, UnitID: unit.ID, Title: "起点", CoreQuestion: "问题"}
	to := model.Lesson{CourseID: course.ID, UnitID: unit.ID, Title: "终点", CoreQuestion: "问题"}
	if err := db.Create(&from).Error; err != nil {
		t.Fatalf("create from lesson: %v", err)
	}
	if err := db.Create(&to).Error; err != nil {
		t.Fatalf("create to lesson: %v", err)
	}
	return db, NewKnowledgeGraphRepository(db), course, from, to
}

func TestKnowledgeGraphRepositoryPersistsAndListsRelations(t *testing.T) {
	db, repository, course, from, to := newKnowledgeGraphRepositoryTest(t)
	relation := model.LessonRelation{CourseID: course.ID, FromLessonID: from.ID, ToLessonID: to.ID, RelationType: model.LessonRelationPrerequisite}
	if err := db.Create(&relation).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}
	if err := db.Create(&model.LessonRelation{CourseID: course.ID, FromLessonID: from.ID, ToLessonID: to.ID, RelationType: model.LessonRelationPrerequisite}).Error; err == nil {
		t.Fatal("expected duplicate relation to be rejected")
	}

	byCourse, err := repository.ListRelationsByCourse(context.Background(), course.ID)
	if err != nil || len(byCourse) != 1 {
		t.Fatalf("list relations by course: len=%d err=%v", len(byCourse), err)
	}
	byLesson, err := repository.ListRelationsForLesson(context.Background(), course.ID, to.ID)
	if err != nil || len(byLesson) != 1 || byLesson[0].FromLessonID != from.ID {
		t.Fatalf("list relations for lesson: %+v err=%v", byLesson, err)
	}
	lessons, err := repository.ListLessonsByCourse(context.Background(), course.ID)
	if err != nil || len(lessons) != 2 {
		t.Fatalf("list lessons by course: len=%d err=%v", len(lessons), err)
	}
}
