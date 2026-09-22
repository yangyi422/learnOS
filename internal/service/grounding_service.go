package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"learnos/internal/model"
	"learnos/internal/repository"
)

var (
	ErrKnowledgeSourceNotFound = errors.New("knowledge source not found")
	ErrSourceEvidenceNotFound  = errors.New("source evidence not found")
	ErrGroundingLinkNotFound   = errors.New("grounding link not found")
	ErrCredibilityNotFound     = errors.New("source credibility assessment not found")
	ErrSourceDuplicate         = errors.New("knowledge source already exists")
	ErrSourceInvalid           = errors.New("invalid knowledge source")
	ErrEvidenceInvalid         = errors.New("invalid source evidence")
	ErrGroundingLinkInvalid    = errors.New("invalid grounding link")
	ErrGroundingLinkConflict   = errors.New("grounding link already exists")
	ErrCredibilityInvalid      = errors.New("invalid credibility assessment")
	ErrGroundingTargetNotFound = errors.New("grounding target not found")
)

type GroundingService struct {
	sources    *repository.GroundingRepository
	curriculum *repository.CurriculumRepository
	courses    *repository.CourseRepository
}

func NewGroundingService(sources *repository.GroundingRepository, curriculum *repository.CurriculumRepository, courses *repository.CourseRepository) *GroundingService {
	return &GroundingService{sources: sources, curriculum: curriculum, courses: courses}
}

type SourceInput struct {
	Title              string     `json:"title"`
	Authors            string     `json:"authors"`
	Organization       string     `json:"organization"`
	SourceType         string     `json:"source_type"`
	Publisher          string     `json:"publisher"`
	PublicationYear    *int       `json:"publication_year"`
	PublishedAt        *time.Time `json:"published_at"`
	UpdatedAtSource    *time.Time `json:"updated_at_source"`
	URL                string     `json:"url"`
	DOI                string     `json:"doi"`
	ISBN               string     `json:"isbn"`
	Language           string     `json:"language"`
	Description        string     `json:"description"`
	AccessStatus       string     `json:"access_status"`
	VerificationStatus string     `json:"verification_status"`
}

type SourcePatch struct {
	Title              *string    `json:"title"`
	Authors            *string    `json:"authors"`
	Organization       *string    `json:"organization"`
	SourceType         *string    `json:"source_type"`
	Publisher          *string    `json:"publisher"`
	PublicationYear    *int       `json:"publication_year"`
	PublishedAt        *time.Time `json:"published_at"`
	UpdatedAtSource    *time.Time `json:"updated_at_source"`
	URL                *string    `json:"url"`
	DOI                *string    `json:"doi"`
	ISBN               *string    `json:"isbn"`
	Language           *string    `json:"language"`
	Description        *string    `json:"description"`
	AccessStatus       *string    `json:"access_status"`
	VerificationStatus *string    `json:"verification_status"`
}

type EvidenceInput struct {
	EvidenceType       string `json:"evidence_type"`
	Locator            string `json:"locator"`
	Quote              string `json:"quote"`
	Summary            string `json:"summary"`
	Language           string `json:"language"`
	ExtractionMethod   string `json:"extraction_method"`
	VerificationStatus string `json:"verification_status"`
}

type EvidencePatch struct {
	EvidenceType       *string `json:"evidence_type"`
	Locator            *string `json:"locator"`
	Quote              *string `json:"quote"`
	Summary            *string `json:"summary"`
	Language           *string `json:"language"`
	ExtractionMethod   *string `json:"extraction_method"`
	VerificationStatus *string `json:"verification_status"`
}

type GroundingLinkInput struct {
	EvidenceID uint   `json:"evidence_id"`
	TargetType string `json:"target_type"`
	TargetID   uint   `json:"target_id"`
	Relation   string `json:"relation"`
	Strength   string `json:"strength"`
	Rationale  string `json:"rationale"`
}

type CredibilityInput struct {
	AuthorityScore     int    `json:"authority_score"`
	MethodologyScore   int    `json:"methodology_score"`
	DirectnessScore    int    `json:"directness_score"`
	RecencyScore       int    `json:"recency_score"`
	IndependenceScore  int    `json:"independence_score"`
	AuthorityReason    string `json:"authority_reason"`
	MethodologyReason  string `json:"methodology_reason"`
	DirectnessReason   string `json:"directness_reason"`
	RecencyReason      string `json:"recency_reason"`
	IndependenceReason string `json:"independence_reason"`
	AssessmentMethod   string `json:"assessment_method"`
}

type SourceView struct {
	Source      model.KnowledgeSource               `json:"source"`
	Credibility []model.SourceCredibilityAssessment `json:"credibility"`
	Evidence    []model.SourceEvidence              `json:"evidence"`
}

type GroundingLinkView struct {
	Link        model.GroundingLink                `json:"link"`
	Evidence    model.SourceEvidence               `json:"evidence"`
	Source      model.KnowledgeSource              `json:"source"`
	Credibility *model.SourceCredibilityAssessment `json:"credibility"`
}

