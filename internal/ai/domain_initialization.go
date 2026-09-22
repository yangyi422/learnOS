package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"learnos/internal/model"
)

type DomainSkeletonRequest struct {
	DomainName   string
	LearningGoal string
	TargetDepth  string
}

type DomainSkeletonResult struct {
	Course                 DomainCourseMetadata    `json:"course"`
	Blueprint              DomainBlueprintMetadata `json:"blueprint"`
	RecommendedStarterKeys []string                `json:"recommended_starter_unit_keys"`
}

type DomainCourseMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DomainBlueprintMetadata struct {
	Name         string       `json:"name"`
	Domain       string       `json:"domain"`
	LearningGoal string       `json:"learning_goal"`
	TargetDepth  string       `json:"target_depth"`
	Units        []DomainUnit `json:"units"`
}

type DomainUnit struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Importance  string `json:"importance"`
}

type DomainStarterRequest struct {
	DomainName          string
	LearningGoal        string
	TargetDepth         string
	Skeleton            DomainSkeletonResult
	SelectedStarterKeys []string
}

type DomainStarterResult struct {
	ExpandedUnits []ExpandedDomainUnit   `json:"expanded_units"`
	Relations     []DomainLessonRelation `json:"relations"`
}

type ExpandedDomainUnit struct {
	Key     string                  `json:"key"`
	Lessons []DomainBlueprintLesson `json:"lessons"`
}

type DomainBlueprintLesson struct {
	Key                   string `json:"key"`
	Title                 string `json:"title"`
	Summary               string `json:"summary"`
	Importance            string `json:"importance"`
	ContentRole           string `json:"content_role"`
	DepthLevel            int    `json:"depth_level"`
	AssessmentTargetLevel string `json:"assessment_target_level"`
}

type DomainLessonRelation struct {
	FromLessonKey string `json:"from_lesson_key"`
	ToLessonKey   string `json:"to_lesson_key"`
	RelationType  string `json:"relation_type"`
}

type InitialWorldRequest struct {
	DomainName   string
	LearningGoal string
	TargetDepth  string
	Starter      DomainStarterResult
}

type InitialWorldResult struct {
	InitialLessons      []InitialWorldLesson   `json:"initial_lessons"`
	Relations           []DomainLessonRelation `json:"relations"`
	RecommendedFirstKey string                 `json:"recommended_first_lesson_key"`
}

type BlueprintUnitExpansionRequest struct {
	DomainName         string
	LearningGoal       string
	TargetDepth        string
	TargetUnit         DomainUnit
	AdjacentUnits      []DomainUnit
	ExistingLessonKeys []string
}

type BlueprintUnitExpansionResult struct {
	Lessons   []DomainBlueprintLesson `json:"lessons"`
	Relations []DomainLessonRelation  `json:"relations"`
}

type InitialWorldLesson struct {
	BlueprintLessonKey    string `json:"blueprint_lesson_key"`
	Title                 string `json:"title"`
	CoreQuestion          string `json:"core_question"`
	ExpectedUnderstanding string `json:"expected_understanding"`
	ContentRole           string `json:"content_role"`
	DepthLevel            int    `json:"depth_level"`
	AssessmentTargetLevel string `json:"assessment_target_level"`
	IsCore                bool   `json:"is_core"`
}

func BuildDomainSkeletonSystemPrompt() string {
	return `你是 LearnOS 的领域地图编辑器。只生成一个学习领域的粗粒度骨架：课程元数据和 6 到 10 个 Unit。

只输出一个 JSON object，不要 Markdown、解释文字或代码围栏。JSON 只能包含以下字段，不能添加其它字段：
{
  "course": {"name": "课程名", "description": "课程描述"},
  "blueprint": {
    "name": "领域地图名",
    "domain": "领域名",
    "learning_goal": "学习目标",
    "target_depth": "期望深度",
    "units": [{"key": "稳定且唯一的英文 key", "title": "Unit 标题", "description": "Unit 描述", "importance": "core|recommended|optional"}]
  },
  "recommended_starter_unit_keys": ["units 中已有的 key，1 到 2 个"]
}
units 必须有 6 到 10 个；不要生成 Lesson、CoreQuestion、ExpectedUnderstanding、关系、来源、用户状态或正式课程节点。`
}

