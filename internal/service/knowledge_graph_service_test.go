package service

import (
	"errors"
	"testing"

	"learnos/internal/model"
)

func testLessons(ids ...uint) []model.Lesson {
	lessons := make([]model.Lesson, 0, len(ids))
	for _, id := range ids {
		lessons = append(lessons, model.Lesson{ID: id, CourseID: 1, IsCore: true, ContentRole: model.ContentRoleCore, DepthLevel: int(id)})
	}
	return lessons
}

func testRelation(from, to uint, relationType model.LessonRelationType) model.LessonRelation {
	return model.LessonRelation{CourseID: 1, FromLessonID: from, ToLessonID: to, RelationType: relationType}
}

func TestTopologicalOrderAllowsBranchesAndMerge(t *testing.T) {
	lessons := testLessons(1, 2, 3, 4)
	relations := []model.LessonRelation{
		testRelation(1, 2, model.LessonRelationPrerequisite),
		testRelation(1, 3, model.LessonRelationPrerequisite),
		testRelation(2, 4, model.LessonRelationPrerequisite),
		testRelation(3, 4, model.LessonRelationPrerequisite),
	}
	order, err := TopologicalOrder(lessons, relations)
	if err != nil {
		t.Fatalf("topological order: %v", err)
	}
	if len(order) != 4 || order[0] != 1 || order[3] != 4 {
		t.Fatalf("unexpected topological order: %v", order)
	}
}

func TestTopologicalOrderRejectsCycles(t *testing.T) {
	for name, relations := range map[string][]model.LessonRelation{
		"two nodes": {
			testRelation(1, 2, model.LessonRelationPrerequisite),
			testRelation(2, 1, model.LessonRelationPrerequisite),
		},
		"three nodes": {
			testRelation(1, 2, model.LessonRelationPrerequisite),
			testRelation(2, 3, model.LessonRelationPrerequisite),
			testRelation(3, 1, model.LessonRelationPrerequisite),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := TopologicalOrder(testLessons(1, 2, 3), relations); !errors.Is(err, ErrPrerequisiteCycle) {
				t.Fatalf("expected cycle error, got %v", err)
			}
		})
	}
}

func TestValidateCourseGraphRejectsSelfRelationAndIgnoresNonPrerequisiteCycles(t *testing.T) {
	if err := ValidateCourseGraph(1, testLessons(1, 2), []model.LessonRelation{testRelation(1, 1, model.LessonRelationRelated)}); !errors.Is(err, ErrInvalidRelation) {
		t.Fatalf("expected self relation error, got %v", err)
	}
	relations := []model.LessonRelation{
		testRelation(1, 2, model.LessonRelationRelated),
		testRelation(2, 1, model.LessonRelationExtends),
		testRelation(1, 2, model.LessonRelationApplication),
	}
	if err := ValidateCourseGraph(1, testLessons(1, 2), relations); err != nil {
		t.Fatalf("non-prerequisite cycle should be allowed: %v", err)
	}
}

func TestCalculateGraphStatsUsesPrerequisiteSubgraph(t *testing.T) {
	lessons := []model.Lesson{
		{ID: 1, IsCore: true, ContentRole: model.ContentRoleCore, DepthLevel: 1},
		{ID: 2, IsCore: true, ContentRole: model.ContentRoleCore, DepthLevel: 2},
		{ID: 3, IsCore: false, ContentRole: model.ContentRoleExtension, DepthLevel: 4},
	}
	relations := []model.LessonRelation{
		testRelation(1, 2, model.LessonRelationPrerequisite),
		testRelation(2, 3, model.LessonRelationPrerequisite),
		testRelation(3, 1, model.LessonRelationRelated),
	}
	stats := CalculateGraphStats(lessons, relations)
	if stats.NodeCount != 3 || stats.EdgeCount != 3 || stats.CoreNodeCount != 2 || stats.OptionalNodeCount != 1 || stats.RootNodeCount != 1 || stats.LeafNodeCount != 1 || stats.MaxDepthLevel != 4 {
		t.Fatalf("unexpected graph stats: %+v", stats)
	}
}