type GroundingTargetView struct {
	TargetType            string                       `json:"target_type"`
	TargetID              uint                         `json:"target_id"`
	GroundingStatus       string                       `json:"grounding_status"`
	ReviewedSourceCount   int                          `json:"reviewed_source_count"`
	ReviewedEvidenceCount int                          `json:"reviewed_evidence_count"`
	ConflictCount         int                          `json:"conflict_count"`
	Links                 []GroundingLinkView          `json:"links"`
	ReviewEvents          []model.GroundingReviewEvent `json:"review_events"`
}

type GroundingCoverageMetrics struct {
	CoreTotal           int     `json:"core_total"`
	CoreGrounded        int     `json:"core_grounded"`
	CorePartial         int     `json:"core_partial"`
	CoreUngrounded      int     `json:"core_ungrounded"`
	CoreConflicted      int     `json:"core_conflicted"`
	RecommendedTotal    int     `json:"recommended_total"`
	RecommendedGrounded int     `json:"recommended_grounded"`
	GroundingPercent    float64 `json:"grounding_percent"`
}

type LessonGroundingSummary struct {
	ID                  uint   `json:"id"`
	Title               string `json:"title"`
	GroundingStatus     string `json:"grounding_status"`
	ReviewedSourceCount int    `json:"reviewed_source_count"`
}

type GroundingCoverageLesson struct {
	BlueprintLesson       model.CurriculumBlueprintLesson `json:"blueprint_lesson"`
	GroundingStatus       string                          `json:"grounding_status"`
	ReviewedSourceCount   int                             `json:"reviewed_source_count"`
	ReviewedEvidenceCount int                             `json:"reviewed_evidence_count"`
	ConflictCount         int                             `json:"conflict_count"`
	AppliedLesson         *LessonGroundingSummary         `json:"applied_lesson"`
}

type GroundingCoverageUnit struct {
	Unit    model.CurriculumBlueprintUnit `json:"unit"`
	Lessons []GroundingCoverageLesson     `json:"lessons"`
}

type GroundingCoverageView struct {
	Blueprint model.CurriculumBlueprint `json:"blueprint"`
	Metrics   GroundingCoverageMetrics  `json:"metrics"`
	Units     []GroundingCoverageUnit   `json:"units"`
}

func (s *GroundingService) ListSources(ctx context.Context, query, sourceType string, limit int) ([]model.KnowledgeSource, error) {
	return s.sources.ListSources(ctx, strings.TrimSpace(query), strings.TrimSpace(sourceType), limit)
}

func (s *GroundingService) CreateSource(ctx context.Context, input SourceInput) (*model.KnowledgeSource, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrSourceInvalid
	}
	source := &model.KnowledgeSource{Title: title, Authors: strings.TrimSpace(input.Authors), Organization: strings.TrimSpace(input.Organization), SourceType: strings.TrimSpace(input.SourceType), Publisher: strings.TrimSpace(input.Publisher), PublicationYear: input.PublicationYear, PublishedAt: input.PublishedAt, UpdatedAtSource: input.UpdatedAtSource, URL: strings.TrimSpace(input.URL), DOI: strings.TrimSpace(input.DOI), ISBN: strings.TrimSpace(input.ISBN), Language: strings.TrimSpace(input.Language), Description: strings.TrimSpace(input.Description), AccessStatus: strings.TrimSpace(input.AccessStatus), VerificationStatus: strings.TrimSpace(input.VerificationStatus), CreatedBy: model.KnowledgeSourceCreatedByManual}
	applySourceDefaults(source)
	source.NormalizedURL = normalizeURL(source.URL)
	duplicate, err := s.sources.FindDuplicateSource(ctx, source)
	if err == nil && duplicate != nil {
		return nil, fmt.Errorf("%w: source_id=%d", ErrSourceDuplicate, duplicate.ID)
	}
	if !repository.IsGroundingNotFound(err) {
		return nil, err
	}
	if err := s.sources.CreateSource(ctx, source); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *GroundingService) GetSource(ctx context.Context, id uint) (*SourceView, error) {
	source, err := s.sources.FindSource(ctx, id)
	if err != nil {
		if repository.IsGroundingNotFound(err) {
			return nil, ErrKnowledgeSourceNotFound
		}
		return nil, err
	}
	credibility, err := s.sources.ListCredibility(ctx, id)
	if err != nil {
		return nil, err
	}
	evidence, err := s.sources.ListEvidence(ctx, id)
	if err != nil {
		return nil, err
	}
	return &SourceView{Source: *source, Credibility: credibility, Evidence: evidence}, nil
}

