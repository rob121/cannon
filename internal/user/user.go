package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/rob121/cannon/internal/csrf"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/session"
	"github.com/rob121/cannon/internal/sites"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const sessionUserKey = "user_id"

var ErrNotAuthenticated = errors.New("not authenticated")

// Service provides user operations for a request context.
type Service struct {
	store *session.Store
	data  session.Data
	id    string
}

// ContextKey is used to store the user service in request context.
type ContextKey struct{}

// NewService loads session data for a request.
func NewService(store *session.Store, sessionID string) (*Service, error) {
	data, err := store.Load(sessionID)
	if err != nil {
		return nil, err
	}
	return &Service{store: store, data: data, id: sessionID}, nil
}

// CurrentID returns the logged-in user id, if any.
func (s *Service) CurrentID() (uint, bool) {
	v, ok := s.data[sessionUserKey]
	if !ok {
		return 0, false
	}
	switch id := v.(type) {
	case float64:
		return uint(id), true
	case int:
		return uint(id), true
	case uint:
		return id, true
	default:
		return 0, false
	}
}

// Current loads the current user model.
func (s *Service) Current(ctx context.Context) (*models.User, error) {
	userID, ok := s.CurrentID()
	if !ok {
		return nil, ErrNotAuthenticated
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var u models.User
	if err := db.Preload("Groups.Roles").First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Login sets the session user id.
func (s *Service) Login(userID uint) error {
	s.data[sessionUserKey] = userID
	return s.store.Save(s.id, s.data)
}

// SessionCSRFToken returns the CSRF token stored in the session, if any.
func (s *Service) SessionCSRFToken() (string, bool) {
	v, ok := s.data[csrf.SessionKey].(string)
	return v, ok && v != ""
}

// EnsureCSRFToken returns the session CSRF token, creating one when needed.
func (s *Service) EnsureCSRFToken() (string, error) {
	if token, ok := s.SessionCSRFToken(); ok {
		return token, nil
	}
	token, err := csrf.GenerateToken()
	if err != nil {
		return "", err
	}
	s.data[csrf.SessionKey] = token
	if err := s.store.Save(s.id, s.data); err != nil {
		return "", err
	}
	return token, nil
}

// ValidateCSRF checks the request CSRF token against the session value.
func (s *Service) ValidateCSRF(r *http.Request) error {
	expected, ok := s.SessionCSRFToken()
	if !ok {
		return csrf.ErrInvalid
	}
	if !csrf.Valid(expected, csrf.SubmittedToken(r)) {
		return csrf.ErrInvalid
	}
	return nil
}

// Logout clears the session.
func (s *Service) Logout() error {
	if s.id == "" {
		return nil
	}
	return s.store.Delete(s.id)
}

// Context returns serializable user context for extensions.
func (s *Service) Context(ctx context.Context) (map[string]any, error) {
	_, ok := s.CurrentID()
	if !ok {
		return map[string]any{"authenticated": false}, nil
	}
	u, err := s.Current(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"authenticated": true,
		"user_id":       u.UserID,
		"username":      u.Username,
		"email":         u.Email,
		"given_name":    u.GivenName,
		"family_name":   u.FamilyName,
	}, nil
}

// CreateLocalUser creates a bcrypt-authenticated user.
func CreateLocalUser(ctx context.Context, givenName, familyName, email, username, password string) (*models.User, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var auth models.Authenticator
	if err := db.Where("name = ?", "local").First(&auth).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			auth = models.Authenticator{Name: "local", Status: models.StatusActive}
			if err := db.Create(&auth).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	u := models.User{
		GivenName:  givenName,
		FamilyName: familyName,
		Email:      email,
		Username:   username,
		Hash:       string(hash),
		Status:     models.StatusActive,
		Validated:  true,
		AuthID:     &auth.AuthID,
	}
	if err := db.Create(&u).Error; err != nil {
		return nil, err
	}
	for _, name := range []string{"public", "registered"} {
		var g models.Group
		if err := db.Where("name = ?", name).First(&g).Error; err == nil {
			_ = db.Model(&u).Association("Groups").Append(&g)
		}
	}
	signupArgs := map[string]any{
		"user_id":  u.UserID,
		"username": u.Username,
		"email":    u.Email,
	}
	_, _ = hooks.Fire(ctx, hooks.OnUserSignup, signupArgs)
	return &u, nil
}

// AuthenticateLocal validates username/password.
func AuthenticateLocal(ctx context.Context, username, password string) (*models.User, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var u models.User
	if err := db.Where("username = ? AND status = ?", username, models.StatusActive).First(&u).Error; err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	return &u, nil
}

// Count returns total users for a site database.
func Count(ctx context.Context) (int64, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.Model(&models.User{}).Count(&count).Error
	return count, err
}

// FromContext returns the user service attached to context.
func FromContext(ctx context.Context) (*Service, error) {
	svc, ok := ctx.Value(ContextKey{}).(*Service)
	if !ok || svc == nil {
		return nil, fmt.Errorf("user service not in context")
	}
	return svc, nil
}

// WithContext attaches the user service to context.
func WithContext(ctx context.Context, svc *Service) context.Context {
	return context.WithValue(ctx, ContextKey{}, svc)
}
