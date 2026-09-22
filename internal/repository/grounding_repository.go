package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type GroundingRepository struct{ db *gorm.DB }

func NewGroundingRepository(db *gorm.DB) *GroundingRepository { return &GroundingRepository{db: db} }

func (r *GroundingRepository) Transaction(ctx context.Context, fn func(*GroundingRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&GroundingRepository{db: tx}) })
}

func (r *GroundingRepository) FindSource(ctx context.Context, id uint) (*model.KnowledgeSource, error) {
	var source model.KnowledgeSource
	if err := r.db.WithContext(ctx).First(&source, id).Error; err != nil {
		return nil, fmt.Errorf("find knowledge source: %w", err)
	}
	return &source, nil
}

func (r *GroundingRepository) ListSources(ctx context.Context, query, sourceType string, limit int) ([]model.KnowledgeSource, error) {
	var sources []model.KnowledgeSource
	db := r.db.WithContext(ctx).Order("updated_at DESC, id DESC")
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("title LIKE ? OR authors LIKE ? OR organization LIKE ?", like, like, like)
	}
	if sourceType != "" {
		db = db.Where("source_type = ?", sourceType)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err := db.Find(&sources).Error; err != nil {
		return nil, fmt.Errorf("list knowledge sources: %w", err)
	}
	return sources, nil
}