func BuildDomainSkeletonUserPrompt(req DomainSkeletonRequest) string {
	return fmt.Sprintf("领域：%s\n学习目标：%s\n期望深度：%s\n请返回 course、blueprint、recommended_starter_unit_keys。blueprint.units 必须为 6 到 10 个。", req.DomainName, req.LearningGoal, req.TargetDepth)
}

func BuildDomainStarterSystemPrompt() string {
	return `你是 LearnOS 的起步蓝图编辑器。只展开用户选择的 1 到 2 个 Unit，生成总计 5 到 10 个 Blueprint Lesson 和必要关系。

只输出一个 JSON object，不要 Markdown、解释文字或代码围栏。JSON 只能包含以下字段，不能添加其它字段：
{
  "expanded_units": [{
    "key": "已选择的 Unit key",
    "lessons": [{
      "key": "稳定且唯一的 lesson key",
      "title": "Lesson 标题",
      "summary": "Lesson 摘要",
      "importance": "core|recommended|optional",
      "content_role": "foundation|core|deepening|application|extension",
      "depth_level": 1,
      "assessment_target_level": "recognize|understand|apply|transfer"
    }]
  }],
  "relations": [{"from_lesson_key": "已有 lesson key", "to_lesson_key": "已有 lesson key", "relation_type": "prerequisite|extends|application|related"}]
}
只展开用户选择的 Unit；expanded_units 必须有 1 到 2 个，所有 lessons 合计 5 到 10 个。不要生成正式 Lesson、CoreQuestion、ExpectedUnderstanding、来源、用户状态或完整领域蓝图。`
}

func BuildDomainStarterUserPrompt(req DomainStarterRequest) string {
	skeleton, _ := json.Marshal(req.Skeleton)
	selected, _ := json.Marshal(req.SelectedStarterKeys)
	return fmt.Sprintf("领域：%s\n学习目标：%s\n期望深度：%s\n领域骨架：%s\n选中的起步 Unit keys：%s\n请返回 expanded_units 和 relations。", req.DomainName, req.LearningGoal, req.TargetDepth, skeleton, selected)
}

func BuildInitialWorldSystemPrompt() string {
	return `你是 LearnOS 的初始知识世界编辑器。根据已展开的起步蓝图，生成 3 到 5 个最适合首次学习的正式 Lesson 草案。

只输出一个 JSON object，不要 Markdown、解释文字或代码围栏。JSON 只能包含以下字段，不能添加其它字段：
{
  "initial_lessons": [{
    "blueprint_lesson_key": "已展开蓝图中的 lesson key",
    "title": "Lesson 标题",
    "core_question": "核心问题",
    "expected_understanding": "完成后应理解什么",
    "content_role": "foundation|core|deepening|application|extension",
    "depth_level": 1,
    "assessment_target_level": "recognize|understand|apply|transfer",
    "is_core": true
  }],
  "relations": [{"from_lesson_key": "initial_lessons 中已有 key", "to_lesson_key": "initial_lessons 中已有 key", "relation_type": "prerequisite|extends|application|related"}],
  "recommended_first_lesson_key": "initial_lessons 中已有的 blueprint_lesson_key"
}
initial_lessons 必须有 3 到 5 个；每个 Lesson 都必须有 core_question 和 expected_understanding。不要创建 Course 或用户认知记录。`
}

func BuildInitialWorldUserPrompt(req InitialWorldRequest) string {
	starter, _ := json.Marshal(req.Starter)
	return fmt.Sprintf("领域：%s\n学习目标：%s\n期望深度：%s\n已展开起步蓝图：%s\n请返回 initial_lessons、relations、recommended_first_lesson_key。", req.DomainName, req.LearningGoal, req.TargetDepth, starter)
}

func BuildBlueprintUnitExpansionSystemPrompt() string {
	return `你是 LearnOS 的知识区域展开编辑器。只展开指定的一个 Blueprint Unit，生成 5 到 10 个 Blueprint Lesson 和它们之间必要的结构关系。

只输出一个 JSON object，不要 Markdown、解释文字或代码围栏。JSON 只能包含以下字段，不能添加其它字段：
{
  "lessons": [{
    "key": "稳定且唯一的英文 key",
    "title": "Lesson 标题",
    "summary": "Lesson 摘要",
    "importance": "core|recommended|optional",
    "content_role": "foundation|core|deepening|application|extension",
    "depth_level": 1,
    "assessment_target_level": "recognize|understand|apply|transfer"
  }],
  "relations": [{"from_lesson_key": "Lesson key", "to_lesson_key": "Lesson key", "relation_type": "prerequisite|extends|application|related"}]
}
只生成目标 Unit 的 Blueprint Lesson；不要生成正式 Lesson、CoreQuestion、ExpectedUnderstanding、来源、用户状态或认知记录。lessons 必须有 5 到 10 个，key 不能和已有 Blueprint Lesson 重复。`
}

