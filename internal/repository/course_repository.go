package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type CourseRepository struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{db: db}
}

func (r *CourseRepository) List(ctx context.Context) ([]model.Course, error) {
	var courses []model.Course
	if err := r.db.WithContext(ctx).Order("updated_at DESC").Find(&courses).Error; err != nil {
		return nil, fmt.Errorf("list courses: %w", err)
	}
	return courses, nil
}

func (r *CourseRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Course{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count courses: %w", err)
	}
	return count, nil
}

func (r *CourseRepository) FindByID(ctx context.Context, id uint) (*model.Course, error) {
	var course model.Course
	if err := r.db.WithContext(ctx).First(&course, id).Error; err != nil {
		return nil, fmt.Errorf("find course by id: %w", err)
	}
	return &course, nil
}

func (r *CourseRepository) FindByName(ctx context.Context, name string) (*model.Course, error) {
	var course model.Course
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&course).Error; err != nil {
		return nil, fmt.Errorf("find course by name: %w", err)
	}
	return &course, nil
}

func (r *CourseRepository) Create(ctx context.Context, course *model.Course) error {
	if err := r.db.WithContext(ctx).Create(course).Error; err != nil {
		return fmt.Errorf("create course: %w", err)
	}
	return nil
}

type starterUnitDefinition struct {
	Title     string
	Objective string
	SortOrder int
	Status    model.CourseUnitStatus
}

type starterLessonDefinition struct {
	UnitTitle             string
	Title                 string
	CoreQuestion          string
	ExpectedUnderstanding string
	SortOrder             int
	Status                model.LessonStatus
	IsCore                bool
	ContentRole           model.ContentRole
	DepthLevel            int
	AssessmentTargetLevel string
}

type starterRelationDefinition struct {
	FromTitle    string
	ToTitle      string
	RelationType model.LessonRelationType
}

const (
	starterCourseName      = "营养学"
	starterLegacyUnitTitle = "水与体液平衡"
	starterUnitATitle      = "水与体液基础"
	starterUnitBTitle      = "饮水信号与日常判断"
	starterUnitCTitle      = "场景应用与扩展"
	starterExistingLesson  = "口渴是否是可靠的饮水依据"
)

