package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrLessonNotInCourse = errors.New("lesson not found in course")
	ErrInvalidRelation   = errors.New("invalid lesson relation")
	ErrPrerequisiteCycle = errors.New("prerequisite graph contains a cycle")
)

var allowedLessonRelationTypes = map[model.LessonRelationType]struct{}{
	model.LessonRelationPrerequisite: {},
	model.LessonRelationExtends:      {},
	model.LessonRelationApplication:  {},
	model.LessonRelationRelated:      {},
}

type KnowledgeGraphService struct {
	courses    *repository.CourseRepository
	graphs     *repository.KnowledgeGraphRepository
	curriculum *repository.CurriculumRepository
}

func NewKnowledgeGraphService(courses *repository.CourseRepository, graphs *repository.KnowledgeGraphRepository, curriculum ...*repository.CurriculumRepository) *KnowledgeGraphService {
	var curriculumRepository *repository.CurriculumRepository
	if len(curriculum) > 0 {
		curriculumRepository = curriculum[0]
	}
	return &KnowledgeGraphService{courses: courses, graphs: graphs, curriculum: curriculumRepository}
}

type KnowledgeGraphCourse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type KnowledgeGraphUnit struct {
	Key                  string                 `json:"key"`
	ID                   uint                   `json:"id"`
	Title                string                 `json:"title"`
	Objective            string                 `json:"objective"`
	SortOrder            int                    `json:"sort_order"`
	Status               model.CourseUnitStatus `json:"status"`
	BlueprintUnitID      *uint                  `json:"blueprint_unit_id"`
	ExpansionStatus      string                 `json:"expansion_status"`
	BlueprintLessonCount int                    `json:"blueprint_lesson_count"`
	AppliedLessonCount   int                    `json:"applied_lesson_count"`
	GenerationScope      string                 `json:"generation_scope"`
	NeedsExpansion       bool                   `json:"needs_expansion"`
	NeedsGeneration      bool                   `json:"needs_generation"`
}

type KnowledgeGraphNode struct {
	ID                uint               `json:"id"`
	NodeID            string             `json:"node_id"`
	NodeType          string             `json:"node_type"`
	LessonID          *uint              `json:"lesson_id"`
	BlueprintLessonID *uint              `json:"blueprint_lesson_id"`
	BlueprintUnitID   *uint              `json:"blueprint_unit_id"`
	UnitKey           string             `json:"unit_key"`
	UnitID            uint               `json:"unit_id"`
	Title             string             `json:"title"`
	Summary           string             `json:"summary"`
	Importance        string             `json:"importance"`
	GenerationScope   string             `json:"generation_scope"`
	IsCore            bool               `json:"is_core"`
	ContentRole       model.ContentRole  `json:"content_role"`
	DepthLevel        int                `json:"depth_level"`
	Status            model.LessonStatus `json:"status"`
	NodeStatus        string             `json:"node_status"`
	IsCurrent         bool               `json:"is_current"`
}

type KnowledgeGraphEdge struct {
	ID                    uint                     `json:"id"`
	EdgeID                string                   `json:"edge_id"`
	Source                string                   `json:"source"`
	Target                string                   `json:"target"`
	FromLessonID          uint                     `json:"from_lesson_id"`
	ToLessonID            uint                     `json:"to_lesson_id"`
	FromBlueprintLessonID *uint                    `json:"from_blueprint_lesson_id"`
	ToBlueprintLessonID   *uint                    `json:"to_blueprint_lesson_id"`
	RelationType          model.LessonRelationType `json:"relation_type"`
}

type KnowledgeGraphStats struct {
	NodeCount         int `json:"node_count"`
	EdgeCount         int `json:"edge_count"`
	CoreNodeCount     int `json:"core_node_count"`
	OptionalNodeCount int `json:"optional_node_count"`
	RootNodeCount     int `json:"root_node_count"`
	LeafNodeCount     int `json:"leaf_node_count"`
	MaxDepthLevel     int `json:"max_depth_level"`
}

type KnowledgeGraph struct {
	Course KnowledgeGraphCourse `json:"course"`
	Units  []KnowledgeGraphUnit `json:"units"`
	Nodes  []KnowledgeGraphNode `json:"nodes"`
	Edges  []KnowledgeGraphEdge `json:"edges"`
	Stats  KnowledgeGraphStats  `json:"stats"`
}

