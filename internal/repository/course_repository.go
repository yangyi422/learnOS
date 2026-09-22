package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"learnos/internal/auth"
	"learnos/internal/model"

	"gorm.io/gorm"
)

type CourseRepository struct {
	db *gorm.DB
}

// CourseProgressFacts contains independent facts used to derive the three
// public progress dimensions. None of these values reuse Course.Progress.
type CourseProgressFacts struct {
	LessonCount            int
	BlueprintLessonCount   int
	GeneratedLessonCount   int
	CoveredLessonCount     int
	MasteryPointTotal      int
	NeedsReviewLessonCount int
}

func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{db: db}
}

func courseScope(ctx context.Context, db *gorm.DB) *gorm.DB {
	if principal, ok := auth.PrincipalFromContext(ctx); ok {
		return db.Where("user_id = ?", principal.UserID)
	}
	return db
}

func (r *CourseRepository) List(ctx context.Context) ([]model.Course, error) {
	var courses []model.Course
	if err := courseScope(ctx, r.db.WithContext(ctx)).Order("updated_at DESC").Find(&courses).Error; err != nil {
		return nil, fmt.Errorf("list courses: %w", err)
	}
	return courses, nil
}

func (r *CourseRepository) ListForUser(ctx context.Context, userID uint) ([]model.Course, error) {
	var courses []model.Course
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC").Find(&courses).Error; err != nil {
		return nil, fmt.Errorf("list user courses: %w", err)
	}
	return courses, nil
}

func (r *CourseRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := courseScope(ctx, r.db.WithContext(ctx).Model(&model.Course{})).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count courses: %w", err)
	}
	return count, nil
}

func (r *CourseRepository) ProgressFacts(ctx context.Context, courseID uint) (CourseProgressFacts, error) {
	facts := CourseProgressFacts{}
	var lessonCount int64
	if err := r.db.WithContext(ctx).Model(&model.Lesson{}).Where("course_id = ?", courseID).Count(&lessonCount).Error; err != nil {
		return facts, fmt.Errorf("count course lessons: %w", err)
	}
	facts.LessonCount = int(lessonCount)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM curriculum_blueprint_lessons bl
		JOIN curriculum_blueprints b ON b.id = bl.blueprint_id
		WHERE b.course_id = ? AND b.status = ?`, courseID, model.CurriculumBlueprintStatusActive).Scan(&facts.BlueprintLessonCount).Error; err != nil {
		return facts, fmt.Errorf("count blueprint lessons: %w", err)
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM curriculum_blueprint_lessons bl
		JOIN curriculum_blueprints b ON b.id = bl.blueprint_id
		JOIN lessons l ON l.id = bl.applied_lesson_id AND l.course_id = b.course_id
		WHERE b.course_id = ? AND b.status = ?`, courseID, model.CurriculumBlueprintStatusActive).Scan(&facts.GeneratedLessonCount).Error; err != nil {
		return facts, fmt.Errorf("count generated lessons: %w", err)
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM (
			SELECT lesson_id FROM learning_turns WHERE course_id = ?
			UNION
			SELECT lesson_id FROM cognitive_states WHERE course_id = ? AND current_level <> ?
		) engaged`, courseID, courseID, model.CognitiveLevelUnseen).Scan(&facts.CoveredLessonCount).Error; err != nil {
		return facts, fmt.Errorf("count covered lessons: %w", err)
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(CASE current_level
			WHEN ? THEN 20 WHEN ? THEN 40 WHEN ? THEN 60
			WHEN ? THEN 80 WHEN ? THEN 100 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
		FROM cognitive_states WHERE course_id = ?`,
		model.CognitiveLevelExposed, model.CognitiveLevelRecognize, model.CognitiveLevelUnderstand,
		model.CognitiveLevelApply, model.CognitiveLevelTransfer, model.CognitiveStatusNeedsReview, courseID,
	).Row().Scan(&facts.MasteryPointTotal, &facts.NeedsReviewLessonCount); err != nil {
		return facts, fmt.Errorf("calculate mastery progress: %w", err)
	}
	if facts.CoveredLessonCount > facts.LessonCount {
		facts.CoveredLessonCount = facts.LessonCount
	}
	return facts, nil
}