func (r *GroundingRepository) FindDuplicateSource(ctx context.Context, source *model.KnowledgeSource) (*model.KnowledgeSource, error) {
	var found model.KnowledgeSource
	db := r.db.WithContext(ctx)
	if source.DOI != "" {
		if err := db.Where("doi = ?", source.DOI).First(&found).Error; err == nil {
			return &found, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find source by doi: %w", err)
		}
	}
	if source.ISBN != "" {
		if err := db.Where("isbn = ?", source.ISBN).First(&found).Error; err == nil {
			return &found, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find source by isbn: %w", err)
		}
	}
	if source.NormalizedURL != "" {
		if err := db.Where("normalized_url = ?", source.NormalizedURL).First(&found).Error; err == nil {
			return &found, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find source by url: %w", err)
		}
	}
	if source.PublicationYear != nil {
		if err := db.Where("title = ? AND organization = ? AND publication_year = ?", source.Title, source.Organization, *source.PublicationYear).First(&found).Error; err == nil {
			return &found, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find source by title: %w", err)
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *GroundingRepository) CreateSource(ctx context.Context, source *model.KnowledgeSource) error {
	if err := r.db.WithContext(ctx).Create(source).Error; err != nil {
		return fmt.Errorf("create knowledge source: %w", err)
	}
	return nil
}

func (r *GroundingRepository) UpdateSource(ctx context.Context, source *model.KnowledgeSource) error {
	if err := r.db.WithContext(ctx).Model(&model.KnowledgeSource{}).Where("id = ?", source.ID).Updates(map[string]interface{}{
		"title": source.Title, "authors": source.Authors, "organization": source.Organization, "source_type": source.SourceType,
		"publisher": source.Publisher, "publication_year": source.PublicationYear, "published_at": source.PublishedAt,
		"updated_at_source": source.UpdatedAtSource, "url": source.URL, "normalized_url": source.NormalizedURL,
		"doi": source.DOI, "isbn": source.ISBN, "language": source.Language, "description": source.Description,
		"access_status": source.AccessStatus, "verification_status": source.VerificationStatus,
	}).Error; err != nil {
		return fmt.Errorf("update knowledge source: %w", err)
	}
	return nil
}

func (r *GroundingRepository) FindEvidence(ctx context.Context, id uint) (*model.SourceEvidence, error) {
	var evidence model.SourceEvidence
	if err := r.db.WithContext(ctx).First(&evidence, id).Error; err != nil {
		return nil, fmt.Errorf("find source evidence: %w", err)
	}
	return &evidence, nil
}

func (r *GroundingRepository) ListEvidence(ctx context.Context, sourceID uint) ([]model.SourceEvidence, error) {
	var evidence []model.SourceEvidence
	if err := r.db.WithContext(ctx).Where("source_id = ?", sourceID).Order("created_at DESC, id DESC").Find(&evidence).Error; err != nil {
		return nil, fmt.Errorf("list source evidence: %w", err)
	}
	return evidence, nil
}

func (r *GroundingRepository) CreateEvidence(ctx context.Context, evidence *model.SourceEvidence) error {
	if err := r.db.WithContext(ctx).Create(evidence).Error; err != nil {
		return fmt.Errorf("create source evidence: %w", err)
	}
	return nil
}

func (r *GroundingRepository) UpdateEvidence(ctx context.Context, evidence *model.SourceEvidence) error {
	if err := r.db.WithContext(ctx).Model(&model.SourceEvidence{}).Where("id = ?", evidence.ID).Updates(map[string]interface{}{
		"evidence_type": evidence.EvidenceType, "locator": evidence.Locator, "quote": evidence.Quote,
		"summary": evidence.Summary, "language": evidence.Language, "extraction_method": evidence.ExtractionMethod,
		"verification_status": evidence.VerificationStatus,
	}).Error; err != nil {
		return fmt.Errorf("update source evidence: %w", err)
	}
	return nil
}

func (r *GroundingRepository) FindLink(ctx context.Context, id uint) (*model.GroundingLink, error) {
	var link model.GroundingLink
	if err := r.db.WithContext(ctx).First(&link, id).Error; err != nil {
		return nil, fmt.Errorf("find grounding link: %w", err)
	}
	return &link, nil
}

func (r *GroundingRepository) ListLinksByTarget(ctx context.Context, targetType string, targetID uint) ([]model.GroundingLink, error) {
	var links []model.GroundingLink
	if err := r.db.WithContext(ctx).Where("target_type = ? AND target_id = ?", targetType, targetID).Order("created_at DESC, id DESC").Find(&links).Error; err != nil {
		return nil, fmt.Errorf("list grounding links by target: %w", err)
	}
	return links, nil
}

func (r *GroundingRepository) ListLinksBySource(ctx context.Context, sourceID uint) ([]model.GroundingLink, error) {
	var links []model.GroundingLink
	if err := r.db.WithContext(ctx).Table("grounding_links AS gl").Joins("JOIN source_evidences AS se ON se.id = gl.evidence_id").Where("se.source_id = ?", sourceID).Order("gl.id ASC").Find(&links).Error; err != nil {
		return nil, fmt.Errorf("list grounding links by source: %w", err)
	}
	return links, nil
}

func (r *GroundingRepository) CreateLink(ctx context.Context, link *model.GroundingLink) error {
	if err := r.db.WithContext(ctx).Create(link).Error; err != nil {
		return fmt.Errorf("create grounding link: %w", err)
	}
	return nil
}

func (r *GroundingRepository) UpdateLinkStatus(ctx context.Context, id uint, status string, reviewedAt *time.Time) error {
	updates := map[string]interface{}{"status": status, "updated_at": time.Now()}
	if reviewedAt != nil {
		updates["reviewed_at"] = reviewedAt
	}
	if err := r.db.WithContext(ctx).Model(&model.GroundingLink{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("update grounding link status: %w", err)
	}
	return nil
}

func (r *GroundingRepository) ListCredibility(ctx context.Context, sourceID uint) ([]model.SourceCredibilityAssessment, error) {
	var assessments []model.SourceCredibilityAssessment
	if err := r.db.WithContext(ctx).Where("source_id = ?", sourceID).Order("created_at DESC, id DESC").Find(&assessments).Error; err != nil {
		return nil, fmt.Errorf("list source credibility: %w", err)
	}
	return assessments, nil
}

func (r *GroundingRepository) FindCredibility(ctx context.Context, id uint) (*model.SourceCredibilityAssessment, error) {
	var assessment model.SourceCredibilityAssessment
	if err := r.db.WithContext(ctx).First(&assessment, id).Error; err != nil {
		return nil, fmt.Errorf("find source credibility: %w", err)
	}
	return &assessment, nil
}

func (r *GroundingRepository) FindLatestReviewedCredibility(ctx context.Context, sourceID uint) (*model.SourceCredibilityAssessment, error) {
	var assessment model.SourceCredibilityAssessment
	if err := r.db.WithContext(ctx).Where("source_id = ? AND status = ?", sourceID, model.CredibilityAssessmentStatusReviewed).Order("created_at DESC, id DESC").First(&assessment).Error; err != nil {
		return nil, fmt.Errorf("find reviewed source credibility: %w", err)
	}
	return &assessment, nil
}

func (r *GroundingRepository) CreateCredibility(ctx context.Context, assessment *model.SourceCredibilityAssessment) error {
	if err := r.db.WithContext(ctx).Create(assessment).Error; err != nil {
		return fmt.Errorf("create source credibility: %w", err)
	}
	return nil
}

func (r *GroundingRepository) ReviewCredibility(ctx context.Context, id uint, reviewedAt time.Time) error {
	if err := r.db.WithContext(ctx).Model(&model.SourceCredibilityAssessment{}).Where("id = ?", id).Updates(map[string]interface{}{"status": model.CredibilityAssessmentStatusReviewed, "reviewed_at": reviewedAt, "updated_at": reviewedAt}).Error; err != nil {
		return fmt.Errorf("review source credibility: %w", err)
	}
	return nil
}

func (r *GroundingRepository) CreateReviewEvent(ctx context.Context, event *model.GroundingReviewEvent) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create grounding review event: %w", err)
	}
	return nil
}

func (r *GroundingRepository) ListReviewEvents(ctx context.Context, targetType string, targetID uint) ([]model.GroundingReviewEvent, error) {
	var events []model.GroundingReviewEvent
	if err := r.db.WithContext(ctx).Where("target_type = ? AND target_id = ?", targetType, targetID).Order("created_at DESC, id DESC").Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list grounding review events: %w", err)
	}
	return events, nil
}