type LessonRelationLesson struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

type LessonRelations struct {
	Lesson        LessonRelationLesson   `json:"lesson"`
	Prerequisites []LessonRelationLesson `json:"prerequisites"`
	NextLessons   []LessonRelationLesson `json:"next_lessons"`
	Extensions    []LessonRelationLesson `json:"extensions"`
	Applications  []LessonRelationLesson `json:"applications"`
	Related       []LessonRelationLesson `json:"related"`
}

func (s *KnowledgeGraphService) GetCourseKnowledgeGraph(ctx context.Context, courseID uint) (*KnowledgeGraph, error) {
	if courseID == 0 {
		return nil, ErrInvalidCourseID
	}
	course, err := s.courses.FindByID(ctx, courseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	units, err := s.graphs.ListUnitsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	lessons, err := s.graphs.ListLessonsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	for index := range lessons {
		// Rows created before Phase 4 may have zero values if a legacy SQLite
		// database was migrated without applying column defaults.
		if lessons[index].ContentRole == "" {
			lessons[index].ContentRole = model.ContentRoleCore
		}
		if lessons[index].DepthLevel < 1 {
			lessons[index].DepthLevel = 1
		}
	}
	relations, err := s.graphs.ListRelationsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if err := ValidateCourseGraph(courseID, lessons, relations); err != nil {
		return nil, err
	}

	unitOrder := make(map[uint]int, len(units))
	graphUnitByBlueprintID := map[uint]KnowledgeGraphUnit{}
	blueprintUnits := []model.CurriculumBlueprintUnit{}
	blueprintLessons := []model.CurriculumBlueprintLesson{}
	blueprintRelations := []model.CurriculumBlueprintRelation{}
	if s.curriculum != nil {
		if blueprint, blueprintErr := s.curriculum.FindActiveBlueprint(ctx, courseID); blueprintErr == nil {
			loadedUnits, unitsErr := s.curriculum.ListBlueprintUnits(ctx, blueprint.ID)
			if unitsErr != nil {
				return nil, unitsErr
			}
			blueprintUnits = loadedUnits
			loadedLessons, lessonsErr := s.curriculum.ListBlueprintLessons(ctx, blueprint.ID)
			if lessonsErr != nil {
				return nil, lessonsErr
			}
			blueprintLessons = loadedLessons
			loadedRelations, relationsErr := s.curriculum.ListBlueprintRelations(ctx, blueprint.ID)
			if relationsErr != nil {
				return nil, relationsErr
			}
			blueprintRelations = loadedRelations
			if err := validateBlueprint(blueprintLessons, blueprintRelations); err != nil {
				return nil, err
			}
		} else if !errors.Is(blueprintErr, gorm.ErrRecordNotFound) {
			return nil, blueprintErr
		}
	}
	for index, unit := range units {
		unitOrder[unit.ID] = index
	}
	blueprintUnitByID := make(map[uint]model.CurriculumBlueprintUnit, len(blueprintUnits))
	blueprintLessonCount := make(map[uint]int, len(blueprintUnits))
	appliedLessonCount := make(map[uint]int, len(blueprintUnits))
	generationScopeByUnit := make(map[uint]string, len(blueprintUnits))
	for _, blueprintUnit := range blueprintUnits {
		blueprintUnitByID[blueprintUnit.ID] = blueprintUnit
		generationScopeByUnit[blueprintUnit.ID] = curriculumGenerationScope(blueprintLessons, blueprintUnit.ID)
	}
	lessonByID := make(map[uint]model.Lesson, len(lessons))
	for _, lesson := range lessons {
		lessonByID[lesson.ID] = lesson
	}
	blueprintUnitByCourseUnitID := make(map[uint]uint, len(units))
	for _, unit := range units {
		if unit.BlueprintUnitID == nil {
			continue
		}
		if _, exists := blueprintUnitByID[*unit.BlueprintUnitID]; !exists {
			return nil, fmt.Errorf("%w: course unit %d references missing blueprint unit %d", ErrInvalidRelation, unit.ID, *unit.BlueprintUnitID)
		}
		blueprintUnitByCourseUnitID[unit.ID] = *unit.BlueprintUnitID
	}
	for _, blueprintLesson := range blueprintLessons {
		blueprintLessonCount[blueprintLesson.BlueprintUnitID]++
		if blueprintLesson.AppliedLessonID != nil {
			appliedLessonCount[blueprintLesson.BlueprintUnitID]++
			lesson, exists := lessonByID[*blueprintLesson.AppliedLessonID]
			if !exists {
				continue
			}
			if linked, exists := blueprintUnitByCourseUnitID[lesson.UnitID]; exists && linked != blueprintLesson.BlueprintUnitID {
				return nil, fmt.Errorf("%w: course unit %d is linked to multiple blueprint units", ErrInvalidRelation, lesson.UnitID)
			}
			blueprintUnitByCourseUnitID[lesson.UnitID] = blueprintLesson.BlueprintUnitID
		}
	}
	graphUnits := make([]KnowledgeGraphUnit, 0, len(units))
	for _, unit := range units {
		graphUnit := KnowledgeGraphUnit{
			Key: graphUnitKey(unit.ID), ID: unit.ID, Title: unit.Title, Objective: unit.Objective, SortOrder: unit.SortOrder, Status: unit.Status,
		}
		if blueprintUnitID, ok := blueprintUnitByCourseUnitID[unit.ID]; ok {
			blueprintUnit := blueprintUnitByID[blueprintUnitID]
			graphUnit.BlueprintUnitID = &blueprintUnit.ID
			graphUnit.ExpansionStatus = blueprintUnit.ExpansionStatus
			graphUnit.BlueprintLessonCount = blueprintLessonCount[blueprintUnit.ID]
			graphUnit.AppliedLessonCount = appliedLessonCount[blueprintUnit.ID]
			graphUnit.NeedsExpansion = blueprintUnit.ExpansionStatus != model.CurriculumUnitExpanded
			if blueprintUnit.ExpansionStatus == model.CurriculumUnitExpanded {
				graphUnit.GenerationScope = generationScopeByUnit[blueprintUnit.ID]
				graphUnit.NeedsGeneration = graphUnit.GenerationScope != ""
			}
		}
		graphUnits = append(graphUnits, graphUnit)
		if graphUnit.BlueprintUnitID != nil {
			if _, exists := graphUnitByBlueprintID[*graphUnit.BlueprintUnitID]; exists {
				continue
			}
			graphUnitByBlueprintID[*graphUnit.BlueprintUnitID] = graphUnit
		}
	}
	for _, blueprintUnit := range blueprintUnits {
		if _, ok := graphUnitByBlueprintID[blueprintUnit.ID]; ok {
			continue
		}
		graphUnit := KnowledgeGraphUnit{
			Key:                  blueprintUnitKey(blueprintUnit.ID),
			Title:                blueprintUnit.Title,
			Objective:            blueprintUnit.Description,
			SortOrder:            blueprintUnit.SortOrder,
			BlueprintUnitID:      &blueprintUnit.ID,
			ExpansionStatus:      blueprintUnit.ExpansionStatus,
			BlueprintLessonCount: blueprintLessonCount[blueprintUnit.ID],
			AppliedLessonCount:   appliedLessonCount[blueprintUnit.ID],
			GenerationScope:      generationScopeByUnit[blueprintUnit.ID],
			NeedsExpansion:       blueprintUnit.ExpansionStatus != model.CurriculumUnitExpanded,
		}
		graphUnit.NeedsGeneration = !graphUnit.NeedsExpansion && graphUnit.GenerationScope != ""
		graphUnits = append(graphUnits, graphUnit)
		graphUnitByBlueprintID[blueprintUnit.ID] = graphUnit
	}
	sort.SliceStable(graphUnits, func(i, j int) bool {
		if graphUnits[i].SortOrder != graphUnits[j].SortOrder {
			return graphUnits[i].SortOrder < graphUnits[j].SortOrder
		}
		return graphUnits[i].Key < graphUnits[j].Key
	})
	sort.SliceStable(lessons, func(i, j int) bool {
		leftUnit, leftOK := unitOrder[lessons[i].UnitID]
		rightUnit, rightOK := unitOrder[lessons[j].UnitID]
		if leftOK && rightOK && leftUnit != rightUnit {
			return leftUnit < rightUnit
		}
		if lessons[i].SortOrder != lessons[j].SortOrder {
			return lessons[i].SortOrder < lessons[j].SortOrder
		}
		return lessons[i].ID < lessons[j].ID
	})

	blueprintLessonByKey := make(map[string]model.CurriculumBlueprintLesson, len(blueprintLessons))
	blueprintNodeIDByKey := make(map[string]string, len(blueprintLessons))
	for _, blueprintLesson := range blueprintLessons {
		blueprintLessonByKey[blueprintLesson.Key] = blueprintLesson
		if blueprintLesson.AppliedLessonID != nil {
			if _, ok := lessonByID[*blueprintLesson.AppliedLessonID]; ok {
				blueprintNodeIDByKey[blueprintLesson.Key] = lessonNodeKey(*blueprintLesson.AppliedLessonID)
			}
		}
	}

	nodes := make([]KnowledgeGraphNode, 0, len(lessons)+len(blueprintLessons))
	for _, lesson := range lessons {
		unitKey := graphUnitKey(lesson.UnitID)
		var blueprintLessonID *uint
		var blueprintUnitID *uint
		var summary, importance, generationScope string
		for _, blueprintLesson := range blueprintLessons {
			if blueprintLesson.AppliedLessonID != nil && *blueprintLesson.AppliedLessonID == lesson.ID {
				id := blueprintLesson.ID
				blueprintLessonID = &id
				unitID := blueprintLesson.BlueprintUnitID
				blueprintUnitID = &unitID
				summary = blueprintLesson.Summary
				importance = blueprintLesson.Importance
				generationScope = generationScopeByUnit[unitID]
				break
			}
		}
		nodes = append(nodes, KnowledgeGraphNode{
			ID: lesson.ID, NodeID: lessonNodeKey(lesson.ID), NodeType: "lesson", LessonID: uintPtr(lesson.ID), BlueprintLessonID: blueprintLessonID,
			BlueprintUnitID: blueprintUnitID, UnitKey: unitKey, UnitID: lesson.UnitID, Title: lesson.Title, Summary: summary, Importance: importance, GenerationScope: generationScope,
			IsCore: lesson.IsCore, ContentRole: lesson.ContentRole, DepthLevel: lesson.DepthLevel, Status: lesson.Status, NodeStatus: string(lesson.Status),
			IsCurrent: course.CurrentLessonID != nil && *course.CurrentLessonID == lesson.ID,
		})
	}
	for _, blueprintLesson := range blueprintLessons {
		if blueprintLesson.AppliedLessonID != nil {
			if _, ok := lessonByID[*blueprintLesson.AppliedLessonID]; ok {
				continue
			}
		}
		graphUnit := graphUnitByBlueprintID[blueprintLesson.BlueprintUnitID]
		unitKey := graphUnit.Key
		unitID := graphUnit.ID
		blueprintLessonID := blueprintLesson.ID
		blueprintUnitID := blueprintLesson.BlueprintUnitID
		nodes = append(nodes, KnowledgeGraphNode{
			ID: blueprintLesson.ID, NodeID: blueprintNodeKey(blueprintLesson.ID), NodeType: "blueprint", BlueprintLessonID: &blueprintLessonID,
			BlueprintUnitID: &blueprintUnitID, UnitKey: unitKey, UnitID: unitID, Title: blueprintLesson.Title, Summary: blueprintLesson.Summary,
			Importance: blueprintLesson.Importance, GenerationScope: graphUnit.GenerationScope, IsCore: blueprintLesson.Importance == model.CurriculumImportanceCore, ContentRole: blueprintLesson.ContentRole,
			DepthLevel: blueprintLesson.DepthLevel, NodeStatus: "blueprint",
		})
		blueprintNodeIDByKey[blueprintLesson.Key] = blueprintNodeKey(blueprintLesson.ID)
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].UnitKey != nodes[j].UnitKey {
			return nodes[i].UnitKey < nodes[j].UnitKey
		}
		if nodes[i].DepthLevel != nodes[j].DepthLevel {
			return nodes[i].DepthLevel < nodes[j].DepthLevel
		}
		return nodes[i].NodeID < nodes[j].NodeID
	})
	edges := make([]KnowledgeGraphEdge, 0, len(relations)+len(blueprintRelations))
	seenEdges := map[string]struct{}{}
	for _, relation := range relations {
		source, target := lessonNodeKey(relation.FromLessonID), lessonNodeKey(relation.ToLessonID)
		key := graphEdgeKey(source, target, relation.RelationType)
		seenEdges[key] = struct{}{}
		edges = append(edges, KnowledgeGraphEdge{ID: relation.ID, EdgeID: fmt.Sprintf("lesson-relation:%d", relation.ID), Source: source, Target: target, FromLessonID: relation.FromLessonID, ToLessonID: relation.ToLessonID, RelationType: relation.RelationType})
	}
	for _, relation := range blueprintRelations {
		source, sourceOK := blueprintNodeIDByKey[relation.FromLessonKey]
		target, targetOK := blueprintNodeIDByKey[relation.ToLessonKey]
		if !sourceOK || !targetOK {
			continue
		}
		key := graphEdgeKey(source, target, model.LessonRelationType(relation.RelationType))
		if _, exists := seenEdges[key]; exists {
			continue
		}
		seenEdges[key] = struct{}{}
		fromBlueprint := blueprintLessonByKey[relation.FromLessonKey].ID
		toBlueprint := blueprintLessonByKey[relation.ToLessonKey].ID
		edges = append(edges, KnowledgeGraphEdge{ID: relation.ID, EdgeID: fmt.Sprintf("blueprint-relation:%d", relation.ID), Source: source, Target: target, FromBlueprintLessonID: uintPtr(fromBlueprint), ToBlueprintLessonID: uintPtr(toBlueprint), RelationType: model.LessonRelationType(relation.RelationType)})
	}
	nodeCountByUnit := make(map[string]int, len(graphUnits))
	for _, node := range nodes {
		nodeCountByUnit[node.UnitKey]++
	}
	visibleUnits := make([]KnowledgeGraphUnit, 0, len(graphUnits))
	for _, unit := range graphUnits {
		if nodeCountByUnit[unit.Key] > 0 || unit.NeedsExpansion || unit.NeedsGeneration {
			visibleUnits = append(visibleUnits, unit)
		}
	}
	stats := CalculateKnowledgeGraphStats(nodes, edges)
	return &KnowledgeGraph{
		Course: KnowledgeGraphCourse{ID: course.ID, Name: course.Name},
		Units:  visibleUnits, Nodes: nodes, Edges: edges, Stats: stats,
	}, nil
}