var starterLessonDefinitions = []starterLessonDefinition{
	{
		UnitTitle: starterUnitATitle, Title: "水在人体中的基本作用",
		CoreQuestion:          "水在人体中承担哪些基本作用？",
		ExpectedUnderstanding: "水参与体温调节、物质运输、代谢和多种生理过程，是维持人体正常功能的基础。",
		SortOrder:             1, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleFoundation, DepthLevel: 1,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: starterUnitATitle, Title: "体液平衡是如何维持的",
		CoreQuestion:          "人体如何在摄入与流失之间维持体液平衡？",
		ExpectedUnderstanding: "体液平衡由饮水、食物、水代谢以及尿液、汗液和呼吸等途径的流失共同维持。",
		SortOrder:             2, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleFoundation, DepthLevel: 1,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: starterUnitBTitle, Title: starterExistingLesson,
		CoreQuestion:          "只要不口渴，是否说明身体不缺水？",
		ExpectedUnderstanding: "口渴是身体水分调节的重要信号之一，但不能作为所有场景下唯一的补水依据。高温、运动、疾病、年龄以及个体感知差异都可能影响口渴信号。",
		SortOrder:             1, Status: model.LessonStatusLearning, IsCore: true, ContentRole: model.ContentRoleCore, DepthLevel: 2,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: starterUnitBTitle, Title: "日常饮水需求应该如何判断",
		CoreQuestion:          "日常生活中应该综合哪些信息判断饮水需求？",
		ExpectedUnderstanding: "应结合口渴、活动量、环境、饮食、身体状态和尿液等信息作出有边界的日常判断，而不是依赖单一指标。",
		SortOrder:             2, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleCore, DepthLevel: 2,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: starterUnitCTitle, Title: "高温和运动为什么会改变补水需求",
		CoreQuestion:          "为什么高温和运动会改变补水需求？",
		ExpectedUnderstanding: "高温和运动通常会增加出汗及其他水分流失，因此补水需求会随环境、运动强度和持续时间变化。",
		SortOrder:             1, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleApplication, DepthLevel: 2,
		AssessmentTargetLevel: model.AssessmentTargetApply,
	},
	{
		UnitTitle: starterUnitCTitle, Title: "大量出汗后为什么需要关注电解质",
		CoreQuestion:          "大量出汗后为什么不能只关注水？",
		ExpectedUnderstanding: "大量出汗会同时带走水分和部分电解质，需要结合出汗量、持续时间和具体场景关注电解质平衡。",
		SortOrder:             2, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleApplication, DepthLevel: 3,
		AssessmentTargetLevel: model.AssessmentTargetApply,
	},
	{
		UnitTitle: starterUnitCTitle, Title: "年龄与疾病状态为什么会影响口渴信号",
		CoreQuestion:          "年龄和疾病状态为什么可能影响口渴信号？",
		ExpectedUnderstanding: "年龄、疾病和部分药物或身体状态可能改变口渴感知及水分调节，因此不能机械地把没有口渴等同于没有补水风险。",
		SortOrder:             3, Status: model.LessonStatusPending, IsCore: false, ContentRole: model.ContentRoleExtension, DepthLevel: 3,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: starterUnitCTitle, Title: "如何综合判断不同场景下的补水策略",
		CoreQuestion:          "如何综合判断不同场景下的补水策略？",
		ExpectedUnderstanding: "应综合基础饮水信号、活动和环境、水分及电解质流失、年龄和疾病状态，形成符合具体场景的补水判断。",
		SortOrder:             4, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleApplication, DepthLevel: 4,
		AssessmentTargetLevel: model.AssessmentTargetApply,
	},
}

// Relation directions always point from a base or earlier lesson to the later
// lesson. This keeps prerequisite, extension, and application edges readable
// in the same left-to-right direction in the structure page.
var starterRelationDefinitions = []starterRelationDefinition{
	{FromTitle: "水在人体中的基本作用", ToTitle: "体液平衡是如何维持的", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: "体液平衡是如何维持的", ToTitle: starterExistingLesson, RelationType: model.LessonRelationPrerequisite},
	{FromTitle: "体液平衡是如何维持的", ToTitle: "日常饮水需求应该如何判断", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: "体液平衡是如何维持的", ToTitle: "高温和运动为什么会改变补水需求", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: starterExistingLesson, ToTitle: "日常饮水需求应该如何判断", RelationType: model.LessonRelationRelated},
	{FromTitle: "高温和运动为什么会改变补水需求", ToTitle: "大量出汗后为什么需要关注电解质", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: starterExistingLesson, ToTitle: "年龄与疾病状态为什么会影响口渴信号", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: starterExistingLesson, ToTitle: "如何综合判断不同场景下的补水策略", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: "日常饮水需求应该如何判断", ToTitle: "如何综合判断不同场景下的补水策略", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: "大量出汗后为什么需要关注电解质", ToTitle: "如何综合判断不同场景下的补水策略", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: starterExistingLesson, ToTitle: "年龄与疾病状态为什么会影响口渴信号", RelationType: model.LessonRelationExtends},
	{FromTitle: "体液平衡是如何维持的", ToTitle: "大量出汗后为什么需要关注电解质", RelationType: model.LessonRelationApplication},
	{FromTitle: starterExistingLesson, ToTitle: "如何综合判断不同场景下的补水策略", RelationType: model.LessonRelationApplication},
	{FromTitle: "日常饮水需求应该如何判断", ToTitle: "如何综合判断不同场景下的补水策略", RelationType: model.LessonRelationApplication},
}

