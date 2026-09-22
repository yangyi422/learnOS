package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"learnos/internal/auth"
	"learnos/internal/model"
	"learnos/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserBlocked        = errors.New("user is blocked")
	ErrUserExists         = errors.New("username already exists")
	ErrAdminRequired      = errors.New("administrator privileges required")
)

type UserService struct{ users *repository.UserRepository }

func NewUserService(users *repository.UserRepository) *UserService { return &UserService{users: users} }

type UserView struct {
	ID          uint             `json:"id"`
	Username    string           `json:"username"`
	DisplayName string           `json:"display_name"`
	Role        model.UserRole   `json:"role"`
	Status      model.UserStatus `json:"status"`
	LastLoginAt *time.Time       `json:"last_login_at"`
	CreatedAt   time.Time        `json:"created_at"`
}

type CreateUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func toUserView(user model.User) UserView {
	return UserView{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role, Status: user.Status, LastLoginAt: user.LastLoginAt, CreatedAt: user.CreatedAt}
}

func (s *UserService) EnsureBootstrap(ctx context.Context, username, passwordHash string) (*model.User, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(passwordHash) == "" {
		return nil, errors.New("bootstrap credentials are not configured")
	}
	user, err := s.users.Bootstrap(ctx, strings.TrimSpace(username), strings.TrimSpace(username), strings.TrimSpace(passwordHash))
	if err != nil {
		return nil, fmt.Errorf("bootstrap user: %w", err)
	}
	if err := s.users.AssignUnownedCourses(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("assign legacy courses: %w", err)
	}
	return user, nil
}

func (s *UserService) List(ctx context.Context) ([]UserView, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]UserView, 0, len(users))
	for _, user := range users {
		result = append(result, toUserView(user))
	}
	return result, nil
}

func (s *UserService) Create(ctx context.Context, request CreateUserRequest) (UserView, error) {
	username := strings.TrimSpace(request.Username)
	displayName := strings.TrimSpace(request.DisplayName)
	if username == "" || displayName == "" || len([]rune(request.Password)) < 8 {
		return UserView{}, errors.New("username, display name and a password of at least 8 characters are required")
	}
	if _, err := s.users.FindByUsername(ctx, username); err == nil {
		return UserView{}, ErrUserExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return UserView{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return UserView{}, fmt.Errorf("hash password: %w", err)
	}
	user := model.User{Username: username, DisplayName: displayName, PasswordHash: string(hash), Role: model.UserRoleUser, Status: model.UserStatusActive}
	if err := s.users.Create(ctx, &user); err != nil {
		return UserView{}, fmt.Errorf("create user: %w", err)
	}
	return toUserView(user), nil
}

func (s *UserService) SetStatus(ctx context.Context, id uint, status model.UserStatus) error {
	if status != model.UserStatusActive && status != model.UserStatusBlocked {
		return errors.New("invalid user status")
	}
	return s.users.Update(ctx, id, map[string]interface{}{"status": status})
}

func (s *UserService) Login(ctx context.Context, username, password string) (string, UserView, error) {
	user, err := s.users.FindByUsername(ctx, strings.TrimSpace(username))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", UserView{}, ErrInvalidCredentials
	}
	if err != nil {
		return "", UserView{}, err
	}
	if user.Status != model.UserStatusActive {
		return "", UserView{}, ErrUserBlocked
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", UserView{}, ErrInvalidCredentials
	}
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", UserView{}, err
	}
	rawToken := hex.EncodeToString(token[:])
	sum := sha256.Sum256([]byte(rawToken))
	if err := s.users.CreateSession(ctx, &model.Session{UserID: user.ID, TokenHash: hex.EncodeToString(sum[:]), ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour)}); err != nil {
		return "", UserView{}, err
	}
	now := time.Now().UTC()
	_ = s.users.Update(ctx, user.ID, map[string]interface{}{"last_login_at": now})
	user.LastLoginAt = &now
	return rawToken, toUserView(*user), nil
}

func (s *UserService) AuthenticateSession(ctx context.Context, rawToken string) (auth.Principal, error) {
	sum := sha256.Sum256([]byte(rawToken))
	session, err := s.users.FindSession(ctx, hex.EncodeToString(sum[:]))
	if err != nil {
		return auth.Principal{}, ErrInvalidCredentials
	}
	user, err := s.users.FindByID(ctx, session.UserID)
	if err != nil || user.Status != model.UserStatusActive {
		return auth.Principal{}, ErrInvalidCredentials
	}
	return auth.Principal{UserID: user.ID, Username: user.Username, Role: string(user.Role)}, nil
}

func (s *UserService) Logout(ctx context.Context, rawToken string) error {
	sum := sha256.Sum256([]byte(rawToken))
	return s.users.DeleteSession(ctx, hex.EncodeToString(sum[:]))
}

func (s *UserService) Get(ctx context.Context, id uint) (UserView, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	return toUserView(*user), nil
}