func graphUnitKey(unitID uint) string { return fmt.Sprintf("course-unit:%d", unitID) }

func blueprintUnitKey(unitID uint) string { return fmt.Sprintf("blueprint-unit:%d", unitID) }

func lessonNodeKey(lessonID uint) string { return fmt.Sprintf("lesson:%d", lessonID) }

func blueprintNodeKey(lessonID uint) string { return fmt.Sprintf("blueprint:%d", lessonID) }

func graphEdgeKey(source, target string, relationType model.LessonRelationType) string {
	return source + "\x00" + target + "\x00" + string(relationType)
}

func uintPtr(value uint) *uint { return &value }

func curriculumGenerationScope(lessons []model.CurriculumBlueprintLesson, unitID uint) string {
	missingCore := false
	missingRecommended := false
	for _, lesson := range lessons {
		if lesson.BlueprintUnitID != unitID || lesson.AppliedLessonID != nil {
			continue
		}
		switch lesson.Importance {
		case model.CurriculumImportanceCore:
			missingCore = true
		case model.CurriculumImportanceRecommended:
			missingRecommended = true
		}
	}
	if missingCore {
		return "missing_core"
	}
	if missingRecommended {
		return "missing_recommended"
	}
	return ""
}

func (s *KnowledgeGraphService) GetLessonRelations(ctx context.Context, courseID, lessonID uint) (*LessonRelations, error) {
	if courseID == 0 {
		return nil, ErrInvalidCourseID
	}
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	lesson, err := s.graphs.FindLessonByCourse(ctx, courseID, lessonID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLessonNotInCourse
		}
		return nil, err
	}
	relations, err := s.graphs.ListRelationsForLesson(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	allLessons, err := s.graphs.ListLessonsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	lessonTitles := make(map[uint]string, len(allLessons))
	for _, item := range allLessons {
		lessonTitles[item.ID] = item.Title
	}
	result := &LessonRelations{
		Lesson:        LessonRelationLesson{ID: lesson.ID, Title: lesson.Title},
		Prerequisites: []LessonRelationLesson{},
		NextLessons:   []LessonRelationLesson{},
		Extensions:    []LessonRelationLesson{},
		Applications:  []LessonRelationLesson{},
		Related:       []LessonRelationLesson{},
	}
	for _, relation := range relations {
		neighborID := relation.FromLessonID
		if relation.FromLessonID == lessonID {
			neighborID = relation.ToLessonID
		}
		neighborTitle, ok := lessonTitles[neighborID]
		if !ok {
			return nil, fmt.Errorf("relation %d references missing lesson", relation.ID)
		}
		neighbor := LessonRelationLesson{ID: neighborID, Title: neighborTitle}
		switch relation.RelationType {
		case model.LessonRelationPrerequisite:
			if relation.ToLessonID == lessonID {
				result.Prerequisites = append(result.Prerequisites, neighbor)
			} else {
				result.NextLessons = append(result.NextLessons, neighbor)
			}
		case model.LessonRelationExtends:
			result.Extensions = append(result.Extensions, neighbor)
		case model.LessonRelationApplication:
			result.Applications = append(result.Applications, neighbor)
		case model.LessonRelationRelated:
			result.Related = append(result.Related, neighbor)
		}
	}
	return result, nil
}