// SeedStarterCourse creates or repairs the fixed starter world atomically.
// Existing Lesson and learning-history rows are never deleted or recreated.
func (r *CourseRepository) SeedStarterCourse(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var course model.Course
		err := tx.Where("name = ?", starterCourseName).First(&course).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find starter course: %w", err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			course = model.Course{
				Name:        starterCourseName,
				Description: "以现实饮食应用为目标的系统营养学课程。",
				Goal:        "建立能够判断、搭配并持续调整个人饮食的知识体系。",
				Status:      model.CourseStatusLearning,
				Progress:    48,
			}
			if err := tx.Create(&course).Error; err != nil {
				return fmt.Errorf("create starter course: %w", err)
			}
		}

		unitDefinitions := []starterUnitDefinition{
			{Title: starterUnitATitle, Objective: "理解水和体液平衡的基础机制。", SortOrder: 1, Status: model.CourseUnitStatusPending},
			{Title: starterUnitBTitle, Objective: "建立基于信号和日常情境的饮水判断。", SortOrder: 2, Status: model.CourseUnitStatusLearning},
			{Title: starterUnitCTitle, Objective: "把水分知识应用到不同场景并探索边界。", SortOrder: 3, Status: model.CourseUnitStatusPending},
		}
		units := make(map[string]model.CourseUnit, len(unitDefinitions))
		for _, definition := range unitDefinitions {
			unit, err := ensureStarterUnit(tx, course.ID, definition, definition.Title == starterUnitBTitle)
			if err != nil {
				return err
			}
			units[definition.Title] = unit
		}

		// The original Phase 2 unit is retained by ID and becomes Unit B. This
		// avoids moving the existing B1 lesson and its historical UnitID.
		unitB := units[starterUnitBTitle]
		if err := tx.Model(&unitB).Updates(map[string]interface{}{
			"title":      starterUnitBTitle,
			"objective":  "建立基于信号和日常情境的饮水判断。",
			"sort_order": 2,
		}).Error; err != nil {
			return fmt.Errorf("update starter unit metadata: %w", err)
		}

		lessons := make(map[string]model.Lesson, len(starterLessonDefinitions))
		for _, definition := range starterLessonDefinitions {
			lesson, err := ensureStarterLesson(tx, course.ID, units[definition.UnitTitle].ID, definition)
			if err != nil {
				return err
			}
			lessons[definition.Title] = lesson
		}

		for _, definition := range starterRelationDefinitions {
			from := lessons[definition.FromTitle]
			to := lessons[definition.ToTitle]
			relation := model.LessonRelation{
				CourseID:     course.ID,
				FromLessonID: from.ID,
				ToLessonID:   to.ID,
				RelationType: definition.RelationType,
			}
			var existing model.LessonRelation
			err := tx.Where("course_id = ? AND from_lesson_id = ? AND to_lesson_id = ? AND relation_type = ?", relation.CourseID, relation.FromLessonID, relation.ToLessonID, relation.RelationType).First(&existing).Error
			switch {
			case err == nil:
			case errors.Is(err, gorm.ErrRecordNotFound):
				if err := tx.Create(&relation).Error; err != nil {
					return fmt.Errorf("create starter lesson relation: %w", err)
				}
			default:
				return fmt.Errorf("find starter lesson relation: %w", err)
			}
		}

		now := time.Now()
		if err := tx.Model(&course).Updates(map[string]interface{}{
			"status":            model.CourseStatusLearning,
			"current_unit":      units[starterUnitBTitle].Title,
			"current_unit_id":   units[starterUnitBTitle].ID,
			"current_lesson_id": lessons[starterExistingLesson].ID,
			"last_studied_at":   now,
		}).Error; err != nil {
			return fmt.Errorf("update starter course: %w", err)
		}
		return nil
	})
}

