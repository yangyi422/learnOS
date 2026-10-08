package service

import (
	"context"
	"errors"
	"strings"

	"learnos/internal/links"
	"learnos/internal/model"
	"learnos/internal/repository"
)

var ErrInvalidRecordInput = errors.New("invalid record input")

type RecordService struct{ records *repository.RecordRepository }

func NewRecordService(records *repository.RecordRepository) *RecordService {
	return &RecordService{records: records}
}

type RecordInput struct {
	CreationKey string `json:"creation_key"`
	Content     string `json:"content"`
	ProjectID   *uint  `json:"project_id"`
	ExternalURL string `json:"external_url"`
	LinkName    string `json:"link_name"`
}

func normalizeRecord(input RecordInput) (model.LightweightRecord, error) {
	input.Content = strings.TrimSpace(input.Content)
	input.ExternalURL = strings.TrimSpace(input.ExternalURL)
	input.LinkName = strings.TrimSpace(input.LinkName)
	if !validRequestKey(input.CreationKey) || input.Content == "" || len([]rune(input.Content)) > 10000 || len([]rune(input.LinkName)) > 160 || (input.ProjectID != nil && *input.ProjectID == 0) || !links.SafeExternal(input.ExternalURL) {
		return model.LightweightRecord{}, ErrInvalidRecordInput
	}
	record := model.LightweightRecord{Content: input.Content, ProjectID: input.ProjectID, ExternalURL: input.ExternalURL, LinkName: input.LinkName}
	if input.CreationKey != "" {
		record.CreationKey = &input.CreationKey
	}
	return record, nil
}
func (s *RecordService) List(ctx context.Context, userID uint, projectID *uint, archived bool) ([]model.LightweightRecord, error) {
	return s.records.List(ctx, userID, projectID, archived)
}
func (s *RecordService) Find(ctx context.Context, userID, id uint) (*model.LightweightRecord, error) {
	return s.records.Find(ctx, userID, id)
}
func (s *RecordService) Save(ctx context.Context, userID, id uint, input RecordInput) (*model.LightweightRecord, error) {
	record, err := normalizeRecord(input)
	if err != nil {
		return nil, err
	}
	return s.records.Save(ctx, userID, id, record)
}
func (s *RecordService) Archive(ctx context.Context, userID, id uint, archived bool) error {
	return s.records.Archive(ctx, userID, id, archived)
}
func (s *RecordService) Delete(ctx context.Context, userID, id uint) error {
	return s.records.Delete(ctx, userID, id)
}
func (s *InboxService) ConvertToRecord(ctx context.Context, userID, itemID uint, input RecordInput) (*model.InboxItem, *model.LightweightRecord, error) {
	item, err := s.items.Find(ctx, userID, itemID)
	if err != nil {
		return nil, nil, err
	}
	if input.Content == "" {
		input.Content = item.Content
	}
	record, err := normalizeRecord(input)
	if err != nil {
		return nil, nil, err
	}
	return s.items.Records().ConvertInbox(ctx, userID, itemID, record)
}

func validRequestKey(key string) bool {
	if len(key) > 80 {
		return false
	}
	for _, r := range key {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