func (s *KnowledgeGraphService) ValidateCourseGraph(ctx context.Context, courseID uint) error {
	lessons, err := s.graphs.ListLessonsByCourse(ctx, courseID)
	if err != nil {
		return err
	}
	relations, err := s.graphs.ListRelationsByCourse(ctx, courseID)
	if err != nil {
		return err
	}
	return ValidateCourseGraph(courseID, lessons, relations)
}

func (s *KnowledgeGraphService) TopologicalOrder(ctx context.Context, courseID uint) ([]uint, error) {
	lessons, err := s.graphs.ListLessonsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	relations, err := s.graphs.ListRelationsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if err := ValidateCourseGraph(courseID, lessons, relations); err != nil {
		return nil, err
	}
	return TopologicalOrder(lessons, relations)
}

func ValidateCourseGraph(courseID uint, lessons []model.Lesson, relations []model.LessonRelation) error {
	lessonIDs := make(map[uint]struct{}, len(lessons))
	for _, lesson := range lessons {
		if lesson.CourseID != courseID {
			return fmt.Errorf("%w: lesson %d belongs to another course", ErrInvalidRelation, lesson.ID)
		}
		if lesson.ContentRole != "" {
			if !lesson.ContentRole.Valid() {
				return fmt.Errorf("%w: unsupported content role %q", ErrInvalidRelation, lesson.ContentRole)
			}
		}
		if lesson.DepthLevel != 0 && (lesson.DepthLevel < 1 || lesson.DepthLevel > 5) {
			return fmt.Errorf("%w: depth level for lesson %d must be between 1 and 5", ErrInvalidRelation, lesson.ID)
		}
		lessonIDs[lesson.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(relations))
	for _, relation := range relations {
		if _, ok := allowedLessonRelationTypes[relation.RelationType]; !ok {
			return fmt.Errorf("%w: unsupported relation type %q", ErrInvalidRelation, relation.RelationType)
		}
		if relation.CourseID != courseID || relation.FromLessonID == relation.ToLessonID {
			return fmt.Errorf("%w: invalid endpoints for relation %d", ErrInvalidRelation, relation.ID)
		}
		if _, ok := lessonIDs[relation.FromLessonID]; !ok {
			return fmt.Errorf("%w: missing from lesson %d", ErrInvalidRelation, relation.FromLessonID)
		}
		if _, ok := lessonIDs[relation.ToLessonID]; !ok {
			return fmt.Errorf("%w: missing to lesson %d", ErrInvalidRelation, relation.ToLessonID)
		}
		key := fmt.Sprintf("%d:%d:%s", relation.FromLessonID, relation.ToLessonID, relation.RelationType)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("%w: duplicate relation %s", ErrInvalidRelation, key)
		}
		seen[key] = struct{}{}
	}
	err := ValidatePrerequisiteDAG(lessons, relations)
	return err
}

