package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) GenerateDomainSkeleton(_ context.Context, req DomainSkeletonRequest) (DomainSkeletonResult, ProviderMeta, error) {
	units := make([]DomainUnit, 0, 6)
	for i, title := range []string{"基础概念与范围", "核心组成", "关键机制", "证据与判断", "常见应用", "边界与实践"} {
		units = append(units, DomainUnit{Key: fmt.Sprintf("unit-%d", i+1), Title: title, Description: fmt.Sprintf("%s的基础区域。", req.DomainName), Importance: "core"})
	}
	result := DomainSkeletonResult{Course: DomainCourseMetadata{Name: req.DomainName, Description: fmt.Sprintf("围绕%s建立渐进式学习地图。", req.DomainName)}, Blueprint: DomainBlueprintMetadata{Name: req.DomainName + "学习地图", Domain: req.DomainName, LearningGoal: req.LearningGoal, TargetDepth: req.TargetDepth, Units: units}, RecommendedStarterKeys: []string{"unit-1"}}
	raw, _ := json.Marshal(result)
	return result, ProviderMeta{Provider: "mock", Model: "mock", PromptVersion: DomainSkeletonPromptVersion, RawResponse: string(raw), AttemptCount: 1}, nil
}

func (p *MockProvider) ExpandDomainStarter(_ context.Context, req DomainStarterRequest) (DomainStarterResult, ProviderMeta, error) {
	keys := req.SelectedStarterKeys
	if len(keys) == 0 {
		keys = req.Skeleton.RecommendedStarterKeys
	}
	if len(keys) > 2 {
		keys = keys[:2]
	}
	lessons := []DomainBlueprintLesson{{Key: "starter-1", Title: "这个领域的核心问题是什么", Summary: "建立领域边界和基本问题意识。", Importance: "core", ContentRole: "foundation", DepthLevel: 1, AssessmentTargetLevel: "understand"}, {Key: "starter-2", Title: "最重要的基础概念如何联系", Summary: "理解组成部分之间的基本关系。", Importance: "core", ContentRole: "foundation", DepthLevel: 1, AssessmentTargetLevel: "understand"}, {Key: "starter-3", Title: "如何判断一个常见说法", Summary: "建立初步的证据和判断框架。", Importance: "core", ContentRole: "core", DepthLevel: 1, AssessmentTargetLevel: "apply"}, {Key: "starter-4", Title: "常见场景中如何应用", Summary: "把基础概念放进日常场景。", Importance: "recommended", ContentRole: "application", DepthLevel: 2, AssessmentTargetLevel: "apply"}, {Key: "starter-5", Title: "这个领域有哪些边界", Summary: "识别过度简化和适用条件。", Importance: "recommended", ContentRole: "extension", DepthLevel: 2, AssessmentTargetLevel: "understand"}}
	result := DomainStarterResult{ExpandedUnits: []ExpandedDomainUnit{{Key: keys[0], Lessons: lessons}}, Relations: []DomainLessonRelation{{FromLessonKey: "starter-1", ToLessonKey: "starter-2", RelationType: "prerequisite"}, {FromLessonKey: "starter-2", ToLessonKey: "starter-3", RelationType: "prerequisite"}, {FromLessonKey: "starter-3", ToLessonKey: "starter-4", RelationType: "application"}}}
	raw, _ := json.Marshal(result)
	return result, ProviderMeta{Provider: "mock", Model: "mock", PromptVersion: DomainStarterPromptVersion, RawResponse: string(raw), AttemptCount: 1}, nil
}

