package model

import "time"

const (
	KnowledgeSourceTypeTextbook         = "textbook"
	KnowledgeSourceTypeGuideline        = "guideline"
	KnowledgeSourceTypeConsensus        = "consensus"
	KnowledgeSourceTypeSystematicReview = "systematic_review"
	KnowledgeSourceTypeReview           = "review"
	KnowledgeSourceTypeResearchPaper    = "research_paper"
	KnowledgeSourceTypeOfficialWeb      = "official_web"
	KnowledgeSourceTypeReferenceWork    = "reference_work"
	KnowledgeSourceTypeCourseMaterial   = "course_material"
	KnowledgeSourceTypeUserNote         = "user_note"
	KnowledgeSourceTypeOther            = "other"
)

const (
	KnowledgeSourceAccessMetadataOnly     = "metadata_only"
	KnowledgeSourceAccessExcerptAvailable = "excerpt_available"
	KnowledgeSourceAccessFullText         = "fulltext_available"
)

const (
	KnowledgeSourceVerificationUnverified = "unverified"
	KnowledgeSourceVerificationMetadata   = "metadata_verified"
	KnowledgeSourceVerificationReviewed   = "reviewed"
	SourceEvidenceVerificationUnverified  = "unverified"
	SourceEvidenceVerificationReviewed    = "reviewed"
)

const (
	KnowledgeSourceCreatedByManual = "manual"
	KnowledgeSourceCreatedBySeed   = "seed"
	KnowledgeSourceCreatedByImport = "import"
	KnowledgeSourceCreatedByAI     = "ai_assisted"
)

const (
	SourceEvidenceTypeExcerpt        = "excerpt"
	SourceEvidenceTypeSummary        = "summary"
	SourceEvidenceTypeTable          = "table"
	SourceEvidenceTypeFigure         = "figure"
	SourceEvidenceTypeRecommendation = "recommendation"
	SourceEvidenceTypeDefinition     = "definition"
	SourceEvidenceTypeFinding        = "finding"
	SourceEvidenceTypeMethodology    = "methodology"
)

const (
	SourceEvidenceExtractionManual = "manual"
	SourceEvidenceExtractionAI     = "ai_assisted"
)

const (
	GroundingTargetCurriculumBlueprint = "curriculum_blueprint"
	GroundingTargetBlueprintLesson     = "blueprint_lesson"
	GroundingTargetLesson              = "lesson"
)

const (
	GroundingRelationSupports       = "supports"
	GroundingRelationContradicts    = "contradicts"
	GroundingRelationContextualizes = "contextualizes"
	GroundingRelationLimits         = "limits"
)

const (
	GroundingStrengthWeak     = "weak"
	GroundingStrengthModerate = "moderate"
	GroundingStrengthStrong   = "strong"
)

const (
	GroundingLinkStatusProposed = "proposed"
	GroundingLinkStatusReviewed = "reviewed"
	GroundingLinkStatusRejected = "rejected"
)

const (
	GroundingCreatedByManual = "manual"
	GroundingCreatedByAI     = "ai_assisted"
)

const (
	CredibilityAssessmentMethodManual = "manual"
	CredibilityAssessmentMethodRule   = "rule"
	CredibilityAssessmentMethodAI     = "ai_assisted"
)

const (
	CredibilityAssessmentStatusDraft    = "draft"
	CredibilityAssessmentStatusReviewed = "reviewed"
	CredibilityAssessmentStatusStale    = "stale"
)

