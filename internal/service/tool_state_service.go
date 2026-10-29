package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"learnos/internal/model"
	"learnos/internal/repository"
)

var (
	ErrInvalidToolState = errors.New("invalid tool state")
	// ErrUnknownTool keeps the endpoint from becoming generic JSON storage.
	ErrUnknownTool = errors.New("unknown tool key")
)

// maxToolStateBytes bounds one document. The FEFW planner's full state is a few
// hundred characters per character entry and stays far below this.
const maxToolStateBytes = 512 * 1024

var knownTools = map[string]bool{
	model.ToolKeyFEFWPlanner: true,
}

type ToolStateService struct {
	states *repository.ToolStateRepository
}

func NewToolStateService(states *repository.ToolStateRepository) *ToolStateService {
	return &ToolStateService{states: states}
}

type ToolStateView struct {
	ToolKey   string          `json:"tool_key"`
	State     json.RawMessage `json:"state"`
	Revision  int64           `json:"revision"`
	UpdatedAt string          `json:"updated_at"`
}

// Get returns the saved document, or repository.ErrToolStateNotFound when the
// user has none yet (the client then offers to upload its local copy).
func (s *ToolStateService) Get(ctx context.Context, userID uint, toolKey string) (*ToolStateView, error) {
	toolKey, err := normalizeToolKey(toolKey)
	if err != nil {
		return nil, err
	}
	row, err := s.states.Get(ctx, userID, toolKey)
	if err != nil {
		return nil, err
	}
	return toToolStateView(row)
}

func (s *ToolStateService) Save(ctx context.Context, userID uint, toolKey string, raw json.RawMessage, expectedRevision int64) (*ToolStateView, error) {
	toolKey, err := normalizeToolKey(toolKey)
	if err != nil {
		return nil, err
	}
	if expectedRevision < 0 {
		return nil, ErrInvalidToolState
	}
	document, err := normalizeToolDocument(raw)
	if err != nil {
		return nil, err
	}
	row, err := s.states.Save(ctx, userID, toolKey, document, expectedRevision)
	if err != nil {
		return nil, err
	}
	return toToolStateView(row)
}

func normalizeToolKey(toolKey string) (string, error) {
	toolKey = strings.TrimSpace(toolKey)
	if !knownTools[toolKey] {
		return "", ErrUnknownTool
	}
	return toolKey, nil
}

// normalizeToolDocument accepts only a JSON object within the size budget and
// stores it compacted, so whitespace cannot inflate the row.
func normalizeToolDocument(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "", ErrInvalidToolState
	}
	if len(trimmed) > maxToolStateBytes {
		return "", ErrInvalidToolState
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(trimmed), &probe); err != nil {
		return "", ErrInvalidToolState
	}
	if probe == nil {
		return "", ErrInvalidToolState
	}
	compacted, err := json.Marshal(probe)
	if err != nil {
		return "", ErrInvalidToolState
	}
	return string(compacted), nil
}

func toToolStateView(row *model.ToolState) (*ToolStateView, error) {
	var document json.RawMessage
	if err := json.Unmarshal([]byte(row.StateJSON), &document); err != nil {
		return nil, ErrInvalidToolState
	}
	if document == nil {
		document = json.RawMessage("{}")
	}
	return &ToolStateView{
		ToolKey:   row.ToolKey,
		State:     document,
		Revision:  row.Revision,
		UpdatedAt: row.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}
