package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/okrammeitei/chatgo/internal/activitylog"
	"github.com/okrammeitei/chatgo/internal/user"
	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

type Service interface {
	Login(ctx context.Context, req *LoginRequest, ip, ua string) (*TokenPair, error)
	Logout(ctx context.Context, sessionID uuid.UUID, ip, ua string) error
	Refresh(ctx context.Context, refreshToken string, ip, ua string) (*TokenPair, error)
	ValidateAccessToken(ctx context.Context, tokenStr string) (*Claims, error)
	SetupMFA(ctx context.Context, userID uuid.UUID) (*MFASetupResponse, error)
	EnableMFA(ctx context.Context, userID uuid.UUID, code string) error
	DisableMFA(ctx context.Context, userID uuid.UUID, code string) error
	RevokeAllSessions(ctx context.Context, userID uuid.UUID) error
}

type service struct {
	repo        Repository
	userRepo    user.Repository
	activitySvc activitylog.Service
	jwtSecret   string
	accessTTL   time.Duration
	refreshTTL  time.Duration
	log         *zap.Logger
}

func NewService(
	repo Repository,
	userRepo user.Repository,
	activitySvc activitylog.Service,
	jwtSecret string,
	accessTTL, refreshTTL time.Duration,
	log *zap.Logger,
) Service {
	return &service{
		repo:        repo,
		userRepo:    userRepo,
		activitySvc: activitySvc,
		jwtSecret:   jwtSecret,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
		log:         log,
	}
}

func (s *service) Login(ctx context.Context, req *LoginRequest, ip, ua string) (*TokenPair, error) {
	u, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		_ = s.activitySvc.Log(ctx, &activitylog.Entry{
			Action: activitylog.ActionSecurityEvent, ResourceType: "auth",
			Details: map[string]string{"reason": "unknown_user", "username": req.Username},
			IPAddress: ip, UserAgent: ua, Severity: activitylog.SeverityWarning,
		})
		return nil, apperr.Unauthorized("invalid credentials")
	}

	if u.Status != user.StatusActive {
		return nil, apperr.Unauthorized("account is not active")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		uid := u.ID
		_ = s.activitySvc.Log(ctx, &activitylog.Entry{
			UserID: &uid, Action: activitylog.ActionSecurityEvent, ResourceType: "auth",
			Details: map[string]string{"reason": "bad_password"}, IPAddress: ip, UserAgent: ua,
			Severity: activitylog.SeverityWarning,
		})
		return nil, apperr.Unauthorized("invalid credentials")
	}

	if u.MFAEnabled {
		if req.MFACode == "" {
			return nil, apperr.BadRequest("MFA code required")
		}
		if !totp.Validate(req.MFACode, u.MFASecret) {
			return nil, apperr.Unauthorized("invalid MFA code")
		}
	}

	return s.issueTokenPair(ctx, u, ip, ua)
}

func (s *service) Logout(ctx context.Context, sessionID uuid.UUID, ip, ua string) error {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := s.repo.RevokeSession(ctx, sessionID); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &sess.UserID, Action: activitylog.ActionUserLogout, ResourceType: "session",
		ResourceID: sessionID.String(), IPAddress: ip, UserAgent: ua,
	})
	return nil
}

func (s *service) Refresh(ctx context.Context, refreshToken string, ip, ua string) (*TokenPair, error) {
	hash := hashToken(refreshToken)
	sess, err := s.repo.GetSessionByRefreshHash(ctx, hash)
	if err != nil {
		return nil, apperr.Unauthorized("invalid refresh token")
	}
	if sess.RevokedAt != nil || time.Now().After(sess.ExpiresAt) {
		return nil, apperr.Unauthorized("refresh token expired or revoked")
	}

	// Rotate: revoke old session
	if err := s.repo.RevokeSession(ctx, sess.ID); err != nil {
		return nil, err
	}

	u, err := s.userRepo.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, u, ip, ua)
}

func (s *service) ValidateAccessToken(_ context.Context, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, apperr.Unauthorized("invalid token")
	}
	c, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, apperr.Unauthorized("invalid token claims")
	}
	return &Claims{
		UserID:    c.UserID,
		Username:  c.Username,
		RoleID:    c.RoleID,
		SessionID: c.SessionID,
	}, nil
}

func (s *service) SetupMFA(ctx context.Context, userID uuid.UUID) (*MFASetupResponse, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "ChatGo",
		AccountName: u.Email,
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}

	u.MFASecret = key.Secret()
	u.UpdatedAt = time.Now().UTC()
	if err := s.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}

	return &MFASetupResponse{Secret: key.Secret(), QRCode: key.URL()}, nil
}

func (s *service) EnableMFA(ctx context.Context, userID uuid.UUID, code string) error {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !totp.Validate(code, u.MFASecret) {
		return apperr.BadRequest("invalid MFA code")
	}
	u.MFAEnabled = true
	u.UpdatedAt = time.Now().UTC()
	if err := s.userRepo.Update(ctx, u); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &userID, Action: activitylog.ActionMFAEnabled, ResourceType: "user",
		ResourceID: userID.String(),
	})
	return nil
}

func (s *service) DisableMFA(ctx context.Context, userID uuid.UUID, code string) error {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !totp.Validate(code, u.MFASecret) {
		return apperr.BadRequest("invalid MFA code")
	}
	u.MFAEnabled = false
	u.MFASecret = ""
	u.UpdatedAt = time.Now().UTC()
	if err := s.userRepo.Update(ctx, u); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &userID, Action: activitylog.ActionMFADisabled, ResourceType: "user",
		ResourceID: userID.String(),
	})
	return nil
}

func (s *service) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.RevokeAllUserSessions(ctx, userID); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &userID, Action: activitylog.ActionSessionRevoked, ResourceType: "session",
	})
	return nil
}

// --- helpers ---

type jwtClaims struct {
	jwt.RegisteredClaims
	UserID    uuid.UUID `json:"uid"`
	Username  string    `json:"usr"`
	RoleID    uuid.UUID `json:"rid"`
	SessionID uuid.UUID `json:"sid"`
}

func (s *service) issueTokenPair(ctx context.Context, u *user.User, ip, ua string) (*TokenPair, error) {
	sessionID := uuid.New()
	now := time.Now().UTC()

	// Access token
	claims := &jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:    u.ID,
		Username:  u.Username,
		RoleID:    u.RoleID,
		SessionID: sessionID,
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, apperr.Internal(err)
	}

	// Refresh token
	raw, err := generateToken()
	if err != nil {
		return nil, apperr.Internal(err)
	}

	sess := &Session{
		ID:               sessionID,
		UserID:           u.ID,
		RefreshTokenHash: hashToken(raw),
		IPAddress:        ip,
		UserAgent:        ua,
		ExpiresAt:        now.Add(s.refreshTTL),
		CreatedAt:        now,
	}
	if err := s.repo.CreateSession(ctx, sess); err != nil {
		return nil, err
	}

	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &u.ID, Action: activitylog.ActionUserLogin, ResourceType: "session",
		ResourceID: sessionID.String(), IPAddress: ip, UserAgent: ua,
	})

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: raw,
		ExpiresAt:    now.Add(s.accessTTL),
	}, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
