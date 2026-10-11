package service

import (
	"context"
	"errors"
	"learnos/internal/model"
	"learnos/internal/repository"
	"strings"
	"unicode/utf8"
)

var ErrAssetInput = errors.New("invalid asset snapshot")

type AssetService struct{ repo *repository.AssetRepository }

func NewAssetService(r *repository.AssetRepository) *AssetService { return &AssetService{r} }

// Yuan strings are parsed by the same exact decimal parser as the ledger.
type AssetInput struct {
	SnapshotDate string  `json:"snapshot_date"`
	Bank         *string `json:"bank"`
	Platform     *string `json:"platform"`
	Investment   *string `json:"investment"`
	Other        *string `json:"other"`
	Debt         *string `json:"debt"`
	Note         string  `json:"note"`
	Version      uint    `json:"version"`
}
type AssetView struct {
	model.AssetSnapshot
	Currency   string `json:"currency"`
	Complete   bool   `json:"complete"`
	AssetTotal *int64 `json:"asset_total_minor,string"`
	NetAsset   *int64 `json:"net_asset_minor,string"`
}

func assetView(row model.AssetSnapshot) AssetView {
	v := AssetView{AssetSnapshot: row, Currency: "CNY", Complete: true}
	var sum int64
	count := 0
	for _, amount := range []*int64{row.BankMinor, row.PlatformMinor, row.InvestmentMinor, row.OtherMinor} {
		if amount == nil {
			v.Complete = false
		} else {
			sum += *amount
			count++
		}
	}
	if count > 0 {
		v.AssetTotal = &sum
	}
	if row.DebtMinor == nil {
		v.Complete = false
	} else if count > 0 {
		net := sum - *row.DebtMinor
		v.NetAsset = &net
	}
	return v
}
func (s *AssetService) List(ctx context.Context, user uint) ([]AssetView, error) {
	rows, err := s.repo.List(ctx, user)
	out := []AssetView{}
	for _, row := range rows {
		out = append(out, assetView(row))
	}
	return out, err
}
func (s *AssetService) Get(ctx context.Context, user, id uint) (AssetView, error) {
	row, err := s.repo.Get(ctx, user, id)
	return assetView(row), err
}
func (s *AssetService) Save(ctx context.Context, user, id uint, in AssetInput) (AssetView, error) {
	if !financeDate(in.SnapshotDate) || !utf8.ValidString(in.Note) || utf8.RuneCountInString(in.Note) > 1000 || strings.ContainsRune(in.Note, 0) || (id > 0 && in.Version == 0) {
		return AssetView{}, ErrAssetInput
	}
	row := model.AssetSnapshot{ID: id, UserID: user, SnapshotDate: in.SnapshotDate, Note: strings.TrimSpace(in.Note), Version: in.Version}
	count := 0
	for _, pair := range []struct {
		input *string
		dest  **int64
	}{{in.Bank, &row.BankMinor}, {in.Platform, &row.PlatformMinor}, {in.Investment, &row.InvestmentMinor}, {in.Other, &row.OtherMinor}, {in.Debt, &row.DebtMinor}} {
		if pair.input == nil {
			continue
		}
		n, err := FinanceMoney(*pair.input)
		if err != nil || n < 0 {
			return AssetView{}, ErrAssetInput
		}
		*pair.dest = &n
		count++
	}
	if count == 0 {
		return AssetView{}, ErrAssetInput
	}
	saved, err := s.repo.Save(ctx, row)
	return assetView(saved), err
}
func (s *AssetService) Delete(ctx context.Context, user, id, version uint) error {
	if version == 0 {
		return ErrAssetInput
	}
	return s.repo.Delete(ctx, user, id, version)
}