func (r *GroundingRepository) UpdateBlueprintGroundingStatus(ctx context.Context, id uint, status string) error {
	return r.updateGroundingStatus(ctx, &model.CurriculumBlueprint{}, id, status, "curriculum blueprint")
}

func (r *GroundingRepository) UpdateBlueprintLessonGroundingStatus(ctx context.Context, id uint, status string) error {
	return r.updateGroundingStatus(ctx, &model.CurriculumBlueprintLesson{}, id, status, "blueprint lesson")
}

func (r *GroundingRepository) UpdateLessonGroundingStatus(ctx context.Context, id uint, status string) error {
	return r.updateGroundingStatus(ctx, &model.Lesson{}, id, status, "lesson")
}

func (r *GroundingRepository) updateGroundingStatus(ctx context.Context, value interface{}, id uint, status, label string) error {
	if err := r.db.WithContext(ctx).Model(value).Where("id = ?", id).Updates(map[string]interface{}{"grounding_status": status, "updated_at": time.Now()}).Error; err != nil {
		return fmt.Errorf("update %s grounding status: %w", label, err)
	}
	return nil
}

func (r *GroundingRepository) ListBlueprintLessonsByAppliedLesson(ctx context.Context, lessonID uint) ([]model.CurriculumBlueprintLesson, error) {
	var lessons []model.CurriculumBlueprintLesson
	if err := r.db.WithContext(ctx).Where("applied_lesson_id = ?", lessonID).Find(&lessons).Error; err != nil {
		return nil, fmt.Errorf("list blueprint lessons by applied lesson: %w", err)
	}
	return lessons, nil
}

func IsGroundingNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