func (s *GroundingService) UpdateSource(ctx context.Context, id uint, patch SourcePatch) (*SourceView, error) {
	source, err := s.sources.FindSource(ctx, id)
	if err != nil {
		if repository.IsGroundingNotFound(err) {
			return nil, ErrKnowledgeSourceNotFound
		}
		return nil, err
	}
	if patch.Title != nil {
		source.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Authors != nil {
		source.Authors = strings.TrimSpace(*patch.Authors)
	}
	if patch.Organization != nil {
		source.Organization = strings.TrimSpace(*patch.Organization)
	}
	if patch.SourceType != nil {
		source.SourceType = strings.TrimSpace(*patch.SourceType)
	}
	if patch.Publisher != nil {
		source.Publisher = strings.TrimSpace(*patch.Publisher)
	}
	if patch.PublicationYear != nil {
		source.PublicationYear = patch.PublicationYear
	}
	if patch.PublishedAt != nil {
		source.PublishedAt = patch.PublishedAt
	}
	if patch.UpdatedAtSource != nil {
		source.UpdatedAtSource = patch.UpdatedAtSource
	}
	if patch.URL != nil {
		source.URL = strings.TrimSpace(*patch.URL)
		source.NormalizedURL = normalizeURL(source.URL)
	}
	if patch.DOI != nil {
		source.DOI = strings.TrimSpace(*patch.DOI)
	}
	if patch.ISBN != nil {
		source.ISBN = strings.TrimSpace(*patch.ISBN)
	}
	if patch.Language != nil {
		source.Language = strings.TrimSpace(*patch.Language)
	}
	if patch.Description != nil {
		source.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.AccessStatus != nil {
		source.AccessStatus = strings.TrimSpace(*patch.AccessStatus)
	}
	if patch.VerificationStatus != nil {
		source.VerificationStatus = strings.TrimSpace(*patch.VerificationStatus)
	}
	if source.Title == "" {
		return nil, ErrSourceInvalid
	}
	applySourceDefaults(source)
	duplicate, duplicateErr := s.sources.FindDuplicateSource(ctx, source)
	if duplicateErr == nil && duplicate != nil && duplicate.ID != source.ID {
		return nil, ErrSourceDuplicate
	}
	if duplicateErr != nil && !repository.IsGroundingNotFound(duplicateErr) {
		return nil, duplicateErr
	}
	if err := s.sources.UpdateSource(ctx, source); err != nil {
		return nil, err
	}
	return s.GetSource(ctx, id)
}

func (s *GroundingService) AddEvidence(ctx context.Context, sourceID uint, input EvidenceInput) (*model.SourceEvidence, error) {
	if _, err := s.sources.FindSource(ctx, sourceID); err != nil {
		if repository.IsGroundingNotFound(err) {
			return nil, ErrKnowledgeSourceNotFound
		}
		return nil, err
	}
	if strings.TrimSpace(input.Quote) == "" && strings.TrimSpace(input.Summary) == "" {
		return nil, ErrEvidenceInvalid
	}
	evidence := &model.SourceEvidence{SourceID: sourceID, EvidenceType: strings.TrimSpace(input.EvidenceType), Locator: strings.TrimSpace(input.Locator), Quote: strings.TrimSpace(input.Quote), Summary: strings.TrimSpace(input.Summary), Language: strings.TrimSpace(input.Language), ExtractionMethod: strings.TrimSpace(input.ExtractionMethod), VerificationStatus: strings.TrimSpace(input.VerificationStatus)}
	if evidence.EvidenceType == "" {
		evidence.EvidenceType = model.SourceEvidenceTypeSummary
	}
	if evidence.ExtractionMethod == "" {
		evidence.ExtractionMethod = model.SourceEvidenceExtractionManual
	}
	if evidence.VerificationStatus == "" {
		evidence.VerificationStatus = model.SourceEvidenceVerificationUnverified
	}
	if err := s.sources.CreateEvidence(ctx, evidence); err != nil {
		return nil, err
	}
	return evidence, nil
}

func (s *GroundingService) ListEvidence(ctx context.Context, sourceID uint) ([]model.SourceEvidence, error) {
	if _, err := s.sources.FindSource(ctx, sourceID); err != nil {
		if repository.IsGroundingNotFound(err) {
			return nil, ErrKnowledgeSourceNotFound
		}
		return nil, err
	}
	return s.sources.ListEvidence(ctx, sourceID)
}

func (s *GroundingService) UpdateEvidence(ctx context.Context, id uint, patch EvidencePatch) (*model.SourceEvidence, error) {
	evidence, err := s.sources.FindEvidence(ctx, id)
	if err != nil {
		if repository.IsGroundingNotFound(err) {
			return nil, ErrSourceEvidenceNotFound
		}
		return nil, err
	}
	if patch.EvidenceType != nil {
		evidence.EvidenceType = strings.TrimSpace(*patch.EvidenceType)
	}
	if patch.Locator != nil {
		evidence.Locator = strings.TrimSpace(*patch.Locator)
	}
	if patch.Quote != nil {
		evidence.Quote = strings.TrimSpace(*patch.Quote)
	}
	if patch.Summary != nil {
		evidence.Summary = strings.TrimSpace(*patch.Summary)
	}
	if patch.Language != nil {
		evidence.Language = strings.TrimSpace(*patch.Language)
	}
	if patch.ExtractionMethod != nil {
		evidence.ExtractionMethod = strings.TrimSpace(*patch.ExtractionMethod)
	}
	if patch.VerificationStatus != nil {
		evidence.VerificationStatus = strings.TrimSpace(*patch.VerificationStatus)
	}
	if evidence.Quote == "" && evidence.Summary == "" {
		return nil, ErrEvidenceInvalid
	}
	if err := s.sources.UpdateEvidence(ctx, evidence); err != nil {
		return nil, err
	}
	return evidence, nil
}

func (s *GroundingService) CreateLink(ctx context.Context, input GroundingLinkInput) (*GroundingLinkView, error) {
	if !validTargetType(input.TargetType) || !validRelation(input.Relation) || !validStrength(input.Strength) || input.EvidenceID == 0 || input.TargetID == 0 {
		return nil, ErrGroundingLinkInvalid
	}
	evidence, err := s.sources.FindEvidence(ctx, input.EvidenceID)
	if err != nil {
		if repository.IsGroundingNotFound(err) {
			return nil, ErrSourceEvidenceNotFound
		}
		return nil, err
	}
	if err := s.validateTarget(ctx, input.TargetType, input.TargetID); err != nil {
		return nil, err
	}
	existing, err := s.sources.ListLinksByTarget(ctx, input.TargetType, input.TargetID)
	if err != nil {
		return nil, err
	}
	for _, item := range existing {
		if item.EvidenceID == input.EvidenceID && item.Relation == input.Relation {
			return nil, ErrGroundingLinkConflict
		}
	}
	link := &model.GroundingLink{EvidenceID: input.EvidenceID, TargetType: input.TargetType, TargetID: input.TargetID, Relation: input.Relation, Strength: input.Strength, Rationale: strings.TrimSpace(input.Rationale), Status: model.GroundingLinkStatusProposed, CreatedBy: model.GroundingCreatedByManual}
	if err := s.sources.CreateLink(ctx, link); err != nil {
		return nil, err
	}
	return s.linkView(ctx, *link, *evidence)
}

func (s *GroundingService) ListLinks(ctx context.Context, targetType string, targetID uint) ([]GroundingLinkView, error) {
	if !validTargetType(targetType) || targetID == 0 {
		return nil, ErrGroundingLinkInvalid
	}
	if err := s.validateTarget(ctx, targetType, targetID); err != nil {
		return nil, err
	}
	links, err := s.sources.ListLinksByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	result := make([]GroundingLinkView, 0, len(links))
	for _, link := range links {
		evidence, err := s.sources.FindEvidence(ctx, link.EvidenceID)
		if err != nil {
			return nil, err
		}
		view, err := s.linkView(ctx, link, *evidence)
		if err != nil {
			return nil, err
		}
		result = append(result, *view)
	}
	return result, nil
}

func (s *GroundingService) ListSourceLinks(ctx context.Context, sourceID uint) ([]GroundingLinkView, error) {
	if _, err := s.sources.FindSource(ctx, sourceID); err != nil {
		if repository.IsGroundingNotFound(err) {
			return nil, ErrKnowledgeSourceNotFound
		}
		return nil, err
	}
	links, err := s.sources.ListLinksBySource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	result := make([]GroundingLinkView, 0, len(links))
	for _, link := range links {
		evidence, err := s.sources.FindEvidence(ctx, link.EvidenceID)
		if err != nil {
			return nil, err
		}
		view, err := s.linkView(ctx, link, *evidence)
		if err != nil {
			return nil, err
		}
		result = append(result, *view)
	}
	return result, nil
}

func (s *GroundingService) ReviewLink(ctx context.Context, id uint) (*GroundingTargetView, error) {
	return s.changeLinkStatus(ctx, id, model.GroundingLinkStatusReviewed)
}
func (s *GroundingService) RejectLink(ctx context.Context, id uint) (*GroundingTargetView, error) {
	return s.changeLinkStatus(ctx, id, model.GroundingLinkStatusRejected)
}

func (s *GroundingService) changeLinkStatus(ctx context.Context, id uint, status string) (*GroundingTargetView, error) {
	var result *GroundingTargetView
	err := s.sources.Transaction(ctx, func(tx *repository.GroundingRepository) error {
		link, err := tx.FindLink(ctx, id)
		if err != nil {
			if repository.IsGroundingNotFound(err) {
				return ErrGroundingLinkNotFound
			}
			return err
		}
		now := time.Now()
		if err := tx.UpdateLinkStatus(ctx, id, status, &now); err != nil {
			return err
		}
		result, err = s.recalculateTargetTx(ctx, tx, link.TargetType, link.TargetID, model.GroundingCreatedByManual)
		return err
	})
	return result, err
}

func (s *GroundingService) ListCredibility(ctx context.Context, sourceID uint) ([]model.SourceCredibilityAssessment, error) {
	if _, err := s.sources.FindSource(ctx, sourceID); err != nil {
		if repository.IsGroundingNotFound(err) {
			return nil, ErrKnowledgeSourceNotFound
		}
		return nil, err
	}
	return s.sources.ListCredibility(ctx, sourceID)
}

func (s *GroundingService) CreateCredibility(ctx context.Context, sourceID uint, input CredibilityInput) (*model.SourceCredibilityAssessment, error) {
	if _, err := s.sources.FindSource(ctx, sourceID); err != nil {
		if repository.IsGroundingNotFound(err) {
			return nil, ErrKnowledgeSourceNotFound
		}
		return nil, err
	}
	if !validScores(input.AuthorityScore, input.MethodologyScore, input.DirectnessScore, input.RecencyScore, input.IndependenceScore) {
		return nil, ErrCredibilityInvalid
	}
	method := strings.TrimSpace(input.AssessmentMethod)
	if method == "" {
		method = model.CredibilityAssessmentMethodManual
	}
	assessment := &model.SourceCredibilityAssessment{SourceID: sourceID, AuthorityScore: input.AuthorityScore, MethodologyScore: input.MethodologyScore, DirectnessScore: input.DirectnessScore, RecencyScore: input.RecencyScore, IndependenceScore: input.IndependenceScore, OverallScore: input.AuthorityScore + input.MethodologyScore + input.DirectnessScore + input.RecencyScore + input.IndependenceScore, AuthorityReason: strings.TrimSpace(input.AuthorityReason), MethodologyReason: strings.TrimSpace(input.MethodologyReason), DirectnessReason: strings.TrimSpace(input.DirectnessReason), RecencyReason: strings.TrimSpace(input.RecencyReason), IndependenceReason: strings.TrimSpace(input.IndependenceReason), AssessmentMethod: method, Status: model.CredibilityAssessmentStatusDraft}
	if err := s.sources.CreateCredibility(ctx, assessment); err != nil {
		return nil, err
	}
	return assessment, nil
}

func (s *GroundingService) ReviewCredibility(ctx context.Context, id uint) (*model.SourceCredibilityAssessment, error) {
	var result *model.SourceCredibilityAssessment
	err := s.sources.Transaction(ctx, func(tx *repository.GroundingRepository) error {
		assessment, err := tx.FindCredibility(ctx, id)
		if err != nil {
			if repository.IsGroundingNotFound(err) {
				return ErrCredibilityNotFound
			}
			return err
		}
		now := time.Now()
		if err := tx.ReviewCredibility(ctx, id, now); err != nil {
			return err
		}
		links, err := tx.ListLinksBySource(ctx, assessment.SourceID)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, link := range links {
			key := link.TargetType + ":" + fmt.Sprint(link.TargetID)
			if seen[key] {
				continue
			}
			seen[key] = true
			if _, err := s.recalculateTargetTx(ctx, tx, link.TargetType, link.TargetID, model.GroundingCreatedByManual); err != nil {
				return err
			}
		}
		result, err = tx.FindCredibility(ctx, id)
		return err
	})
	return result, err
}

func (s *GroundingService) GetTarget(ctx context.Context, targetType string, targetID uint) (*GroundingTargetView, error) {
	if err := s.validateTarget(ctx, targetType, targetID); err != nil {
		return nil, err
	}
	links, err := s.sources.ListLinksByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	status, err := s.targetStatus(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	return s.buildTargetView(ctx, targetType, targetID, status, links)
}

func (s *GroundingService) GetCoverage(ctx context.Context, courseID uint) (*GroundingCoverageView, error) {
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return nil, mapGroundingCourseError(err)
	}
	blueprint, err := s.curriculum.FindActiveBlueprint(ctx, courseID)
	if err != nil {
		if repository.IsCurriculumNotFound(err) {
			return nil, ErrCurriculumBlueprintNotFound
		}
		return nil, err
	}
	units, err := s.curriculum.ListBlueprintUnits(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	blueprintLessons, err := s.curriculum.ListBlueprintLessons(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	courseLessons, err := s.curriculum.ListCourseLessons(ctx, courseID)
	if err != nil {
		return nil, err
	}
	lessonByID := map[uint]model.Lesson{}
	for _, lesson := range courseLessons {
		lessonByID[lesson.ID] = lesson
	}
	byUnit := map[uint][]model.CurriculumBlueprintLesson{}
	for _, lesson := range blueprintLessons {
		byUnit[lesson.BlueprintUnitID] = append(byUnit[lesson.BlueprintUnitID], lesson)
	}
	view := &GroundingCoverageView{Blueprint: *blueprint, Units: make([]GroundingCoverageUnit, 0, len(units))}
	for _, unit := range units {
		coverageUnit := GroundingCoverageUnit{Unit: unit, Lessons: []GroundingCoverageLesson{}}
		for _, lesson := range byUnit[unit.ID] {
			status, err := s.targetStatus(ctx, model.GroundingTargetBlueprintLesson, lesson.ID)
			if err != nil {
				return nil, err
			}
			links, err := s.sources.ListLinksByTarget(ctx, model.GroundingTargetBlueprintLesson, lesson.ID)
			if err != nil {
				return nil, err
			}
			stats := linkStats(ctx, s.sources, links)
			item := GroundingCoverageLesson{BlueprintLesson: lesson, GroundingStatus: status, ReviewedSourceCount: stats.sources, ReviewedEvidenceCount: stats.evidence, ConflictCount: stats.conflicts}
			if lesson.AppliedLessonID != nil {
				if applied, ok := lessonByID[*lesson.AppliedLessonID]; ok {
					appliedLinks, _ := s.sources.ListLinksByTarget(ctx, model.GroundingTargetLesson, applied.ID)
					appliedStats := linkStats(ctx, s.sources, appliedLinks)
					appliedStatus, statusErr := s.targetStatus(ctx, model.GroundingTargetLesson, applied.ID)
					if statusErr != nil {
						return nil, statusErr
					}
					item.AppliedLesson = &LessonGroundingSummary{ID: applied.ID, Title: applied.Title, GroundingStatus: appliedStatus, ReviewedSourceCount: appliedStats.sources}
				}
			}
			coverageUnit.Lessons = append(coverageUnit.Lessons, item)
			updateGroundingMetrics(&view.Metrics, lesson.Importance, status)
		}
		view.Units = append(view.Units, coverageUnit)
	}
	if view.Metrics.CoreTotal > 0 {
		view.Metrics.GroundingPercent = float64(view.Metrics.CoreGrounded) * 100 / float64(view.Metrics.CoreTotal)
	}
	return view, nil
}

type linkStat struct{ sources, evidence, conflicts int }

func linkStats(ctx context.Context, repo *repository.GroundingRepository, links []model.GroundingLink) linkStat {
	result := linkStat{}
	sourceIDs := map[uint]bool{}
	for _, link := range links {
		if link.Status != model.GroundingLinkStatusReviewed {
			continue
		}
		result.evidence++
		if link.Relation == model.GroundingRelationContradicts {
			result.conflicts++
		}
		if evidence, err := repo.FindEvidence(ctx, link.EvidenceID); err == nil {
			sourceIDs[evidence.SourceID] = true
		}
	}
	result.sources = len(sourceIDs)
	return result
}

func (s *GroundingService) recalculateTargetTx(ctx context.Context, tx *repository.GroundingRepository, targetType string, targetID uint, triggeredBy string) (*GroundingTargetView, error) {
	if err := s.validateTargetTx(ctx, tx, targetType, targetID); err != nil {
		return nil, err
	}
	old, err := s.targetStatusTx(ctx, tx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	status, err := deriveGroundingStatus(ctx, tx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	if err := updateTargetStatus(ctx, tx, targetType, targetID, status); err != nil {
		return nil, err
	}
	if old != status {
		if err := tx.CreateReviewEvent(ctx, &model.GroundingReviewEvent{TargetType: targetType, TargetID: targetID, FromStatus: old, ToStatus: status, Reason: "recalculated from reviewed source evidence and credibility", TriggeredBy: triggeredBy}); err != nil {
			return nil, err
		}
	}
	if targetType == model.GroundingTargetBlueprintLesson {
		lesson, err := s.curriculum.FindBlueprintLesson(ctx, targetID)
		if err != nil {
			return nil, err
		}
		if _, err := s.recalculateBlueprintAggregateTx(ctx, tx, lesson.BlueprintID, triggeredBy); err != nil {
			return nil, err
		}
	}
	if targetType == model.GroundingTargetLesson {
		lessons, err := s.sources.ListBlueprintLessonsByAppliedLesson(ctx, targetID)
		if err != nil {
			return nil, err
		}
		seen := map[uint]bool{}
		for _, lesson := range lessons {
			if seen[lesson.BlueprintID] {
				continue
			}
			seen[lesson.BlueprintID] = true
			if _, err := s.recalculateBlueprintAggregateTx(ctx, tx, lesson.BlueprintID, triggeredBy); err != nil {
				return nil, err
			}
		}
	}
	return s.buildTargetViewTx(ctx, tx, targetType, targetID, status)
}

func (s *GroundingService) recalculateBlueprintAggregateTx(ctx context.Context, tx *repository.GroundingRepository, blueprintID uint, triggeredBy string) (string, error) {
	blueprint, err := s.curriculum.FindBlueprintByID(ctx, blueprintID)
	if err != nil {
		return "", err
	}
	old := blueprint.GroundingStatus
	direct, err := deriveGroundingStatus(ctx, tx, model.GroundingTargetCurriculumBlueprint, blueprintID)
	if err != nil {
		return "", err
	}
	lessons, err := s.curriculum.ListBlueprintLessons(ctx, blueprintID)
	if err != nil {
		return "", err
	}
	status := model.CurriculumGroundingUngrounded
	hasSupport := direct != model.CurriculumGroundingUngrounded
	allCoreGrounded := true
	coreCount := 0
	if direct == model.CurriculumGroundingConflicted {
		status = model.CurriculumGroundingConflicted
	}
	for _, lesson := range lessons {
		childStatus, err := deriveGroundingStatus(ctx, tx, model.GroundingTargetBlueprintLesson, lesson.ID)
		if err != nil {
			return "", err
		}
		if childStatus == model.CurriculumGroundingConflicted {
			status = model.CurriculumGroundingConflicted
		}
		if childStatus != model.CurriculumGroundingUngrounded {
			hasSupport = true
		}
		if lesson.Importance == model.CurriculumImportanceCore {
			coreCount++
			if childStatus != model.CurriculumGroundingGrounded {
				allCoreGrounded = false
			}
		}
	}
	if status != model.CurriculumGroundingConflicted {
		if coreCount > 0 && allCoreGrounded {
			status = model.CurriculumGroundingGrounded
		} else if hasSupport {
			status = model.CurriculumGroundingPartially
		}
	}
	if err := tx.UpdateBlueprintGroundingStatus(ctx, blueprintID, status); err != nil {
		return "", err
	}
	if old != status {
		if err := tx.CreateReviewEvent(ctx, &model.GroundingReviewEvent{TargetType: model.GroundingTargetCurriculumBlueprint, TargetID: blueprintID, FromStatus: old, ToStatus: status, Reason: "aggregated blueprint lesson grounding", TriggeredBy: triggeredBy}); err != nil {
			return "", err
		}
	}
	return status, nil
}

func (s *GroundingService) targetStatus(ctx context.Context, targetType string, targetID uint) (string, error) {
	return deriveGroundingStatus(ctx, s.sources, targetType, targetID)
}
func (s *GroundingService) targetStatusTx(ctx context.Context, tx *repository.GroundingRepository, targetType string, targetID uint) (string, error) {
	return storedTargetStatus(ctx, tx, s.curriculum, targetType, targetID)
}

func (s *GroundingService) validateTarget(ctx context.Context, targetType string, targetID uint) error {
	return s.validateTargetTx(ctx, s.sources, targetType, targetID)
}
func (s *GroundingService) validateTargetTx(ctx context.Context, _ *repository.GroundingRepository, targetType string, targetID uint) error {
	if !validTargetType(targetType) || targetID == 0 {
		return ErrGroundingTargetNotFound
	}
	switch targetType {
	case model.GroundingTargetCurriculumBlueprint:
		if _, err := s.curriculum.FindBlueprintByID(ctx, targetID); err != nil {
			return ErrGroundingTargetNotFound
		}
	case model.GroundingTargetBlueprintLesson:
		if _, err := s.curriculum.FindBlueprintLesson(ctx, targetID); err != nil {
			return ErrGroundingTargetNotFound
		}
	case model.GroundingTargetLesson:
		if _, err := s.curriculum.FindCourseLesson(ctx, targetID); err != nil {
			return ErrGroundingTargetNotFound
		}
	}
	return nil
}

func (s *GroundingService) buildTargetView(ctx context.Context, targetType string, targetID uint, status string, links []model.GroundingLink) (*GroundingTargetView, error) {
	return s.buildTargetViewTx(ctx, s.sources, targetType, targetID, status)
}
func (s *GroundingService) buildTargetViewTx(ctx context.Context, repo *repository.GroundingRepository, targetType string, targetID uint, status string) (*GroundingTargetView, error) {
	links, err := repo.ListLinksByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	result := &GroundingTargetView{TargetType: targetType, TargetID: targetID, GroundingStatus: status, Links: []GroundingLinkView{}}
	seenSources := map[uint]bool{}
	for _, link := range links {
		evidence, err := repo.FindEvidence(ctx, link.EvidenceID)
		if err != nil {
			return nil, err
		}
		view, err := s.linkViewWithRepo(ctx, repo, link, *evidence)
		if err != nil {
			return nil, err
		}
		result.Links = append(result.Links, *view)
		if link.Status == model.GroundingLinkStatusReviewed {
			result.ReviewedEvidenceCount++
			if link.Relation == model.GroundingRelationContradicts {
				result.ConflictCount++
			}
			seenSources[evidence.SourceID] = true
		}
	}
	result.ReviewedSourceCount = len(seenSources)
	result.ReviewEvents, err = repo.ListReviewEvents(ctx, targetType, targetID)
	return result, err
}

func (s *GroundingService) linkView(ctx context.Context, link model.GroundingLink, evidence model.SourceEvidence) (*GroundingLinkView, error) {
	return s.linkViewWithRepo(ctx, s.sources, link, evidence)
}
func (s *GroundingService) linkViewWithRepo(ctx context.Context, repo *repository.GroundingRepository, link model.GroundingLink, evidence model.SourceEvidence) (*GroundingLinkView, error) {
	source, err := repo.FindSource(ctx, evidence.SourceID)
	if err != nil {
		return nil, err
	}
	credibility, _ := repo.FindLatestReviewedCredibility(ctx, source.ID)
	return &GroundingLinkView{Link: link, Evidence: evidence, Source: *source, Credibility: credibility}, nil
}

func deriveGroundingStatus(ctx context.Context, repo *repository.GroundingRepository, targetType string, targetID uint) (string, error) {
	links, err := repo.ListLinksByTarget(ctx, targetType, targetID)
	if err != nil {
		return "", err
	}
	hasSupport, hasQualified, hasContradiction := false, false, false
	for _, link := range links {
		if link.Status != model.GroundingLinkStatusReviewed {
			continue
		}
		if link.Relation == model.GroundingRelationContradicts {
			hasContradiction = true
		}
		if link.Relation != model.GroundingRelationSupports {
			continue
		}
		hasSupport = true
		evidence, err := repo.FindEvidence(ctx, link.EvidenceID)
		if err != nil {
			return "", err
		}
		credibility, err := repo.FindLatestReviewedCredibility(ctx, evidence.SourceID)
		if err == nil && credibility.OverallScore >= 60 && (link.Strength == model.GroundingStrengthModerate || link.Strength == model.GroundingStrengthStrong) {
			hasQualified = true
		}
	}
	if hasSupport && hasContradiction {
		return model.CurriculumGroundingConflicted, nil
	}
	if hasQualified {
		return model.CurriculumGroundingGrounded, nil
	}
	if hasSupport {
		return model.CurriculumGroundingPartially, nil
	}
	return model.CurriculumGroundingUngrounded, nil
}

func storedTargetStatus(ctx context.Context, repo *repository.GroundingRepository, curriculum *repository.CurriculumRepository, targetType string, targetID uint) (string, error) {
	switch targetType {
	case model.GroundingTargetCurriculumBlueprint:
		item, err := curriculum.FindBlueprintByID(ctx, targetID)
		if err != nil {
			return "", err
		}
		return normalizeGroundingStatus(item.GroundingStatus), nil
	case model.GroundingTargetBlueprintLesson:
		item, err := curriculum.FindBlueprintLesson(ctx, targetID)
		if err != nil {
			return "", err
		}
		return normalizeGroundingStatus(item.GroundingStatus), nil
	case model.GroundingTargetLesson:
		item, err := curriculum.FindCourseLesson(ctx, targetID)
		if err != nil {
			return "", err
		}
		return normalizeGroundingStatus(item.GroundingStatus), nil
	}
	return "", ErrGroundingTargetNotFound
}

func updateTargetStatus(ctx context.Context, repo *repository.GroundingRepository, targetType string, targetID uint, status string) error {
	switch targetType {
	case model.GroundingTargetCurriculumBlueprint:
		return repo.UpdateBlueprintGroundingStatus(ctx, targetID, status)
	case model.GroundingTargetBlueprintLesson:
		return repo.UpdateBlueprintLessonGroundingStatus(ctx, targetID, status)
	case model.GroundingTargetLesson:
		return repo.UpdateLessonGroundingStatus(ctx, targetID, status)
	}
	return ErrGroundingTargetNotFound
}

func updateGroundingMetrics(metrics *GroundingCoverageMetrics, importance, status string) {
	if importance == model.CurriculumImportanceCore {
		metrics.CoreTotal++
		switch status {
		case model.CurriculumGroundingGrounded:
			metrics.CoreGrounded++
		case model.CurriculumGroundingPartially:
			metrics.CorePartial++
		case model.CurriculumGroundingConflicted:
			metrics.CoreConflicted++
		default:
			metrics.CoreUngrounded++
		}
	} else if importance == model.CurriculumImportanceRecommended {
		metrics.RecommendedTotal++
		if status == model.CurriculumGroundingGrounded {
			metrics.RecommendedGrounded++
		}
	}
}
func validTargetType(value string) bool {
	return value == model.GroundingTargetCurriculumBlueprint || value == model.GroundingTargetBlueprintLesson || value == model.GroundingTargetLesson
}
func validRelation(value string) bool {
	return value == model.GroundingRelationSupports || value == model.GroundingRelationContradicts || value == model.GroundingRelationContextualizes || value == model.GroundingRelationLimits
}
func validStrength(value string) bool {
	return value == model.GroundingStrengthWeak || value == model.GroundingStrengthModerate || value == model.GroundingStrengthStrong
}
func validScores(values ...int) bool {
	for _, value := range values {
		if value < 0 || value > 20 {
			return false
		}
	}
	return true
}
func normalizeGroundingStatus(status string) string {
	if status == model.CurriculumGroundingProvisional || status == "" {
		return model.CurriculumGroundingUngrounded
	}
	return status
}
func applySourceDefaults(source *model.KnowledgeSource) {
	if source.SourceType == "" {
		source.SourceType = model.KnowledgeSourceTypeOther
	}
	if source.AccessStatus == "" {
		source.AccessStatus = model.KnowledgeSourceAccessMetadataOnly
	}
	if source.VerificationStatus == "" {
		source.VerificationStatus = model.KnowledgeSourceVerificationUnverified
	}
	if source.CreatedBy == "" {
		source.CreatedBy = model.KnowledgeSourceCreatedByManual
	}
}
func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(strings.TrimRight(raw, "/"))
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return strings.TrimRight(parsed.String(), "/")
}
func mapGroundingCourseError(err error) error {
	if repository.IsCurriculumNotFound(err) {
		return ErrCourseNotFound
	}
	return err
}