// ReconcileCourseStatuses repairs the legacy state in which a course has a
// valid current Lesson but remains "initializing". It is idempotent and never
// creates, removes, or re-points learning data.
func (r *CourseRepository) ReconcileCourseStatuses(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE courses
		SET status = ?, updated_at = ?
		WHERE status = ?
		  AND current_unit_id IS NOT NULL
		  AND current_lesson_id IS NOT NULL
		  AND EXISTS (
			SELECT 1 FROM lessons l
			JOIN course_units u ON u.id = l.unit_id AND u.course_id = courses.id
			WHERE l.id = courses.current_lesson_id AND l.course_id = courses.id
			  AND u.id = courses.current_unit_id
		  )`, model.CourseStatusLearning, time.Now().UTC(), model.CourseStatusInitializing)
	if result.Error != nil {
		return 0, fmt.Errorf("reconcile course statuses: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *CourseRepository) FindByID(ctx context.Context, id uint) (*model.Course, error) {
	var course model.Course
	if err := courseScope(ctx, r.db.WithContext(ctx)).First(&course, id).Error; err != nil {
		return nil, fmt.Errorf("find course by id: %w", err)
	}
	return &course, nil
}

func (r *CourseRepository) FindByName(ctx context.Context, name string) (*model.Course, error) {
	var course model.Course
	name = strings.TrimSpace(name)
	if err := courseScope(ctx, r.db.WithContext(ctx)).Where("LOWER(TRIM(name)) = LOWER(?)", name).First(&course).Error; err != nil {
		return nil, fmt.Errorf("find course by name: %w", err)
	}
	return &course, nil
}

func (r *CourseRepository) Create(ctx context.Context, course *model.Course) error {
	if principal, ok := auth.PrincipalFromContext(ctx); ok && course.UserID == 0 {
		course.UserID = principal.UserID
	}
	if err := r.db.WithContext(ctx).Create(course).Error; err != nil {
		return fmt.Errorf("create course: %w", err)
	}
	return nil
}

// SetCurrentLesson changes only the course's explicit mainline pointer. It
// deliberately does not create learning, mastery, or cognitive records.
func (r *CourseRepository) SetCurrentLesson(ctx context.Context, courseID, lessonID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lesson model.Lesson
		if err := tx.Where("id = ? AND course_id = ?", lessonID, courseID).First(&lesson).Error; err != nil {
			return err
		}
		courseQuery := tx.Model(&model.Course{}).Where("id = ?", courseID)
		if principal, ok := auth.PrincipalFromContext(ctx); ok {
			courseQuery = courseQuery.Where("user_id = ?", principal.UserID)
		}
		if err := courseQuery.Updates(map[string]interface{}{
			"current_unit":      "",
			"current_unit_id":   lesson.UnitID,
			"current_lesson_id": lesson.ID,
			"status": gorm.Expr(
				"CASE WHEN status = ? THEN ? ELSE status END",
				model.CourseStatusInitializing,
				model.CourseStatusLearning,
			),
		}).Error; err != nil {
			return fmt.Errorf("set current lesson: %w", err)
		}
		var unit model.CourseUnit
		if err := tx.First(&unit, lesson.UnitID).Error; err != nil {
			return err
		}
		return tx.Model(&model.Course{}).Where("id = ?", courseID).Update("current_unit", unit.Title).Error
	})
}

// Delete removes one complete course graph and its user-generated records in
// a single transaction. Shared knowledge sources and their evidence are kept;
// only links from the deleted course are removed.
func (r *CourseRepository) Delete(ctx context.Context, courseID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var course model.Course
		courseQuery := tx
		if principal, ok := auth.PrincipalFromContext(ctx); ok {
			courseQuery = courseQuery.Where("user_id = ?", principal.UserID)
		}
		if err := courseQuery.First(&course, courseID).Error; err != nil {
			return err
		}

		var lessonIDs []uint
		if err := tx.Model(&model.Lesson{}).Where("course_id = ?", courseID).Pluck("id", &lessonIDs).Error; err != nil {
			return fmt.Errorf("collect course lessons: %w", err)
		}
		var blueprintIDs []uint
		if err := tx.Model(&model.CurriculumBlueprint{}).Where("course_id = ?", courseID).Pluck("id", &blueprintIDs).Error; err != nil {
			return fmt.Errorf("collect course blueprints: %w", err)
		}
		var blueprintLessonIDs []uint
		if len(blueprintIDs) > 0 {
			if err := tx.Model(&model.CurriculumBlueprintLesson{}).Where("blueprint_id IN ?", blueprintIDs).Pluck("id", &blueprintLessonIDs).Error; err != nil {
				return fmt.Errorf("collect blueprint lessons: %w", err)
			}
		}
		var misconceptionIDs []uint
		if err := tx.Model(&model.Misconception{}).Where("course_id = ?", courseID).Pluck("id", &misconceptionIDs).Error; err != nil {
			return fmt.Errorf("collect misconceptions: %w", err)
		}

		for _, target := range []struct {
			typeName string
			ids      []uint
		}{
			{model.GroundingTargetCurriculumBlueprint, blueprintIDs},
			{model.GroundingTargetBlueprintLesson, blueprintLessonIDs},
			{model.GroundingTargetLesson, lessonIDs},
		} {
			if len(target.ids) == 0 {
				continue
			}
			if err := tx.Where("target_type = ? AND target_id IN ?", target.typeName, target.ids).Delete(&model.GroundingLink{}).Error; err != nil {
				return fmt.Errorf("delete %s grounding links: %w", target.typeName, err)
			}
			if err := tx.Where("target_type = ? AND target_id IN ?", target.typeName, target.ids).Delete(&model.GroundingReviewEvent{}).Error; err != nil {
				return fmt.Errorf("delete %s grounding events: %w", target.typeName, err)
			}
		}

		if err := tx.Where("course_id = ? OR target_course_id = ?", courseID, courseID).Delete(&model.ExplorationQuestion{}).Error; err != nil {
			return fmt.Errorf("delete exploration questions: %w", err)
		}
		if err := tx.Where("course_id = ? OR target_course_id = ?", courseID, courseID).Delete(&model.ExplorationDirection{}).Error; err != nil {
			return fmt.Errorf("delete exploration directions: %w", err)
		}
		if err := tx.Where("applied_course_id = ?", courseID).Delete(&model.DomainInitializationDraft{}).Error; err != nil {
			return fmt.Errorf("delete applied domain drafts: %w", err)
		}

		if len(blueprintIDs) > 0 {
			if err := tx.Where("blueprint_id IN ? OR course_id = ?", blueprintIDs, courseID).Delete(&model.CurriculumDraft{}).Error; err != nil {
				return fmt.Errorf("delete curriculum drafts: %w", err)
			}
			for _, item := range []struct {
				model interface{}
				name  string
			}{
				{&model.CurriculumBlueprintRelation{}, "blueprint relations"},
				{&model.CurriculumBlueprintLesson{}, "blueprint lessons"},
				{&model.CurriculumBlueprintUnit{}, "blueprint units"},
			} {
				if err := tx.Where("blueprint_id IN ?", blueprintIDs).Delete(item.model).Error; err != nil {
					return fmt.Errorf("delete %s: %w", item.name, err)
				}
			}
			if err := tx.Where("id IN ?", blueprintIDs).Delete(&model.CurriculumBlueprint{}).Error; err != nil {
				return fmt.Errorf("delete blueprints: %w", err)
			}
		} else if err := tx.Where("course_id = ?", courseID).Delete(&model.CurriculumDraft{}).Error; err != nil {
			return fmt.Errorf("delete curriculum drafts: %w", err)
		}

		if err := tx.Where("course_id = ?", courseID).Delete(&model.ChallengeAttempt{}).Error; err != nil {
			return fmt.Errorf("delete challenge attempts: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.AssessmentChallenge{}).Error; err != nil {
			return fmt.Errorf("delete assessment challenges: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.MisconceptionEvent{}).Error; err != nil {
			return fmt.Errorf("delete misconception events: %w", err)
		}
		if len(misconceptionIDs) > 0 {
			if err := tx.Where("misconception_id IN ?", misconceptionIDs).Delete(&model.MisconceptionPatternLink{}).Error; err != nil {
				return fmt.Errorf("delete misconception pattern links: %w", err)
			}
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.Misconception{}).Error; err != nil {
			return fmt.Errorf("delete misconceptions: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.CognitiveStateEvent{}).Error; err != nil {
			return fmt.Errorf("delete cognitive state events: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.CognitiveEvidence{}).Error; err != nil {
			return fmt.Errorf("delete cognitive evidence: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.CognitiveState{}).Error; err != nil {
			return fmt.Errorf("delete cognitive states: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.MasteryRecord{}).Error; err != nil {
			return fmt.Errorf("delete mastery records: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.AIEvaluationRun{}).Error; err != nil {
			return fmt.Errorf("delete AI evaluation runs: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.LearningTurn{}).Error; err != nil {
			return fmt.Errorf("delete learning turns: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.LessonRelation{}).Error; err != nil {
			return fmt.Errorf("delete lesson relations: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.Lesson{}).Error; err != nil {
			return fmt.Errorf("delete lessons: %w", err)
		}
		if err := tx.Where("course_id = ?", courseID).Delete(&model.CourseUnit{}).Error; err != nil {
			return fmt.Errorf("delete course units: %w", err)
		}
		if err := tx.Delete(&course).Error; err != nil {
			return fmt.Errorf("delete course: %w", err)
		}
		return nil
	})
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
	logicCourseName        = "逻辑与科学思维"
	logicUnitTitle         = "日常推理基础"
	psychologyCourseName   = "心理学"
	psychologyUnitTitle    = "认知与行为基础"
)

var starterLessonDefinitions = []starterLessonDefinition{
	{
		UnitTitle: starterUnitATitle, Title: "水在人体中的基本作用",
		CoreQuestion:          "为什么人体不能简单把水理解成“解渴用的饮料”？水在身体里主要承担哪些作用？",
		ExpectedUnderstanding: "水不仅用于缓解口渴，还参与体温调节、体液与血液运输、代谢反应、营养物质和代谢产物运输，以及维持细胞和组织正常环境。应把水理解为人体内部环境的重要组成部分，而不是单一饮用品。",
		SortOrder:             1, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleFoundation, DepthLevel: 1,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: starterUnitATitle, Title: "体液平衡是如何维持的",
		CoreQuestion:          "人体每天都在摄入和排出水分，为什么体内水分通常还能维持在相对稳定的范围？",
		ExpectedUnderstanding: "人体通过摄入、尿液、汗液、呼吸等途径持续发生水分交换，并通过口渴、肾脏调节和相关激素机制维持动态平衡。体液平衡不是静止状态，而是持续调节。",
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
		CoreQuestion:          "日常生活中，能不能用一个固定饮水量适用于所有人？判断自己是否需要补水时应该考虑哪些因素？",
		ExpectedUnderstanding: "饮水需求受体型、饮食、环境温湿度、活动量、出汗和身体状态等多因素影响，不存在适用于所有人所有场景的唯一固定量。判断应综合口渴、实际摄入、排尿、活动和环境等信息。",
		SortOrder:             2, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleCore, DepthLevel: 2,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: starterUnitCTitle, Title: "高温和运动为什么会改变补水需求",
		CoreQuestion:          "一个人在炎热天气运动后，即使和平时喝了同样多的水，为什么仍可能出现水分不足？",
		ExpectedUnderstanding: "高温和运动会提高体温调节需求并增加出汗，使水分损失增加，因此日常静息状态下合适的饮水量可能不足，需要结合环境、强度和出汗动态调整。",
		SortOrder:             1, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleApplication, DepthLevel: 2,
		AssessmentTargetLevel: model.AssessmentTargetApply,
	},
	{
		UnitTitle: starterUnitCTitle, Title: "大量出汗后为什么需要关注电解质",
		CoreQuestion:          "大量出汗以后为什么有时只补大量白水并不是最完整的补水思路？",
		ExpectedUnderstanding: "汗液不仅带走水，也会带走钠等电解质。在长时间、大量出汗场景中，仅补水可能不足以恢复水和电解质平衡；但普通日常活动并不意味着必须额外补电解质，应结合实际出汗程度和持续时间判断。",
		SortOrder:             2, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleApplication, DepthLevel: 3,
		AssessmentTargetLevel: model.AssessmentTargetApply,
	},
	{
		UnitTitle: starterUnitCTitle, Title: "年龄与疾病状态为什么会影响口渴信号",
		CoreQuestion:          "为什么同样处于水分不足状态，不同年龄或不同身体状态的人可能表现出不同程度的口渴？",
		ExpectedUnderstanding: "口渴感受和水分调节会受到年龄、身体状态以及部分疾病或药物等因素影响，因此其敏感程度并非所有人都相同。重点是理解口渴有价值但存在个体和情境边界。",
		SortOrder:             3, Status: model.LessonStatusPending, IsCore: false, ContentRole: model.ContentRoleExtension, DepthLevel: 3,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: starterUnitCTitle, Title: "如何综合判断不同场景下的补水策略",
		CoreQuestion:          "如果一个人既不明显口渴，又处在高温、运动或长时间低饮水等特殊场景中，应该怎样综合判断是否需要补水？",
		ExpectedUnderstanding: "应综合环境、活动量、出汗、实际饮水、饮食、口渴、排尿和身体状态等多种信息，避免依赖单一指标，也避免从“不能只看口渴”走向“必须机械大量喝水”的另一个极端。",
		SortOrder:             4, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleApplication, DepthLevel: 4,
		AssessmentTargetLevel: model.AssessmentTargetApply,
	},
}

var logicLessonDefinitions = []starterLessonDefinition{
	{
		UnitTitle: logicUnitTitle, Title: "单因素解释的陷阱",
		CoreQuestion:          "当一个现象发生时，为什么“找到一个可能原因”并不等于“已经解释了这个现象”？",
		ExpectedUnderstanding: "现实问题通常可能受多个变量共同影响，一个因素与结果有关并不意味着它能单独解释全部结果。可靠判断需要考虑其他因素、条件和替代解释，避免把复杂问题简化成单一原因。",
		SortOrder:             1, Status: model.LessonStatusLearning, IsCore: true, ContentRole: model.ContentRoleFoundation, DepthLevel: 1,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: logicUnitTitle, Title: "相关不等于因果",
		CoreQuestion:          "如果两件事情经常同时出现，为什么不能直接认为其中一件导致了另一件？",
		ExpectedUnderstanding: "相关只能说明变量共同变化，不能自动证明因果。可能存在反向因果、共同原因、选择偏差或偶然关系，因果判断需要额外证据。",
		SortOrder:             2, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleCore, DepthLevel: 2,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: logicUnitTitle, Title: "如何判断一条证据有多可靠",
		CoreQuestion:          "面对“有人亲身试过有效”和“多个较高质量研究得到一致结果”两类信息时，应该如何比较它们的可信度？",
		ExpectedUnderstanding: "证据可靠性与样本数量、研究设计、偏差控制、可重复性、来源独立性和证据一致性有关。个人经验可提供线索，但更容易受偶然性和认知偏差影响。",
		SortOrder:             3, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleApplication, DepthLevel: 2,
		AssessmentTargetLevel: model.AssessmentTargetApply,
	},
}

var psychologyLessonDefinitions = []starterLessonDefinition{
	{
		UnitTitle: psychologyUnitTitle, Title: "确认偏误",
		CoreQuestion:          "为什么人在已经相信一个观点以后，往往更容易注意到支持它的信息，而忽略反对它的信息？",
		ExpectedUnderstanding: "确认偏误指人更容易寻找、注意、解释和记住支持既有信念的信息，同时低估冲突证据。它是常见认知倾向，并不意味着人是在故意欺骗自己。",
		SortOrder:             1, Status: model.LessonStatusLearning, IsCore: true, ContentRole: model.ContentRoleFoundation, DepthLevel: 1,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: psychologyUnitTitle, Title: "情绪如何影响判断",
		CoreQuestion:          "为什么人在焦虑、愤怒或兴奋时，面对同一个问题可能做出和平静时不同的判断？",
		ExpectedUnderstanding: "情绪会影响注意、风险感知、信息解释和决策权重，因此同一信息在不同情绪状态下可能被赋予不同意义。情绪包含信息，但不能自动等同于客观事实。",
		SortOrder:             2, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleCore, DepthLevel: 2,
		AssessmentTargetLevel: model.AssessmentTargetUnderstand,
	},
	{
		UnitTitle: psychologyUnitTitle, Title: "习惯为什么会自动发生",
		CoreQuestion:          "为什么有些行为明明没有经过认真决定，却会在固定时间或场景中自动发生？",
		ExpectedUnderstanding: "重复行为会逐渐与环境线索、时间、情绪或上下文形成稳定联系，使行为启动越来越依赖线索。改变习惯不仅依靠意志力，也可通过调整线索、环境和替代行为实现。",
		SortOrder:             3, Status: model.LessonStatusPending, IsCore: true, ContentRole: model.ContentRoleApplication, DepthLevel: 2,
		AssessmentTargetLevel: model.AssessmentTargetApply,
	},
}

var logicRelationDefinitions = []starterRelationDefinition{
	{FromTitle: "单因素解释的陷阱", ToTitle: "相关不等于因果", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: "单因素解释的陷阱", ToTitle: "如何判断一条证据有多可靠", RelationType: model.LessonRelationPrerequisite},
	{FromTitle: "相关不等于因果", ToTitle: "如何判断一条证据有多可靠", RelationType: model.LessonRelationRelated},
}

var psychologyRelationDefinitions = []starterRelationDefinition{
	{FromTitle: "确认偏误", ToTitle: "情绪如何影响判断", RelationType: model.LessonRelationRelated},
	{FromTitle: "情绪如何影响判断", ToTitle: "习惯为什么会自动发生", RelationType: model.LessonRelationRelated},
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
		courseCreated := false
		err := tx.Where("name = ?", starterCourseName).First(&course).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find starter course: %w", err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			courseCreated = true
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
			if err := ensureStarterRelation(tx, course.ID, from.ID, to.ID, definition.RelationType); err != nil {
				return err
			}
		}

		if courseCreated || course.CurrentUnitID == nil || course.CurrentLessonID == nil {
			now := time.Now()
			updates := map[string]interface{}{
				"current_unit":      units[starterUnitBTitle].Title,
				"current_unit_id":   units[starterUnitBTitle].ID,
				"current_lesson_id": lessons[starterExistingLesson].ID,
				"last_studied_at":   now,
			}
			if courseCreated {
				updates["status"] = model.CourseStatusLearning
			}
			if err := tx.Model(&course).Updates(updates).Error; err != nil {
				return fmt.Errorf("update starter course: %w", err)
			}
		}

		if err := seedTestCourse(tx, model.Course{
			Name:        logicCourseName,
			Description: "帮助识别常见推理错误、理解证据与因果关系，并建立更可靠的日常判断框架。",
			Goal:        "建立识别推理错误、比较证据可靠性和分析因果关系的基础能力。",
			Status:      model.CourseStatusInitializing,
		}, logicUnitTitle, logicLessonDefinitions, logicRelationDefinitions); err != nil {
			return err
		}
		if err := seedTestCourse(tx, model.Course{
			Name:        psychologyCourseName,
			Description: "从认知、判断与行为角度理解人的心理过程，以及这些过程如何影响日常选择。",
			Goal:        "理解认知、情绪和环境如何共同影响判断与行为。",
			Status:      model.CourseStatusInitializing,
		}, psychologyUnitTitle, psychologyLessonDefinitions, psychologyRelationDefinitions); err != nil {
			return err
		}
		lessonIDs := make(map[string]uint, len(lessons))
		for title, lesson := range lessons {
			lessonIDs[title] = lesson.ID
		}
		if err := seedNutritionBlueprint(tx, course.ID, lessonIDs); err != nil {
			return err
		}
		return nil
	})
}

func seedTestCourse(tx *gorm.DB, definition model.Course, unitTitle string, lessonDefinitions []starterLessonDefinition, relationDefinitions []starterRelationDefinition) error {
	course, created, err := ensureSeedCourse(tx, definition)
	if err != nil {
		return err
	}

	unitDefinition := starterUnitDefinition{
		Title:     unitTitle,
		Objective: definition.Goal,
		SortOrder: 1,
		Status:    model.CourseUnitStatusLearning,
	}
	unit, err := ensureStarterUnit(tx, course.ID, unitDefinition, false)
	if err != nil {
		return err
	}

	lessons := make(map[string]model.Lesson, len(lessonDefinitions))
	for _, lessonDefinition := range lessonDefinitions {
		lesson, err := ensureStarterLesson(tx, course.ID, unit.ID, lessonDefinition)
		if err != nil {
			return err
		}
		lessons[lessonDefinition.Title] = lesson
	}
	for _, relationDefinition := range relationDefinitions {
		from := lessons[relationDefinition.FromTitle]
		to := lessons[relationDefinition.ToTitle]
		if err := ensureStarterRelation(tx, course.ID, from.ID, to.ID, relationDefinition.RelationType); err != nil {
			return err
		}
	}

	if created || course.CurrentUnitID == nil || course.CurrentLessonID == nil || course.Status == model.CourseStatusInitializing {
		firstLesson := lessonDefinitions[0]
		first := lessons[firstLesson.Title]
		updates := map[string]interface{}{
			"current_unit":      unit.Title,
			"current_unit_id":   unit.ID,
			"current_lesson_id": first.ID,
		}
		if created {
			updates["progress"] = definition.Progress
		}
		updates["status"] = model.CourseStatusLearning
		if err := tx.Model(&course).Updates(updates).Error; err != nil {
			return fmt.Errorf("update seeded course %q: %w", course.Name, err)
		}
	}
	return nil
}

func ensureSeedCourse(tx *gorm.DB, definition model.Course) (model.Course, bool, error) {
	var course model.Course
	err := tx.Where("name = ?", definition.Name).First(&course).Error
	if err == nil {
		if err := tx.Model(&course).Updates(map[string]interface{}{
			"description": definition.Description,
			"goal":        definition.Goal,
		}).Error; err != nil {
			return model.Course{}, false, fmt.Errorf("update seeded course %q: %w", definition.Name, err)
		}
		course.Description = definition.Description
		course.Goal = definition.Goal
		return course, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Course{}, false, fmt.Errorf("find seeded course %q: %w", definition.Name, err)
	}
	if err := tx.Create(&definition).Error; err != nil {
		return model.Course{}, false, fmt.Errorf("create seeded course %q: %w", definition.Name, err)
	}
	return definition, true, nil
}

func ensureStarterRelation(tx *gorm.DB, courseID, fromLessonID, toLessonID uint, relationType model.LessonRelationType) error {
	relation := model.LessonRelation{
		CourseID:     courseID,
		FromLessonID: fromLessonID,
		ToLessonID:   toLessonID,
		RelationType: relationType,
	}
	var existing model.LessonRelation
	err := tx.Where("course_id = ? AND from_lesson_id = ? AND to_lesson_id = ? AND relation_type = ?", relation.CourseID, relation.FromLessonID, relation.ToLessonID, relation.RelationType).First(&existing).Error
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := tx.Create(&relation).Error; err != nil {
			return fmt.Errorf("create starter lesson relation: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("find starter lesson relation: %w", err)
	}
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
		return unit, nil
	}
	if err := tx.Model(&unit).Updates(map[string]interface{}{
		"title":      definition.Title,
		"objective":  definition.Objective,
		"sort_order": definition.SortOrder,
	}).Error; err != nil {
		return model.CourseUnit{}, fmt.Errorf("update starter unit %q: %w", definition.Title, err)
	}
	unit.Title = definition.Title
	unit.Objective = definition.Objective
	unit.SortOrder = definition.SortOrder
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
	if lesson.Status == "" {
		if err := tx.Model(&lesson).Update("status", definition.Status).Error; err != nil {
			return model.Lesson{}, fmt.Errorf("repair starter lesson status %q: %w", definition.Title, err)
		}
		lesson.Status = definition.Status
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