type KnowledgeSource struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	Title              string     `gorm:"size:500;not null" json:"title"`
	Authors            string     `gorm:"type:text" json:"authors"`
	Organization       string     `gorm:"size:255" json:"organization"`
	SourceType         string     `gorm:"size:64;not null;index" json:"source_type"`
	Publisher          string     `gorm:"size:255" json:"publisher"`
	PublicationYear    *int       `json:"publication_year"`
	PublishedAt        *time.Time `json:"published_at"`
	UpdatedAtSource    *time.Time `json:"updated_at_source"`
	URL                string     `gorm:"size:2048" json:"url"`
	NormalizedURL      string     `gorm:"size:2048;index" json:"-"`
	DOI                string     `gorm:"size:255;index" json:"doi"`
	ISBN               string     `gorm:"size:64;index" json:"isbn"`
	Language           string     `gorm:"size:32" json:"language"`
	Description        string     `gorm:"type:text" json:"description"`
	AccessStatus       string     `gorm:"size:32;not null;default:'metadata_only'" json:"access_status"`
	VerificationStatus string     `gorm:"size:32;not null;default:'unverified';index" json:"verification_status"`
	CreatedBy          string     `gorm:"size:32;not null;default:'manual'" json:"created_by"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type SourceEvidence struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	SourceID           uint      `gorm:"not null;index" json:"source_id"`
	EvidenceType       string    `gorm:"size:64;not null" json:"evidence_type"`
	Locator            string    `gorm:"size:500" json:"locator"`
	Quote              string    `gorm:"type:text" json:"quote"`
	Summary            string    `gorm:"type:text" json:"summary"`
	Language           string    `gorm:"size:32" json:"language"`
	ExtractionMethod   string    `gorm:"size:32;not null;default:'manual'" json:"extraction_method"`
	VerificationStatus string    `gorm:"size:32;not null;default:'unverified';index" json:"verification_status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type GroundingLink struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	EvidenceID uint       `gorm:"not null;index;uniqueIndex:idx_grounding_link_identity" json:"evidence_id"`
	TargetType string     `gorm:"size:64;not null;index;uniqueIndex:idx_grounding_link_identity" json:"target_type"`
	TargetID   uint       `gorm:"not null;index;uniqueIndex:idx_grounding_link_identity" json:"target_id"`
	Relation   string     `gorm:"size:32;not null;uniqueIndex:idx_grounding_link_identity" json:"relation"`
	Strength   string     `gorm:"size:32;not null" json:"strength"`
	Rationale  string     `gorm:"type:text" json:"rationale"`
	Status     string     `gorm:"size:32;not null;default:'proposed';index" json:"status"`
	CreatedBy  string     `gorm:"size:32;not null;default:'manual'" json:"created_by"`
	ReviewedAt *time.Time `json:"reviewed_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type SourceCredibilityAssessment struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	SourceID           uint       `gorm:"not null;index" json:"source_id"`
	AuthorityScore     int        `gorm:"not null" json:"authority_score"`
	MethodologyScore   int        `gorm:"not null" json:"methodology_score"`
	DirectnessScore    int        `gorm:"not null" json:"directness_score"`
	RecencyScore       int        `gorm:"not null" json:"recency_score"`
	IndependenceScore  int        `gorm:"not null" json:"independence_score"`
	OverallScore       int        `gorm:"not null;index" json:"overall_score"`
	AuthorityReason    string     `gorm:"type:text" json:"authority_reason"`
	MethodologyReason  string     `gorm:"type:text" json:"methodology_reason"`
	DirectnessReason   string     `gorm:"type:text" json:"directness_reason"`
	RecencyReason      string     `gorm:"type:text" json:"recency_reason"`
	IndependenceReason string     `gorm:"type:text" json:"independence_reason"`
	AssessmentMethod   string     `gorm:"size:32;not null" json:"assessment_method"`
	Status             string     `gorm:"size:32;not null;index" json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	ReviewedAt         *time.Time `json:"reviewed_at"`
}

type GroundingReviewEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TargetType  string    `gorm:"size:64;not null;index" json:"target_type"`
	TargetID    uint      `gorm:"not null;index" json:"target_id"`
	FromStatus  string    `gorm:"size:32;not null" json:"from_status"`
	ToStatus    string    `gorm:"size:32;not null" json:"to_status"`
	Reason      string    `gorm:"type:text" json:"reason"`
	TriggeredBy string    `gorm:"size:32;not null" json:"triggered_by"`
	CreatedAt   time.Time `json:"created_at"`
}
