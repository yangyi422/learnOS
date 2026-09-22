package ai

import (
	"context"
	"sync"
)

// RuntimeProvider keeps service dependencies stable while allowing the
// single-user system configuration to replace the active provider at runtime.
// It implements every optional provider interface and returns ErrNotConfigured
// when the active provider does not support an operation.
type RuntimeProvider struct {
	mu       sync.RWMutex
	provider AIProvider
}

func NewRuntimeProvider(provider AIProvider) *RuntimeProvider {
	return &RuntimeProvider{provider: provider}
}

func (p *RuntimeProvider) SetProvider(provider AIProvider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.provider = provider
}

func (p *RuntimeProvider) current() AIProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.provider
}

func (p *RuntimeProvider) CurrentProvider() AIProvider {
	return p.current()
}

func (p *RuntimeProvider) EvaluateLessonAnswer(ctx context.Context, req EvaluationRequest) (EvaluationResult, ProviderMeta, error) {
	provider, ok := p.current().(AIProvider)
	if !ok || provider == nil {
		return EvaluationResult{}, ProviderMeta{}, ErrNotConfigured
	}
	return provider.EvaluateLessonAnswer(ctx, req)
}

func (p *RuntimeProvider) GenerateExploration(ctx context.Context, req ExplorationRequest) (ExplorationResult, ProviderMeta, error) {
	provider, ok := p.current().(ExplorationProvider)
	if !ok {
		return ExplorationResult{}, ProviderMeta{}, ErrNotConfigured
	}
	return provider.GenerateExploration(ctx, req)
}

func (p *RuntimeProvider) GenerateCurriculumDraft(ctx context.Context, req CurriculumDraftRequest) (CurriculumDraftResult, ProviderMeta, error) {
	provider, ok := p.current().(CurriculumDraftProvider)
	if !ok {
		return CurriculumDraftResult{}, ProviderMeta{}, ErrNotConfigured
	}
	return provider.GenerateCurriculumDraft(ctx, req)
}

func (p *RuntimeProvider) GenerateDomainSkeleton(ctx context.Context, req DomainSkeletonRequest) (DomainSkeletonResult, ProviderMeta, error) {
	provider, ok := p.current().(DomainInitializationProvider)
	if !ok {
		return DomainSkeletonResult{}, ProviderMeta{}, ErrNotConfigured
	}
	return provider.GenerateDomainSkeleton(ctx, req)
}

func (p *RuntimeProvider) ExpandDomainStarter(ctx context.Context, req DomainStarterRequest) (DomainStarterResult, ProviderMeta, error) {
	provider, ok := p.current().(DomainInitializationProvider)
	if !ok {
		return DomainStarterResult{}, ProviderMeta{}, ErrNotConfigured
	}
	return provider.ExpandDomainStarter(ctx, req)
}

func (p *RuntimeProvider) GenerateInitialWorld(ctx context.Context, req InitialWorldRequest) (InitialWorldResult, ProviderMeta, error) {
	provider, ok := p.current().(DomainInitializationProvider)
	if !ok {
		return InitialWorldResult{}, ProviderMeta{}, ErrNotConfigured
	}
	return provider.GenerateInitialWorld(ctx, req)
}

func (p *RuntimeProvider) ExpandBlueprintUnit(ctx context.Context, req BlueprintUnitExpansionRequest) (BlueprintUnitExpansionResult, ProviderMeta, error) {
	provider, ok := p.current().(BlueprintUnitExpansionProvider)
	if !ok {
		return BlueprintUnitExpansionResult{}, ProviderMeta{}, ErrNotConfigured
	}
	return provider.ExpandBlueprintUnit(ctx, req)
}

func (p *RuntimeProvider) GenerateChallenge(ctx context.Context, req ChallengeGenerationRequest) (ChallengeGenerationResult, ProviderMeta, error) {
	provider, ok := p.current().(ChallengeProvider)
	if !ok {
		return ChallengeGenerationResult{}, ProviderMeta{}, ErrNotConfigured
	}
	return provider.GenerateChallenge(ctx, req)
}

func (p *RuntimeProvider) EvaluateChallenge(ctx context.Context, req ChallengeEvaluationRequest) (ChallengeEvaluationResult, ProviderMeta, error) {
	provider, ok := p.current().(ChallengeProvider)
	if !ok {
		return ChallengeEvaluationResult{}, ProviderMeta{}, ErrNotConfigured
	}
	return provider.EvaluateChallenge(ctx, req)
}
