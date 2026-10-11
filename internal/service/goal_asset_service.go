package service

import (
	"context"
	"errors"
	"learnos/internal/model"
	"strings"
	"unicode/utf8"
)

var ErrGoalAssetInput = errors.New("invalid goal asset reference")

type GoalAssetInput struct {
	Note         string `json:"note"`
	JudgmentDate string `json:"judgment_date"`
	Version      uint   `json:"version"`
}
type GoalAssetView struct {
	model.GoalAssetReference
	Snapshot AssetView `json:"snapshot"`
}

func (s *AssetService) GoalReferences(ctx context.Context, user, goal uint) ([]GoalAssetView, error) {
	rows, err := s.repo.GoalReferences(ctx, user, goal)
	if err != nil {
		return nil, err
	}
	out := []GoalAssetView{}
	for _, row := range rows {
		snapshot, err := s.Get(ctx, user, row.AssetSnapshotID)
		if err != nil {
			return nil, err
		}
		out = append(out, GoalAssetView{row, snapshot})
	}
	return out, nil
}
func (s *AssetService) ReferencingGoals(ctx context.Context, user, asset uint) ([]model.LifeGoal, error) {
	return s.repo.ReferencingGoals(ctx, user, asset)
}
func (s *AssetService) SaveReference(ctx context.Context, user, goal, asset uint, in GoalAssetInput, create bool) (model.GoalAssetReference, error) {
	if goal == 0 || asset == 0 || !financeDate(in.JudgmentDate) || !utf8.ValidString(in.Note) || utf8.RuneCountInString(in.Note) > 1000 || strings.ContainsRune(in.Note, 0) || (!create && in.Version == 0) {
		return model.GoalAssetReference{}, ErrGoalAssetInput
	}
	return s.repo.SaveReference(ctx, user, model.GoalAssetReference{GoalID: goal, AssetSnapshotID: asset, Note: strings.TrimSpace(in.Note), JudgmentDate: in.JudgmentDate, Version: in.Version}, create)
}
func (s *AssetService) DeleteReference(ctx context.Context, user, goal, asset, version uint) error {
	if version == 0 {
		return ErrGoalAssetInput
	}
	return s.repo.DeleteReference(ctx, user, goal, asset, version)
}
