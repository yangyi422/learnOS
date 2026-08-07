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

var allowedContentRoles = map[model.ContentRole]struct{}{
	model.ContentRoleFoundation:  {},
	model.ContentRoleCore:        {},
	model.ContentRoleApplication: {},
	model.ContentRoleExtension:   {},
}

type KnowledgeGraphService struct {
	courses *repository.CourseRepository
	graphs  *repository.KnowledgeGraphRepository
}

func NewKnowledgeGraphService(courses *repository.CourseRepository, graphs *repository.KnowledgeGraphRepository) *KnowledgeGraphService {
	return &KnowledgeGraphService{courses: courses, graphs: graphs}
}

type KnowledgeGraphCourse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type KnowledgeGraphUnit struct {
	ID        uint                   `json:"id"`
	Title     string                 `json:"title"`
	Objective string                 `json:"objective"`
	SortOrder int                    `json:"sort_order"`
	Status    model.CourseUnitStatus `json:"status"`
}

type KnowledgeGraphNode struct {
	ID          uint               `json:"id"`
	UnitID      uint               `json:"unit_id"`
	Title       string             `json:"title"`
	IsCore      bool               `json:"is_core"`
	ContentRole model.ContentRole  `json:"content_role"`
	DepthLevel  int                `json:"depth_level"`
	Status      model.LessonStatus `json:"status"`
}

type KnowledgeGraphEdge struct {
	ID           uint                     `json:"id"`
	FromLessonID uint                     `json:"from_lesson_id"`
	ToLessonID   uint                     `json:"to_lesson_id"`
	RelationType model.LessonRelationType `json:"relation_type"`
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
	graphUnits := make([]KnowledgeGraphUnit, 0, len(units))
	for index, unit := range units {
		unitOrder[unit.ID] = index
		graphUnits = append(graphUnits, KnowledgeGraphUnit{
			ID: unit.ID, Title: unit.Title, Objective: unit.Objective, SortOrder: unit.SortOrder, Status: unit.Status,
		})
	}
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

	nodes := make([]KnowledgeGraphNode, 0, len(lessons))
	for _, lesson := range lessons {
		nodes = append(nodes, KnowledgeGraphNode{
			ID: lesson.ID, UnitID: lesson.UnitID, Title: lesson.Title, IsCore: lesson.IsCore,
			ContentRole: lesson.ContentRole, DepthLevel: lesson.DepthLevel, Status: lesson.Status,
		})
	}
	edges := make([]KnowledgeGraphEdge, 0, len(relations))
	for _, relation := range relations {
		edges = append(edges, KnowledgeGraphEdge{
			ID: relation.ID, FromLessonID: relation.FromLessonID, ToLessonID: relation.ToLessonID, RelationType: relation.RelationType,
		})
	}
	stats := CalculateGraphStats(lessons, relations)
	return &KnowledgeGraph{
		Course: KnowledgeGraphCourse{ID: course.ID, Name: course.Name},
		Units:  graphUnits, Nodes: nodes, Edges: edges, Stats: stats,
	}, nil
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
			if _, ok := allowedContentRoles[lesson.ContentRole]; !ok {
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