func (p *MockProvider) GenerateInitialWorld(_ context.Context, req InitialWorldRequest) (InitialWorldResult, ProviderMeta, error) {
	all := make([]DomainBlueprintLesson, 0)
	for _, unit := range req.Starter.ExpandedUnits {
		all = append(all, unit.Lessons...)
	}
	if len(all) < 3 {
		return InitialWorldResult{}, ProviderMeta{}, newProviderError(ErrInvalidResponse, fmt.Errorf("mock starter has too few lessons"))
	}
	items := make([]InitialWorldLesson, 0, 3)
	for _, lesson := range all[:3] {
		items = append(items, InitialWorldLesson{BlueprintLessonKey: lesson.Key, Title: lesson.Title, CoreQuestion: "关于" + lesson.Title + "，最需要先理解什么？", ExpectedUnderstanding: "能够说明这个知识点的基本含义、适用边界以及它与其他概念的联系。", ContentRole: lesson.ContentRole, DepthLevel: lesson.DepthLevel, AssessmentTargetLevel: lesson.AssessmentTargetLevel, IsCore: true})
	}
	result := InitialWorldResult{InitialLessons: items, RecommendedFirstKey: items[0].BlueprintLessonKey, Relations: []DomainLessonRelation{{FromLessonKey: items[0].BlueprintLessonKey, ToLessonKey: items[1].BlueprintLessonKey, RelationType: "prerequisite"}, {FromLessonKey: items[1].BlueprintLessonKey, ToLessonKey: items[2].BlueprintLessonKey, RelationType: "prerequisite"}}}
	raw, _ := json.Marshal(result)
	return result, ProviderMeta{Provider: "mock", Model: "mock", PromptVersion: DomainInitialWorldPromptVersion, RawResponse: string(raw), AttemptCount: 1}, nil
}

func (p *MockProvider) ExpandBlueprintUnit(_ context.Context, req BlueprintUnitExpansionRequest) (BlueprintUnitExpansionResult, ProviderMeta, error) {
	keyPrefix := req.TargetUnit.Key
	lessons := make([]DomainBlueprintLesson, 0, 7)
	for index, title := range []string{"基本概念", "关键组成", "运行机制", "常见判断", "实践方法", "边界条件", "综合应用"} {
		role := "core"
		if index == 0 {
			role = "foundation"
		} else if index >= 4 {
			role = "application"
		}
		lessons = append(lessons, DomainBlueprintLesson{
			Key: fmt.Sprintf("%s.lesson-%d", keyPrefix, index+1), Title: req.TargetUnit.Title + "：" + title,
			Summary: req.TargetUnit.Title + "中的" + title + "。", Importance: "core", ContentRole: role,
			DepthLevel: 1 + index/4, AssessmentTargetLevel: "understand",
		})
	}
	relations := make([]DomainLessonRelation, 0, len(lessons)-1)
	for index := 1; index < len(lessons); index++ {
		relations = append(relations, DomainLessonRelation{FromLessonKey: lessons[index-1].Key, ToLessonKey: lessons[index].Key, RelationType: "prerequisite"})
	}
	result := BlueprintUnitExpansionResult{Lessons: lessons, Relations: relations}
	raw, _ := json.Marshal(result)
	return result, ProviderMeta{Provider: "mock", Model: "mock", PromptVersion: BlueprintUnitExpansionPromptVersion, RawResponse: string(raw), AttemptCount: 1}, nil
}

func (p *MockProvider) GenerateExploration(_ context.Context, req ExplorationRequest) (ExplorationResult, ProviderMeta, error) {
	result := ExplorationResult{
		Summary:           fmt.Sprintf("从%s延伸到%s，观察一个新的知识连接。", req.SourceLesson, req.TargetLesson),
		WhyWorthExploring: req.Why,
	}
	if strings.TrimSpace(result.WhyWorthExploring) == "" {
		result.WhyWorthExploring = "它可以帮助你把当前判断放到另一个知识场景中检验。"
	}
	raw, _ := json.Marshal(result)
	return result, ProviderMeta{Provider: "mock", Model: "mock", PromptVersion: ExplorationPromptVersion, RawResponse: string(raw), AttemptCount: 1}, nil
}