func ValidatePrerequisiteDAG(lessons []model.Lesson, relations []model.LessonRelation) error {
	_, err := TopologicalOrder(lessons, relations)
	return err
}

func TopologicalOrder(lessons []model.Lesson, relations []model.LessonRelation) ([]uint, error) {
	indegree := make(map[uint]int, len(lessons))
	adjacency := make(map[uint][]uint, len(lessons))
	for _, lesson := range lessons {
		indegree[lesson.ID] = 0
		adjacency[lesson.ID] = []uint{}
	}
	for _, relation := range relations {
		if relation.RelationType != model.LessonRelationPrerequisite {
			continue
		}
		if relation.FromLessonID == relation.ToLessonID {
			return nil, ErrInvalidRelation
		}
		if _, fromOK := indegree[relation.FromLessonID]; !fromOK {
			return nil, fmt.Errorf("%w: missing from lesson %d", ErrInvalidRelation, relation.FromLessonID)
		}
		if _, toOK := indegree[relation.ToLessonID]; !toOK {
			return nil, fmt.Errorf("%w: missing to lesson %d", ErrInvalidRelation, relation.ToLessonID)
		}
		adjacency[relation.FromLessonID] = append(adjacency[relation.FromLessonID], relation.ToLessonID)
		indegree[relation.ToLessonID]++
	}
	queue := make([]uint, 0)
	for lessonID, degree := range indegree {
		if degree == 0 {
			queue = append(queue, lessonID)
		}
	}
	sort.Slice(queue, func(i, j int) bool { return queue[i] < queue[j] })
	order := make([]uint, 0, len(lessons))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)
		sort.Slice(adjacency[current], func(i, j int) bool { return adjacency[current][i] < adjacency[current][j] })
		for _, next := range adjacency[current] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
				sort.Slice(queue, func(i, j int) bool { return queue[i] < queue[j] })
			}
		}
	}
	if len(order) != len(lessons) {
		return nil, ErrPrerequisiteCycle
	}
	return order, nil
}

