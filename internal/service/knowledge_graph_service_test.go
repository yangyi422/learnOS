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

func TestCurriculumGenerationScopePrefersMissingCore(t *testing.T) {
	coreApplied := uint(1)
	lessons := []model.CurriculumBlueprintLesson{
		{BlueprintUnitID: 7, Importance: model.CurriculumImportanceCore, AppliedLessonID: &coreApplied},
		{BlueprintUnitID: 7, Importance: model.CurriculumImportanceRecommended},
		{BlueprintUnitID: 8, Importance: model.CurriculumImportanceCore},
	}
	if got := curriculumGenerationScope(lessons, 7); got != "missing_recommended" {
		t.Fatalf("scope for unit with only recommended lessons = %q, want missing_recommended", got)
	}
	if got := curriculumGenerationScope(lessons, 8); got != "missing_core" {
		t.Fatalf("scope for unit with missing core lesson = %q, want missing_core", got)
	}
	if got := curriculumGenerationScope(lessons, 9); got != "" {
		t.Fatalf("scope for fully covered or unknown unit = %q, want empty", got)
	}
}

func TestKnowledgeGraphNodeKeysSeparateAppliedAndBlueprintNodes(t *testing.T) {
	lessonID := uint(42)
	blueprintID := uint(9)
	nodes := []KnowledgeGraphNode{
		{ID: lessonID, NodeID: lessonNodeKey(lessonID), NodeType: "lesson", LessonID: &lessonID, BlueprintLessonID: &blueprintID, IsCore: true, DepthLevel: 1},
		{ID: 10, NodeID: blueprintNodeKey(10), NodeType: "blueprint", BlueprintLessonID: uintPtr(10), IsCore: false, DepthLevel: 2},
	}
	if nodes[0].NodeID != "lesson:42" || nodes[1].NodeID != "blueprint:10" || nodes[0].NodeID == nodes[1].NodeID {
		t.Fatalf("unstable or colliding graph node keys: %+v", nodes)
	}
	stats := CalculateKnowledgeGraphStats(nodes, []KnowledgeGraphEdge{
		{EdgeID: "e1", Source: nodes[0].NodeID, Target: nodes[1].NodeID, RelationType: model.LessonRelationPrerequisite},
	})
	if stats.NodeCount != 2 || stats.EdgeCount != 1 || stats.RootNodeCount != 1 || stats.LeafNodeCount != 1 {
		t.Fatalf("unexpected mixed graph stats: %+v", stats)
	}
}

func TestKnowledgeGraphUnitKeysUseStableIDs(t *testing.T) {
	if graphUnitKey(12) != "course-unit:12" || blueprintUnitKey(12) != "blueprint-unit:12" {
		t.Fatalf("knowledge regions must be keyed by stable database IDs")
	}
}

func TestValidateCourseGraphUsesCanonicalContentRolesIndependentOfDepth(t *testing.T) {
	roles := []model.ContentRole{
		model.ContentRoleFoundation,
		model.ContentRoleCore,
		model.ContentRoleDeepening,
		model.ContentRoleApplication,
		model.ContentRoleExtension,
	}
	lessons := make([]model.Lesson, 0, len(roles))
	for index, role := range roles {
		lessons = append(lessons, model.Lesson{
			ID: uint(index + 1), CourseID: 1, ContentRole: role,
			// The same depth is intentional: depth must not determine type.
			DepthLevel: 3,
		})
	}
	if err := ValidateCourseGraph(1, lessons, nil); err != nil {
		t.Fatalf("canonical roles should be valid independent of depth: %v", err)
	}
}