func (p *MockProvider) EvaluateLessonAnswer(_ context.Context, req EvaluationRequest) (EvaluationResult, ProviderMeta, error) {
	result := EvaluationResult{
		Result:      "mostly_correct",
		Feedback:    "你正确认识到口渴不是判断身体水分状态的唯一依据。",
		Explanation: "口渴是重要的水分调节信号，但不同环境、身体状态和个体差异会影响它作为唯一依据的可靠性。",
		CorrectParts: []string{
			"认识到口渴只是身体调节水分的信号之一",
			"认识到不能只依赖单一信号判断",
		},
		MissingParts: []string{
			"还可以补充高温、运动、疾病和年龄等影响因素",
		},
		BoundaryConditions: []string{
			"口渴仍然是日常饮水的重要信号，但不能被理解为所有场景下的唯一依据",
		},
		MasteryEvidence: []string{
			"回答体现了对单一口渴信号局限性的认识",
		},
		EvidenceUsed: []string{
			"用户明确写到不能只依赖单一口渴信号",
			"用户提到了环境和身体状态会影响判断",
		},
		Confidence:                0.86,
		Uncertainty:               "回答尚未展开年龄、疾病和运动强度等具体边界，因此对完整理解仍有保留。",
		RecommendedNextAction:     "补充一个具体场景，说明除口渴外还会结合哪些信号判断。",
		TransferChallengeEligible: false,
		MasteryScore:              0.75,
		NeedsReview:               true,
		DemonstratedLevel:         "understand",
		UserUnderstandingSummary:  "用户认识到口渴不是唯一的补水依据，并能指出环境和身体状态会影响判断。",
		CognitiveEvidence: []EvaluationEvidence{
			{EvidenceType: "recognition", CognitiveLevel: "recognize", Polarity: "support", Description: "能够识别没有口渴不等于一定不缺水"},
			{EvidenceType: "boundary_awareness", CognitiveLevel: "understand", Polarity: "support", Description: "能够指出环境和身体状态会影响补水判断"},
		},
	}
	if len([]rune(strings.TrimSpace(req.UserAnswer))) < 5 {
		result.Result = "insufficient"
		result.Feedback = "回答还不够完整，请进一步说明口渴信号的局限性。"
		result.Explanation = "当前回答过短，无法可靠判断你是否理解口渴信号的适用边界。"
		result.CorrectParts = []string{}
		result.MissingParts = []string{"还需要说明口渴不能作为唯一的补水依据"}
		result.MasteryEvidence = []string{}
		result.EvidenceUsed = []string{"回答内容过短，未提供可验证的判断依据"}
		result.Confidence = 0.95
		result.Uncertainty = "信息不足，无法判断用户是否理解核心机制。"
		result.RecommendedNextAction = "用自己的话解释口渴信号为什么不能作为唯一依据。"
		result.TransferChallengeEligible = false
		result.MasteryScore = 0.25
		result.DemonstratedLevel = "exposed"
		result.UserUnderstandingSummary = "本次回答没有提供足够信息证明对当前知识的理解。"
		result.CognitiveEvidence = []EvaluationEvidence{}
	}
	if req.AssessmentTargetLevel == "recognize" && result.DemonstratedLevel != "exposed" {
		result.DemonstratedLevel = "recognize"
		result.UserUnderstandingSummary = "用户能够识别当前概念的核心判断，但本题目标不要求完整解释。"
		result.CognitiveEvidence = []EvaluationEvidence{{
			EvidenceType: "recognition", CognitiveLevel: "recognize", Polarity: "support",
			Description: "能够识别当前概念的核心判断",
		}}
	}
	var err error
	if strings.TrimSpace(req.AssessmentTargetLevel) == "" {
		result, err = NormalizeAndValidate(result)
	} else {
		result, err = NormalizeAndValidateForTarget(result, req.AssessmentTargetLevel)
	}
	if err != nil {
		return EvaluationResult{}, ProviderMeta{}, err
	}
	raw, _ := json.Marshal(result)
	return result, ProviderMeta{
		Provider:      "mock",
		Model:         "mock",
		PromptVersion: PromptVersion,
		RawResponse:   string(raw),
		AttemptCount:  1,
	}, nil
}