func CalculateGraphStats(lessons []model.Lesson, relations []model.LessonRelation) KnowledgeGraphStats {
	stats := KnowledgeGraphStats{NodeCount: len(lessons), EdgeCount: len(relations)}
	allIDs := make(map[uint]struct{}, len(lessons))
	incoming := make(map[uint]bool, len(lessons))
	outgoing := make(map[uint]bool, len(lessons))
	for _, lesson := range lessons {
		allIDs[lesson.ID] = struct{}{}
		if lesson.IsCore {
			stats.CoreNodeCount++
		} else {
			stats.OptionalNodeCount++
		}
		if lesson.DepthLevel > stats.MaxDepthLevel {
			stats.MaxDepthLevel = lesson.DepthLevel
		}
	}
	for _, relation := range relations {
		if relation.RelationType != model.LessonRelationPrerequisite {
			continue
		}
		if _, ok := allIDs[relation.FromLessonID]; ok {
			outgoing[relation.FromLessonID] = true
		}
		if _, ok := allIDs[relation.ToLessonID]; ok {
			incoming[relation.ToLessonID] = true
		}
	}
	for lessonID := range allIDs {
		if !incoming[lessonID] {
			stats.RootNodeCount++
		}
		if !outgoing[lessonID] {
			stats.LeafNodeCount++
		}
	}
	return stats
}

func CalculateKnowledgeGraphStats(nodes []KnowledgeGraphNode, edges []KnowledgeGraphEdge) KnowledgeGraphStats {
	stats := KnowledgeGraphStats{NodeCount: len(nodes), EdgeCount: len(edges)}
	allIDs := make(map[string]struct{}, len(nodes))
	incoming := make(map[string]bool, len(nodes))
	outgoing := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		allIDs[node.NodeID] = struct{}{}
		if node.IsCore {
			stats.CoreNodeCount++
		} else {
			stats.OptionalNodeCount++
		}
		if node.DepthLevel > stats.MaxDepthLevel {
			stats.MaxDepthLevel = node.DepthLevel
		}
	}
	for _, edge := range edges {
		if edge.RelationType != model.LessonRelationPrerequisite {
			continue
		}
		if _, ok := allIDs[edge.Source]; ok {
			outgoing[edge.Source] = true
		}
		if _, ok := allIDs[edge.Target]; ok {
			incoming[edge.Target] = true
		}
	}
	for nodeID := range allIDs {
		if !incoming[nodeID] {
			stats.RootNodeCount++
		}
		if !outgoing[nodeID] {
			stats.LeafNodeCount++
		}
	}
	return stats
}
