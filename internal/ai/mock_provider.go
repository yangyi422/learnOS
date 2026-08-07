package ai

import (
	"context"
	"encoding/json"
	"strings"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
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
		MasteryScore:             0.75,
		NeedsReview:              true,
		DemonstratedLevel:        "understand",
		UserUnderstandingSummary: "用户认识到口渴不是唯一的补水依据，并能指出环境和身体状态会影响判断。",
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