func (p *MockProvider) GenerateChallenge(_ context.Context, req ChallengeGenerationRequest) (ChallengeGenerationResult, ProviderMeta, error) {
	result := ChallengeGenerationResult{
		Prompt:          "一名老人在高温天气散步后表示自己并不口渴，家人因此认为无需关注饮水。这个判断有什么问题？你会结合哪些信息判断？",
		ScenarioContext: "高温天气；一名老人散步后没有明显口渴感。",
		TargetLevel:     "transfer",
		EvaluationCriteria: []string{
			"识别不能只用单一口渴信号判断",
			"结合人物和环境场景说明需要关注的信息",
			"说明结论的适用边界",
		},
		WhyThisIsTransfer: "该任务把口渴信号的局限迁移到不同人群和环境场景中。",
		SourceConcepts:    []string{"口渴信号的局限", "场景因素"},
	}
	if req.ChallengeType == "misconception_recheck" {
		result.TargetLevel = "understand"
		result.Prompt = "有人认为“不口渴就一定不缺水”。请结合一个具体日常场景说明这个判断是否可靠，以及还需要考虑什么。"
		result.ScenarioContext = "针对已发现的判断误区设计一个新的日常场景。"
		result.WhyThisIsTransfer = "该任务要求用户在新的问题表述中重新解释并限定原有判断。"
	}
	result, err := NormalizeAndValidateChallengeGeneration(result, req.ChallengeType)
	if err != nil {
		return ChallengeGenerationResult{}, ProviderMeta{}, err
	}
	raw, _ := json.Marshal(result)
	return result, ProviderMeta{Provider: "mock", Model: "mock", PromptVersion: TransferGeneratorPromptVersion, RawResponse: string(raw), AttemptCount: 1}, nil
}

func (p *MockProvider) EvaluateChallenge(_ context.Context, req ChallengeEvaluationRequest) (ChallengeEvaluationResult, ProviderMeta, error) {
	result := ChallengeEvaluationResult{
		Result:            "mostly_correct",
		Feedback:          "回答能够把核心判断应用到新的场景中。",
		Explanation:       "回答识别了单一信号的局限，并结合场景因素进行判断。",
		DemonstratedLevel: "transfer",
		CognitiveEvidence: []EvaluationEvidence{{EvidenceType: "transfer", CognitiveLevel: "transfer", Polarity: "support", Description: "能够将核心判断迁移到新的场景"}},
	}
	if req.ChallengeType == "misconception_recheck" {
		result.DemonstratedLevel = "understand"
		result.CognitiveEvidence = []EvaluationEvidence{{EvidenceType: "concept_explanation", CognitiveLevel: "understand", Polarity: "support", Description: "能够在新问题中解释并限定原有判断"}}
		result.MisconceptionValidation = &MisconceptionValidation{TargetMisconceptionID: req.TargetMisconceptionID, Status: "corrected", Evidence: "用户在新的问题中否定了绝对化判断，并说明了需要结合场景因素。"}
	}
	if len([]rune(strings.TrimSpace(req.UserAnswer))) < 5 || strings.Contains(req.UserAnswer, "不知道") {
		result.Result = "insufficient"
		result.Feedback = "回答没有提供足够信息完成挑战。"
		result.Explanation = "当前回答不足以判断是否完成了迁移或误区修正。"
		result.DemonstratedLevel = "exposed"
		result.CognitiveEvidence = []EvaluationEvidence{}
		if req.ChallengeType == "misconception_recheck" {
			result.MisconceptionValidation.Status = "unclear"
			result.MisconceptionValidation.Evidence = "本次回答没有提供足够信息判断误区是否修正。"
		}
	}
	result, err := NormalizeAndValidateChallengeEvaluation(result, req.ChallengeType, req.TargetLevel, req.TargetMisconceptionID)
	if err != nil {
		return ChallengeEvaluationResult{}, ProviderMeta{}, err
	}
	raw, _ := json.Marshal(result)
	return result, ProviderMeta{Provider: "mock", Model: "mock", PromptVersion: ChallengeEvaluatorPromptVersion, RawResponse: string(raw), AttemptCount: 1}, nil
}
