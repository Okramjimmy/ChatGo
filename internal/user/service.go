package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/okrammeitei/chatgo/internal/activitylog"
	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

type Service interface {
	Create(ctx context.Context, req *CreateUserRequest) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ChangePassword(ctx context.Context, id uuid.UUID, req *ChangePasswordRequest) error
	List(ctx context.Context, f *ListFilter) ([]*User, int64, error)
	AssignRole(ctx context.Context, userID, roleID uuid.UUID, actorID uuid.UUID) error
	ListRoles(ctx context.Context) ([]*Role, error)
}

type service struct {
	repo        Repository
	activitySvc activitylog.Service
	defaultRole uuid.UUID
	log         *zap.Logger
}

func NewService(repo Repository, activitySvc activitylog.Service, defaultRole uuid.UUID, log *zap.Logger) Service {
	return &service{repo: repo, activitySvc: activitySvc, defaultRole: defaultRole, log: log}
}

func (s *service) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
	if existing, _ := s.repo.GetByUsername(ctx, req.Username); existing != nil {
		return nil, apperr.Conflict("username already taken")
	}
	if existing, _ := s.repo.GetByEmail(ctx, req.Email); existing != nil {
		return nil, apperr.Conflict("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	now := time.Now().UTC()
	u := &User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		Status:       StatusActive,
		RoleID:       s.defaultRole,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &u.ID, Action: activitylog.ActionUserCreate,
		ResourceType: "user", ResourceID: u.ID.String(),
	})
	return u, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.DisplayName != "" {
		u.DisplayName = req.DisplayName
	}
	if req.AvatarURL != "" {
		u.AvatarURL = req.AvatarURL
	}
	u.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &id, Action: activitylog.ActionUserUpdate,
		ResourceType: "user", ResourceID: id.String(),
	})
	return u, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &id, Action: activitylog.ActionUserDelete,
		ResourceType: "user", ResourceID: id.String(),
	})
	return nil
}

func (s *service) ChangePassword(ctx context.Context, id uuid.UUID, req *ChangePasswordRequest) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return apperr.Unauthorized("current password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperr.Internal(err)
	}
	u.PasswordHash = string(hash)
	u.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, u); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &id, Action: activitylog.ActionPasswordChange,
		ResourceType: "user", ResourceID: id.String(), Severity: activitylog.SeverityWarning,
	})
	return nil
}

func (s *service) List(ctx context.Context, f *ListFilter) ([]*User, int64, error) {
	return s.repo.List(ctx, f)
}

func (s *service) AssignRole(ctx context.Context, userID, roleID uuid.UUID, actorID uuid.UUID) error {
	if err := s.repo.AssignRole(ctx, userID, roleID); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &actorID, Action: activitylog.ActionRoleChanged,
		ResourceType: "user", ResourceID: userID.String(),
		Details: map[string]string{"role_id": roleID.String()},
		Severity: activitylog.SeverityWarning,
	})
	return nil
}

func (s *service) ListRoles(ctx context.Context) ([]*Role, error) {
	return s.repo.ListRoles(ctx)
}
