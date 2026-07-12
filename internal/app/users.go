package app

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"dback/internal/store"
	"dback/models"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidPhone       = errors.New("invalid phone number")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrUserDisabled       = errors.New("user is disabled")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

var iranMobileRE = regexp.MustCompile(`^09\d{9}$`)

func (a *App) ListUsers() ([]models.User, error) {
	users, err := a.store.ListUsers()
	if err != nil {
		return nil, err
	}
	out := make([]models.User, len(users))
	for i, u := range users {
		out[i] = redactUser(u)
	}
	return out, nil
}

func (a *App) GetUser(id string) (models.User, error) {
	u, err := a.store.GetUser(id)
	if err != nil {
		return models.User{}, err
	}
	return redactUser(u), nil
}

func (a *App) GetUserFull(id string) (models.User, error) {
	return a.store.GetUser(id)
}

func (a *App) CreateUser(phone, password, name string) (models.User, error) {
	phone, err := ValidatePhone(phone)
	if err != nil {
		return models.User{}, err
	}
	if err := validatePassword(password); err != nil {
		return models.User{}, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return models.User{}, err
	}
	now := time.Now().UTC()
	user := models.User{
		Phone:        phone,
		Name:         strings.TrimSpace(name),
		PasswordHash: hash,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := a.store.SaveUser(user); err != nil {
		if errors.Is(err, store.ErrUserExists) {
			return models.User{}, ErrUserExists
		}
		return models.User{}, err
	}
	users, err := a.store.ListUsers()
	if err != nil {
		return models.User{}, err
	}
	for _, u := range users {
		if u.Phone == phone {
			return redactUser(u), nil
		}
	}
	return models.User{}, ErrUserNotFound
}

func (a *App) UpdateUser(id string, phone, password, name string, enabled *bool) (models.User, error) {
	user, err := a.store.GetUser(id)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, err
	}
	if phone != "" {
		user.Phone, err = ValidatePhone(phone)
		if err != nil {
			return models.User{}, err
		}
	}
	if name != "" {
		user.Name = strings.TrimSpace(name)
	}
	if password != "" {
		if err := validatePassword(password); err != nil {
			return models.User{}, err
		}
		hash, err := hashPassword(password)
		if err != nil {
			return models.User{}, err
		}
		user.PasswordHash = hash
	}
	if enabled != nil {
		user.Enabled = *enabled
	}
	if err := a.store.SaveUser(user); err != nil {
		if errors.Is(err, store.ErrUserExists) {
			return models.User{}, ErrUserExists
		}
		return models.User{}, err
	}
	return a.GetUser(id)
}

func (a *App) DeleteUser(id string) error {
	if err := a.store.DeleteUser(id); err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (a *App) Authenticate(phone, password string) (models.User, error) {
	phone, err := ValidatePhone(phone)
	if err != nil {
		return models.User{}, ErrInvalidCredentials
	}
	user, err := a.store.GetUserByPhone(phone)
	if err != nil {
		return models.User{}, ErrInvalidCredentials
	}
	if !user.Enabled {
		return models.User{}, ErrUserDisabled
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return models.User{}, ErrInvalidCredentials
	}
	return user, nil
}

func ValidatePhone(phone string) (string, error) {
	var digits []rune
	for _, r := range phone {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}
	norm := string(digits)
	if strings.HasPrefix(norm, "98") && len(norm) == 12 {
		norm = "0" + norm[2:]
	}
	if strings.HasPrefix(norm, "9") && len(norm) == 10 {
		norm = "0" + norm
	}
	if !iranMobileRE.MatchString(norm) {
		return "", ErrInvalidPhone
	}
	return norm, nil
}

func validatePassword(password string) error {
	if len(password) < 6 {
		return ErrInvalidPassword
	}
	return nil
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func redactUser(u models.User) models.User {
	u.PasswordHash = ""
	return u
}

func IsUserNotFound(err error) bool {
	return errors.Is(err, ErrUserNotFound) || errors.Is(err, store.ErrUserNotFound)
}

func IsUserExists(err error) bool {
	return errors.Is(err, ErrUserExists) || errors.Is(err, store.ErrUserExists)
}

// EnsureDefaultAdmin creates the first admin user when the store has no users.
// Existing deployments are left unchanged.
func (a *App) EnsureDefaultAdmin(phone, password, name string) (models.User, bool, error) {
	users, err := a.store.ListUsers()
	if err != nil {
		return models.User{}, false, err
	}
	if len(users) > 0 {
		return models.User{}, false, nil
	}
	user, err := a.CreateUser(phone, password, name)
	if err != nil {
		return models.User{}, false, err
	}
	return user, true, nil
}

func userPublicMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidPhone):
		return "invalid phone number (expected 09XXXXXXXXX)"
	case errors.Is(err, ErrInvalidPassword):
		return "password must be at least 6 characters"
	case errors.Is(err, ErrUserExists):
		return "user with this phone already exists"
	case errors.Is(err, ErrUserNotFound):
		return "user not found"
	default:
		return fmt.Sprintf("%v", err)
	}
}