func BuildBlueprintUnitExpansionUserPrompt(req BlueprintUnitExpansionRequest) string {
	target, _ := json.Marshal(req.TargetUnit)
	adjacent, _ := json.Marshal(req.AdjacentUnits)
	existing, _ := json.Marshal(req.ExistingLessonKeys)
	return fmt.Sprintf("领域：%s\n学习目标：%s\n期望深度：%s\n目标 Unit：%s\n相邻 Unit 摘要：%s\n已有 Lesson keys：%s\n请只返回 lessons 和 relations。", req.DomainName, req.LearningGoal, req.TargetDepth, target, adjacent, existing)
}

func ParseAndValidateDomainSkeleton(content string) (DomainSkeletonResult, error) {
	var result DomainSkeletonResult
	if err := decodeStrict(content, &result); err != nil {
		return DomainSkeletonResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("decode domain skeleton: %w", err))
	}
	if strings.TrimSpace(result.Course.Name) == "" || strings.TrimSpace(result.Blueprint.Name) == "" || strings.TrimSpace(result.Blueprint.Domain) == "" || len(result.Blueprint.Units) < 6 || len(result.Blueprint.Units) > 10 {
		return DomainSkeletonResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("domain skeleton must contain 6 to 10 units"))
	}
	seen := map[string]bool{}
	for _, unit := range result.Blueprint.Units {
		if strings.TrimSpace(unit.Key) == "" || strings.TrimSpace(unit.Title) == "" || seen[unit.Key] {
			return DomainSkeletonResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("invalid or duplicate skeleton unit"))
		}
		seen[unit.Key] = true
		if unit.Importance != "core" && unit.Importance != "recommended" && unit.Importance != "optional" {
			return DomainSkeletonResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("invalid skeleton importance"))
		}
	}
	if len(result.RecommendedStarterKeys) < 1 || len(result.RecommendedStarterKeys) > 2 {
		return DomainSkeletonResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("recommended starter units must be 1 to 2"))
	}
	for _, key := range result.RecommendedStarterKeys {
		if !seen[key] {
			return DomainSkeletonResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("recommended starter unit is not in skeleton"))
		}
	}
	return result, nil
}

func ParseAndValidateDomainStarter(content string, skeleton DomainSkeletonResult, selected []string) (DomainStarterResult, error) {
	var result DomainStarterResult
	if err := decodeStrict(content, &result); err != nil {
		return DomainStarterResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("decode starter blueprint: %w", err))
	}
	if len(result.ExpandedUnits) < 1 || len(result.ExpandedUnits) > 2 {
		return DomainStarterResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("starter expansion must contain 1 to 2 units"))
	}
	validUnits := map[string]bool{}
	for _, unit := range skeleton.Blueprint.Units {
		validUnits[unit.Key] = true
	}
	selectedSet := map[string]bool{}
	for _, key := range selected {
		if selectedSet[key] {
			return DomainStarterResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("duplicate selected starter unit"))
		}
		selectedSet[key] = true
	}
	lessons := map[string]bool{}
	expandedUnits := map[string]bool{}
	total := 0
	for _, unit := range result.ExpandedUnits {
		if expandedUnits[unit.Key] || !validUnits[unit.Key] || (len(selectedSet) > 0 && !selectedSet[unit.Key]) || len(unit.Lessons) == 0 {
			return DomainStarterResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("invalid expanded starter unit"))
		}
		expandedUnits[unit.Key] = true
		for _, lesson := range unit.Lessons {
			if lesson.Key == "" || lesson.Title == "" || lessons[lesson.Key] {
				return DomainStarterResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("invalid or duplicate starter lesson"))
			}
			lessons[lesson.Key] = true
			total++
			if !validContentRole(lesson.ContentRole) || !validAssessmentLevel(lesson.AssessmentTargetLevel) {
				return DomainStarterResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("invalid starter lesson metadata"))
			}
		}
	}
	if total < 5 || total > 10 {
		return DomainStarterResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("starter expansion must contain 5 to 10 lessons"))
	}
	if err := validateDomainRelations(result.Relations, lessons); err != nil {
		return DomainStarterResult{}, err
	}
	return result, nil
}

