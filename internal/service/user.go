package service

import (
	"errors"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// RegistrationEnabledKey is the system setting key controlling whether new
// users may self-register.
const RegistrationEnabledKey = "registration_enabled"

// UserService manages system users and registration settings.
type UserService struct {
	repo    *repository.UserRepository
	settings *repository.SystemSettingRepository
}

// NewUserService creates a new user service.
func NewUserService(repo *repository.UserRepository, settings *repository.SystemSettingRepository) *UserService {
	return &UserService{repo: repo, settings: settings}
}

// List returns all users (without password hashes).
func (s *UserService) List() ([]models.User, error) {
	users, _, err := s.repo.List(1000, 0)
	if err != nil {
		return nil, err
	}
	// Strip password hashes before returning.
	for i := range users {
		users[i].PasswordHash = ""
	}
	return users, nil
}

// Create adds a new user (admin action). role may be "admin" or "user".
func (s *UserService) Create(username, password, role string) error {
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if role != "admin" && role != "user" {
		role = "user"
	}
	existing, _ := s.repo.FindByUsername(username)
	if existing != nil {
		return errors.New("username already exists")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &models.User{
		ID:           generateID(),
		Username:     username,
		PasswordHash: string(hashed),
		Role:         role,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return s.repo.Create(user)
}

// UpdateRole changes a user's role (admin/user). Only admins may be demoted by
// another admin; the last admin cannot be demoted.
func (s *UserService) UpdateRole(id, role string) error {
	if role != "admin" && role != "user" {
		return errors.New("invalid role")
	}
	user, err := s.repo.FindByID(id)
	if err != nil {
		return errors.New("user not found")
	}
	if user.Role == "admin" && role != "admin" {
		// Prevent demoting the last admin.
		admins, err := s.adminCount()
		if err != nil {
			return err
		}
		if admins <= 1 {
			return errors.New("cannot demote the last admin")
		}
	}
	user.Role = role
	return s.repo.Update(user)
}

// SetStatus enables or disables a user account.
func (s *UserService) SetStatus(id, status string) error {
	if status != "active" && status != "disabled" {
		return errors.New("invalid status")
	}
	user, err := s.repo.FindByID(id)
	if err != nil {
		return errors.New("user not found")
	}
	if user.Role == "admin" && status != "active" {
		admins, err := s.adminCount()
		if err != nil {
			return err
		}
		if admins <= 1 {
			return errors.New("cannot disable the last admin")
		}
	}
	user.Status = status
	return s.repo.Update(user)
}

// Delete removes a user. The last admin cannot be deleted.
func (s *UserService) Delete(id string) error {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return errors.New("user not found")
	}
	if user.Role == "admin" {
		admins, err := s.adminCount()
		if err != nil {
			return err
		}
		if admins <= 1 {
			return errors.New("cannot delete the last admin")
		}
	}
	return s.repo.Delete(id)
}

// RegistrationEnabled reports whether self-registration is allowed.
func (s *UserService) RegistrationEnabled() (bool, error) {
	v, err := s.settings.Get(RegistrationEnabledKey)
	if err != nil {
		return false, err
	}
	// Default: registration enabled.
	return v != "false", nil
}

// SetRegistrationEnabled toggles self-registration.
func (s *UserService) SetRegistrationEnabled(enabled bool) error {
	v := "true"
	if !enabled {
		v = "false"
	}
	return s.settings.Set(RegistrationEnabledKey, v)
}

func (s *UserService) adminCount() (int, error) {
	users, _, err := s.repo.List(1000, 0)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, u := range users {
		if u.Role == "admin" {
			n++
		}
	}
	return n, nil
}
