package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"learnos/internal/auth"
	"learnos/internal/model"
	"learnos/internal/repository"
)

var (
	ErrInvalidTokenInput = errors.New("invalid api token input")
	// ErrInvalidAPIToken is the only error surfaced to clients so a guessed
	// token cannot distinguish "unknown" from "revoked" or "expired".
	ErrInvalidAPIToken = errors.New("invalid api token")
)

const (
	tokenRawPrefix   = "los_pat_"
	tokenRandomBytes = 24
	defaultTokenDays = 90
	maxTokenDays     = 365
)

type APITokenService struct {
	tokens *repository.APITokenRepository
	users  *repository.UserRepository
}

func NewAPITokenService(tokens *repository.APITokenRepository, users *repository.UserRepository) *APITokenService {
	return &APITokenService{tokens: tokens, users: users}
}

type APITokenView struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	Expired    bool       `json:"expired"`
}

type CreatedAPIToken struct {
	APITokenView
	// Token is returned exactly once, at creation time.
	Token string `json:"token"`
}

// Create mints a capture-only credential. expiresInDays is clamped to
// [1, maxTokenDays], defaulting to defaultTokenDays when unset.
func (s *APITokenService) Create(ctx context.Context, userID uint, name string, expiresInDays int) (*CreatedAPIToken, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "快速采集"
	}
	if len([]rune(name)) > 60 {
		return nil, ErrInvalidTokenInput
	}
	if expiresInDays == 0 {
		expiresInDays = defaultTokenDays
	}
	if expiresInDays < 1 || expiresInDays > maxTokenDays {
		return nil, ErrInvalidTokenInput
	}

	random := make([]byte, tokenRandomBytes)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}
	raw := tokenRawPrefix + hex.EncodeToString(random)
	sum := sha256.Sum256([]byte(raw))
	prefix := raw
	if len(prefix) > model.TokenPrefixLength {
		prefix = prefix[:model.TokenPrefixLength]
	}

	token := &model.APIToken{
		UserID:    userID,
		Name:      name,
		TokenHash: hex.EncodeToString(sum[:]),
		Prefix:    prefix,
		Scopes:    model.TokenScopeCaptureInbox,
		ExpiresAt: time.Now().UTC().Add(time.Duration(expiresInDays) * 24 * time.Hour),
	}
	if err := s.tokens.Create(ctx, token); err != nil {
		return nil, err
	}
	return &CreatedAPIToken{APITokenView: toAPITokenView(*token), Token: raw}, nil
}

func (s *APITokenService) List(ctx context.Context, userID uint) ([]APITokenView, error) {
	tokens, err := s.tokens.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	views := make([]APITokenView, 0, len(tokens))
	for _, token := range tokens {
		views = append(views, toAPITokenView(token))
	}
	return views, nil
}

func (s *APITokenService) Revoke(ctx context.Context, userID, id uint) error {
	return s.tokens.Revoke(ctx, userID, id)
}

// AuthenticateToken validates a bearer credential and returns a least-privilege
// principal: never admin, scoped to inbox capture only.
func (s *APITokenService) AuthenticateToken(ctx context.Context, raw string) (auth.Principal, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, tokenRawPrefix) {
		return auth.Principal{}, ErrInvalidAPIToken
	}
	sum := sha256.Sum256([]byte(raw))
	token, err := s.tokens.FindByHash(ctx, hex.EncodeToString(sum[:]))
	if err != nil {
		return auth.Principal{}, ErrInvalidAPIToken
	}
	if token.RevokedAt != nil {
		return auth.Principal{}, ErrInvalidAPIToken
	}
	if !token.ExpiresAt.IsZero() && !time.Now().UTC().Before(token.ExpiresAt) {
		return auth.Principal{}, ErrInvalidAPIToken
	}
	user, err := s.users.FindByID(ctx, token.UserID)
	if err != nil || user.Status != model.UserStatusActive {
		return auth.Principal{}, ErrInvalidAPIToken
	}
	s.tokens.TouchLastUsed(ctx, token.ID)
	return auth.Principal{
		UserID:   user.ID,
		Username: user.Username,
		Role:     string(model.UserRoleUser),
		Scopes:   token.ScopesList(),
	}, nil
}

func toAPITokenView(token model.APIToken) APITokenView {
	return APITokenView{
		ID:         token.ID,
		Name:       token.Name,
		Prefix:     token.Prefix,
		CreatedAt:  token.CreatedAt,
		LastUsedAt: token.LastUsedAt,
		ExpiresAt:  token.ExpiresAt,
		RevokedAt:  token.RevokedAt,
		Expired:    !token.ExpiresAt.IsZero() && !time.Now().UTC().Before(token.ExpiresAt),
	}
}