func ensureStarterUnit(tx *gorm.DB, courseID uint, definition starterUnitDefinition, allowLegacyTitle bool) (model.CourseUnit, error) {
	var unit model.CourseUnit
	err := tx.Where("course_id = ? AND title = ?", courseID, definition.Title).First(&unit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) && allowLegacyTitle {
		err = tx.Where("course_id = ? AND title = ?", courseID, starterLegacyUnitTitle).First(&unit).Error
		if err == nil {
			if err := tx.Model(&unit).Updates(map[string]interface{}{
				"title":      definition.Title,
				"objective":  definition.Objective,
				"sort_order": definition.SortOrder,
			}).Error; err != nil {
				return model.CourseUnit{}, fmt.Errorf("migrate starter unit: %w", err)
			}
			unit.Title = definition.Title
			unit.Objective = definition.Objective
			unit.SortOrder = definition.SortOrder
			return unit, nil
		}
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.CourseUnit{}, fmt.Errorf("find starter unit %q: %w", definition.Title, err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		unit = model.CourseUnit{
			CourseID:  courseID,
			Title:     definition.Title,
			Objective: definition.Objective,
			SortOrder: definition.SortOrder,
			Status:    definition.Status,
		}
		if err := tx.Create(&unit).Error; err != nil {
			return model.CourseUnit{}, fmt.Errorf("create starter unit %q: %w", definition.Title, err)
		}
	}
	return unit, nil
}

func ensureStarterLesson(tx *gorm.DB, courseID, unitID uint, definition starterLessonDefinition) (model.Lesson, error) {
	var lesson model.Lesson
	err := tx.Where("course_id = ? AND title = ?", courseID, definition.Title).First(&lesson).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Lesson{}, fmt.Errorf("find starter lesson %q: %w", definition.Title, err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		lesson = model.Lesson{
			CourseID:              courseID,
			UnitID:                unitID,
			Title:                 definition.Title,
			CoreQuestion:          definition.CoreQuestion,
			ExpectedUnderstanding: definition.ExpectedUnderstanding,
			SortOrder:             definition.SortOrder,
			Status:                definition.Status,
			IsCore:                definition.IsCore,
			ContentRole:           definition.ContentRole,
			DepthLevel:            definition.DepthLevel,
			AssessmentTargetLevel: definition.AssessmentTargetLevel,
		}
		if err := tx.Create(&lesson).Error; err != nil {
			return model.Lesson{}, fmt.Errorf("create starter lesson %q: %w", definition.Title, err)
		}
		if err := tx.Model(&lesson).Update("is_core", definition.IsCore).Error; err != nil {
			return model.Lesson{}, fmt.Errorf("repair starter lesson core flag %q: %w", definition.Title, err)
		}
		lesson.IsCore = definition.IsCore
		return lesson, nil
	}
	if err := tx.Model(&lesson).Updates(map[string]interface{}{
		"course_id":               courseID,
		"unit_id":                 unitID,
		"core_question":           definition.CoreQuestion,
		"expected_understanding":  definition.ExpectedUnderstanding,
		"sort_order":              definition.SortOrder,
		"is_core":                 definition.IsCore,
		"content_role":            definition.ContentRole,
		"depth_level":             definition.DepthLevel,
		"assessment_target_level": definition.AssessmentTargetLevel,
	}).Error; err != nil {
		return model.Lesson{}, fmt.Errorf("update starter lesson %q: %w", definition.Title, err)
	}
	if err := tx.Model(&lesson).Update("is_core", definition.IsCore).Error; err != nil {
		return model.Lesson{}, fmt.Errorf("repair starter lesson core flag %q: %w", definition.Title, err)
	}
	lesson.CourseID = courseID
	lesson.UnitID = unitID
	lesson.CoreQuestion = definition.CoreQuestion
	lesson.ExpectedUnderstanding = definition.ExpectedUnderstanding
	lesson.SortOrder = definition.SortOrder
	lesson.IsCore = definition.IsCore
	lesson.ContentRole = definition.ContentRole
	lesson.DepthLevel = definition.DepthLevel
	lesson.AssessmentTargetLevel = definition.AssessmentTargetLevel
	return lesson, nil
}