func ParseAndValidateInitialWorld(content string, starter DomainStarterResult) (InitialWorldResult, error) {
	var result InitialWorldResult
	if err := decodeStrict(content, &result); err != nil {
		return InitialWorldResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("decode initial world: %w", err))
	}
	if len(result.InitialLessons) < 3 || len(result.InitialLessons) > 5 {
		return InitialWorldResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("initial world must contain 3 to 5 lessons"))
	}
	valid := map[string]bool{}
	for _, unit := range starter.ExpandedUnits {
		for _, lesson := range unit.Lessons {
			valid[lesson.Key] = true
		}
	}
	lessons := map[string]bool{}
	for _, lesson := range result.InitialLessons {
		if !valid[lesson.BlueprintLessonKey] || lesson.Title == "" || lesson.CoreQuestion == "" || lesson.ExpectedUnderstanding == "" || lessons[lesson.BlueprintLessonKey] || !validContentRole(lesson.ContentRole) || !validAssessmentLevel(lesson.AssessmentTargetLevel) {
			return InitialWorldResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("invalid initial world lesson"))
		}
		lessons[lesson.BlueprintLessonKey] = true
	}
	if !lessons[result.RecommendedFirstKey] {
		return InitialWorldResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("invalid recommended first lesson"))
	}
	if err := validateDomainRelations(result.Relations, lessons); err != nil {
		return InitialWorldResult{}, err
	}
	return result, nil
}

func ParseAndValidateBlueprintUnitExpansion(content string, existingLessonKeys []string) (BlueprintUnitExpansionResult, error) {
	var result BlueprintUnitExpansionResult
	if err := decodeStrict(content, &result); err != nil {
		return BlueprintUnitExpansionResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("decode blueprint unit expansion: %w", err))
	}
	if len(result.Lessons) < 5 || len(result.Lessons) > 10 {
		return BlueprintUnitExpansionResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("blueprint unit expansion must contain 5 to 10 lessons"))
	}
	seen := map[string]bool{}
	for _, key := range existingLessonKeys {
		seen[key] = true
	}
	for _, lesson := range result.Lessons {
		if strings.TrimSpace(lesson.Key) == "" || strings.TrimSpace(lesson.Title) == "" || seen[lesson.Key] || !validContentRole(lesson.ContentRole) || !validAssessmentLevel(lesson.AssessmentTargetLevel) {
			return BlueprintUnitExpansionResult{}, newProviderError(ErrInvalidResponse, fmt.Errorf("invalid or duplicate blueprint unit lesson"))
		}
		seen[lesson.Key] = true
	}
	if err := validateDomainRelations(result.Relations, seen); err != nil {
		return BlueprintUnitExpansionResult{}, err
	}
	return result, nil
}

func decodeStrict(content string, value interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(normalizeJSONEnvelope(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON")
		}
		return err
	}
	return nil
}

// normalizeJSONEnvelope removes only the presentation wrapper occasionally
// added by chat models. The JSON is still decoded with DisallowUnknownFields
// and trailing-content checks below, so this does not relax the schema.
func normalizeJSONEnvelope(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	newline := strings.IndexByte(trimmed, '\n')
	if newline < 0 {
		return trimmed
	}
	trimmed = strings.TrimSpace(trimmed[newline+1:])
	if strings.HasSuffix(trimmed, "```") {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, "```"))
	}
	return trimmed
}
func validContentRole(value string) bool {
	return model.ContentRole(value).Valid()
}
func validAssessmentLevel(value string) bool {
	return value == "recognize" || value == "understand" || value == "apply" || value == "transfer"
}
func validateDomainRelations(relations []DomainLessonRelation, lessons map[string]bool) error {
	for _, relation := range relations {
		if !lessons[relation.FromLessonKey] || !lessons[relation.ToLessonKey] || relation.FromLessonKey == relation.ToLessonKey || !validRelationType(relation.RelationType) {
			return newProviderError(ErrInvalidResponse, fmt.Errorf("invalid domain lesson relation"))
		}
	}
	return nil
}
func validRelationType(value string) bool {
	return value == "prerequisite" || value == "extends" || value == "application" || value == "related"
}
