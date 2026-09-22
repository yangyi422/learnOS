package repository

import (
	"errors"
	"fmt"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type nutritionBlueprintUnitDefinition struct {
	Key, Title, Description, Importance string
	SortOrder                           int
}

type nutritionBlueprintLessonDefinition struct {
	UnitKey, Key, Title, Summary, Importance string
	ContentRole                              model.ContentRole
	DepthLevel                               int
	AssessmentTargetLevel                    string
	SortOrder                                int
}

type nutritionBlueprintRelationDefinition struct{ From, To, RelationType string }

var nutritionBlueprintUnits = []nutritionBlueprintUnitDefinition{
	{Key: "nutrition.foundation", Title: "营养学基础与能量", Description: "建立能量、营养密度和消化吸收的基础概念。", Importance: model.CurriculumImportanceCore, SortOrder: 1},
	{Key: "nutrition.carb", Title: "碳水化合物与膳食纤维", Description: "理解碳水化合物来源、作用与膳食纤维。", Importance: model.CurriculumImportanceCore, SortOrder: 2},
	{Key: "nutrition.protein", Title: "蛋白质", Description: "理解蛋白质的作用、来源质量和摄入分布。", Importance: model.CurriculumImportanceCore, SortOrder: 3},
	{Key: "nutrition.fat", Title: "脂肪", Description: "理解脂肪的必要性、类型和日常判断。", Importance: model.CurriculumImportanceCore, SortOrder: 4},
	{Key: "nutrition.vitamin", Title: "维生素", Description: "理解维生素的作用、分类和食物来源。", Importance: model.CurriculumImportanceCore, SortOrder: 5},
	{Key: "nutrition.mineral", Title: "矿物质与电解质", Description: "理解矿物质、电解质以及钠钾等常见关系。", Importance: model.CurriculumImportanceCore, SortOrder: 6},
	{Key: "nutrition.hydration", Title: "水与体液平衡", Description: "理解水、体液、口渴和不同场景下的补水判断。", Importance: model.CurriculumImportanceCore, SortOrder: 7},
	{Key: "nutrition.label", Title: "食品标签与饮食判断", Description: "学会结合配料表和营养成分表判断食品。", Importance: model.CurriculumImportanceCore, SortOrder: 8},
	{Key: "nutrition.meal", Title: "膳食结构与实际应用", Description: "把营养知识用于一餐和长期饮食模式的优化。", Importance: model.CurriculumImportanceCore, SortOrder: 9},
}

var nutritionBlueprintLessons = []nutritionBlueprintLessonDefinition{
	{UnitKey: "nutrition.foundation", Key: "nutrition.foundation.energy_balance", Title: "能量摄入与消耗为什么需要平衡", Summary: "理解能量摄入、消耗和长期平衡。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 1},
	{UnitKey: "nutrition.foundation", Key: "nutrition.foundation.nutrient_density", Title: "热量充足为什么不等于营养均衡", Summary: "区分热量充足与营养密度、营养均衡。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleCore, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 2},
	{UnitKey: "nutrition.foundation", Key: "nutrition.foundation.digestion_absorption", Title: "食物中的营养素如何被消化和吸收", Summary: "了解食物营养素进入身体的基本过程。", Importance: model.CurriculumImportanceRecommended, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 3},
	{UnitKey: "nutrition.carb", Key: "nutrition.carb.function", Title: "碳水化合物在身体中的主要作用", Summary: "理解碳水化合物的供能和其他基本作用。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 1},
	{UnitKey: "nutrition.carb", Key: "nutrition.carb.quality", Title: "不同碳水来源为什么对身体影响不同", Summary: "比较不同碳水来源及其情境差异。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 2},
	{UnitKey: "nutrition.carb", Key: "nutrition.carb.fiber", Title: "膳食纤维为什么重要", Summary: "理解膳食纤维的作用和常见来源。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 3},
	{UnitKey: "nutrition.protein", Key: "nutrition.protein.function", Title: "蛋白质为什么重要", Summary: "理解蛋白质在组织、酶和调节中的作用。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 1},
	{UnitKey: "nutrition.protein", Key: "nutrition.protein.quality", Title: "蛋白质来源和质量有什么区别", Summary: "理解蛋白质来源、氨基酸组成和质量差异。", Importance: model.CurriculumImportanceRecommended, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 2},
	{UnitKey: "nutrition.protein", Key: "nutrition.protein.distribution", Title: "为什么蛋白质摄入不仅看一天总量", Summary: "理解蛋白质摄入分布和实际饮食情境。", Importance: model.CurriculumImportanceRecommended, ContentRole: model.ContentRoleApplication, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 3},
	{UnitKey: "nutrition.fat", Key: "nutrition.fat.function", Title: "脂肪为什么是必需营养素", Summary: "理解脂肪的供能、结构和吸收相关作用。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 1},
	{UnitKey: "nutrition.fat", Key: "nutrition.fat.types", Title: "不同类型脂肪为什么不能一概而论", Summary: "区分不同脂肪类型及其日常判断边界。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 2},
	{UnitKey: "nutrition.fat", Key: "nutrition.fat.saturated", Title: "为什么需要关注饱和脂肪", Summary: "理解关注饱和脂肪时不能脱离整体饮食。", Importance: model.CurriculumImportanceRecommended, ContentRole: model.ContentRoleApplication, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 3},
	{UnitKey: "nutrition.vitamin", Key: "nutrition.vitamin.role", Title: "维生素为什么需要少量却不可缺少", Summary: "理解维生素的调节作用和缺乏风险。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 1},
	{UnitKey: "nutrition.vitamin", Key: "nutrition.vitamin.solubility", Title: "脂溶性和水溶性维生素有什么区别", Summary: "比较两类维生素的基本特点。", Importance: model.CurriculumImportanceRecommended, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 2},
	{UnitKey: "nutrition.vitamin", Key: "nutrition.vitamin.food_first", Title: "为什么一般优先从食物获得维生素", Summary: "理解食物优先和补充剂使用的基本边界。", Importance: model.CurriculumImportanceRecommended, ContentRole: model.ContentRoleApplication, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 3},
	{UnitKey: "nutrition.mineral", Key: "nutrition.mineral.role", Title: "矿物质在人体中主要做什么", Summary: "理解矿物质参与结构、调节和代谢的作用。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 1},
	{UnitKey: "nutrition.mineral", Key: "nutrition.mineral.sodium_potassium", Title: "为什么需要同时理解钠和钾", Summary: "理解钠钾在体液和电解质判断中的关系。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 2},
	{UnitKey: "nutrition.mineral", Key: "nutrition.mineral.calcium_iron", Title: "为什么钙和铁的营养问题不能只看单次化验", Summary: "理解营养判断需要结合长期饮食和多种信息。", Importance: model.CurriculumImportanceRecommended, ContentRole: model.ContentRoleApplication, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 3},
	{UnitKey: "nutrition.hydration", Key: "nutrition.hydration.water_roles", Title: "水在人体中的基本作用", Summary: "理解水不仅用于解渴，也参与体温、运输和细胞环境。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 1},
	{UnitKey: "nutrition.hydration", Key: "nutrition.hydration.fluid_balance", Title: "体液平衡是如何维持的", Summary: "理解摄入、排出和调节共同形成动态体液平衡。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 2},
	{UnitKey: "nutrition.hydration", Key: "nutrition.hydration.thirst_signal", Title: "口渴是否是可靠的饮水依据", Summary: "理解口渴是重要信号但不是所有场景下的唯一依据。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 3},
	{UnitKey: "nutrition.hydration", Key: "nutrition.hydration.daily_need", Title: "日常饮水需求应该如何判断", Summary: "理解饮水判断需要结合个体、环境和活动等因素。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 4},
	{UnitKey: "nutrition.hydration", Key: "nutrition.hydration.heat_exercise", Title: "高温和运动为什么会改变补水需求", Summary: "把体液知识应用到高温和运动场景。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleApplication, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 5},
	{UnitKey: "nutrition.hydration", Key: "nutrition.hydration.electrolytes", Title: "大量出汗后为什么需要关注电解质", Summary: "理解大量出汗时水和电解质的共同变化。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleApplication, DepthLevel: 3, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 6},
	{UnitKey: "nutrition.hydration", Key: "nutrition.hydration.age_condition", Title: "年龄与疾病状态为什么会影响口渴信号", Summary: "理解年龄和身体状态带来的信号边界。", Importance: model.CurriculumImportanceRecommended, ContentRole: model.ContentRoleExtension, DepthLevel: 3, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 7},
	{UnitKey: "nutrition.hydration", Key: "nutrition.hydration.integrated_strategy", Title: "如何综合判断不同场景下的补水策略", Summary: "综合多个信息判断不同场景下的补水策略。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleApplication, DepthLevel: 4, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 8},
	{UnitKey: "nutrition.label", Key: "nutrition.label.ingredients", Title: "配料表能告诉我们什么", Summary: "理解配料表提供的信息及其局限。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleFoundation, DepthLevel: 1, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 1},
	{UnitKey: "nutrition.label", Key: "nutrition.label.nutrition_facts", Title: "营养成分表能告诉我们什么", Summary: "理解营养成分表的主要字段和比较方式。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 2},
	{UnitKey: "nutrition.label", Key: "nutrition.label.combined_reasoning", Title: "为什么需要把配料表和营养成分表结合判断", Summary: "综合两类标签信息进行饮食判断。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleApplication, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 3},
	{UnitKey: "nutrition.meal", Key: "nutrition.meal.balance", Title: "一餐怎样同时考虑主食、蛋白质和蔬菜", Summary: "理解一餐中主要营养结构的基本平衡。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleApplication, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 1},
	{UnitKey: "nutrition.meal", Key: "nutrition.meal.pattern", Title: "为什么长期饮食模式比单顿饭更重要", Summary: "区分单顿选择和长期饮食模式。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleCore, DepthLevel: 2, AssessmentTargetLevel: model.AssessmentTargetUnderstand, SortOrder: 2},
	{UnitKey: "nutrition.meal", Key: "nutrition.meal.optimization", Title: "怎样给一顿普通饮食做优先级优化", Summary: "把营养知识用于现实饮食的优先级调整。", Importance: model.CurriculumImportanceCore, ContentRole: model.ContentRoleApplication, DepthLevel: 3, AssessmentTargetLevel: model.AssessmentTargetApply, SortOrder: 3},
}

var nutritionBlueprintRelations = []nutritionBlueprintRelationDefinition{
	{From: "nutrition.foundation.energy_balance", To: "nutrition.foundation.nutrient_density", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.carb.function", To: "nutrition.carb.quality", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.protein.function", To: "nutrition.protein.quality", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.fat.function", To: "nutrition.fat.types", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.mineral.role", To: "nutrition.mineral.sodium_potassium", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.hydration.water_roles", To: "nutrition.hydration.fluid_balance", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.hydration.fluid_balance", To: "nutrition.hydration.thirst_signal", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.hydration.thirst_signal", To: "nutrition.hydration.daily_need", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.hydration.daily_need", To: "nutrition.hydration.integrated_strategy", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.label.ingredients", To: "nutrition.label.combined_reasoning", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.label.nutrition_facts", To: "nutrition.label.combined_reasoning", RelationType: string(model.LessonRelationPrerequisite)},
	{From: "nutrition.meal.balance", To: "nutrition.meal.optimization", RelationType: string(model.LessonRelationPrerequisite)},
}

func seedNutritionBlueprint(tx *gorm.DB, courseID uint, lessonIDs map[string]uint) error {
	var blueprint model.CurriculumBlueprint
	err := tx.Where("course_id = ? AND name = ? AND version = ?", courseID, "Nutrition Core Blueprint", "v0").First(&blueprint).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		blueprint = model.CurriculumBlueprint{CourseID: &courseID, Name: "Nutrition Core Blueprint", Domain: "nutrition", Description: "用于检查营养学核心知识覆盖的暂定课程蓝图。", LearningGoal: "建立理解营养素、体液平衡、食品标签和膳食应用的知识骨架。", Audience: "希望建立系统营养学基础的学习者", TargetDepth: "foundation_to_application", Version: "v0", Status: model.CurriculumBlueprintStatusActive, CreatedBy: model.CurriculumBlueprintCreatedBySeed, GroundingStatus: model.CurriculumGroundingProvisional}
		if err := tx.Create(&blueprint).Error; err != nil {
			return fmt.Errorf("create nutrition blueprint: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find nutrition blueprint: %w", err)
	}
	unitIDs := map[string]uint{}
	for _, definition := range nutritionBlueprintUnits {
		var unit model.CurriculumBlueprintUnit
		err := tx.Where("blueprint_id = ? AND key = ?", blueprint.ID, definition.Key).First(&unit).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			unit = model.CurriculumBlueprintUnit{BlueprintID: blueprint.ID, Key: definition.Key, Title: definition.Title, Description: definition.Description, SortOrder: definition.SortOrder, Importance: definition.Importance}
			if err := tx.Create(&unit).Error; err != nil {
				return fmt.Errorf("create nutrition blueprint unit %s: %w", definition.Key, err)
			}
		} else if err != nil {
			return fmt.Errorf("find nutrition blueprint unit %s: %w", definition.Key, err)
		}
		unitIDs[definition.Key] = unit.ID
	}
	for _, definition := range nutritionBlueprintLessons {
		var lesson model.CurriculumBlueprintLesson
		err := tx.Where("blueprint_id = ? AND key = ?", blueprint.ID, definition.Key).First(&lesson).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lesson = model.CurriculumBlueprintLesson{BlueprintID: blueprint.ID, BlueprintUnitID: unitIDs[definition.UnitKey], Key: definition.Key, Title: definition.Title, Summary: definition.Summary, Importance: definition.Importance, ContentRole: definition.ContentRole, DepthLevel: definition.DepthLevel, AssessmentTargetLevel: definition.AssessmentTargetLevel, SortOrder: definition.SortOrder, GroundingStatus: model.CurriculumGroundingUngrounded}
			if appliedID := lessonIDs[definition.Title]; appliedID != 0 {
				lesson.AppliedLessonID = &appliedID
			}
			if err := tx.Create(&lesson).Error; err != nil {
				return fmt.Errorf("create nutrition blueprint lesson %s: %w", definition.Key, err)
			}
		} else if err != nil {
			return fmt.Errorf("find nutrition blueprint lesson %s: %w", definition.Key, err)
		} else if lesson.AppliedLessonID == nil {
			if appliedID := lessonIDs[definition.Title]; appliedID != 0 {
				if err := tx.Model(&lesson).Update("applied_lesson_id", appliedID).Error; err != nil {
					return fmt.Errorf("map nutrition blueprint lesson %s: %w", definition.Key, err)
				}
			}
		}
	}
	for _, definition := range nutritionBlueprintRelations {
		var relation model.CurriculumBlueprintRelation
		err := tx.Where("blueprint_id = ? AND from_lesson_key = ? AND to_lesson_key = ? AND relation_type = ?", blueprint.ID, definition.From, definition.To, definition.RelationType).First(&relation).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&model.CurriculumBlueprintRelation{BlueprintID: blueprint.ID, FromLessonKey: definition.From, ToLessonKey: definition.To, RelationType: definition.RelationType}).Error; err != nil {
				return fmt.Errorf("create nutrition blueprint relation: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("find nutrition blueprint relation: %w", err)
		}
	}
	return nil
}
