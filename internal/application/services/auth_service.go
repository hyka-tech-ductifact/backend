package services

import (
	"context"
	"errors"

	"ductifact/internal/application/ports"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"
	"ductifact/internal/domain/valueobjects"
)

// --- Application-level errors ---

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
)

// authService implements usecases.AuthService.
type authService struct {
	userRepo      repositories.UserRepository
	tokenProvider ports.TokenProvider
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo repositories.UserRepository, tokenProvider ports.TokenProvider) *authService {
	return &authService{
		userRepo:      userRepo,
		tokenProvider: tokenProvider,
	}
}

// Register creates a new user with a hashed password and returns a JWT.
func (s *authService) Register(ctx context.Context, name, email, password string) (*entities.User, string, error) {
	// Step 1: Check if email is already taken
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, "", err
	}
	if existing != nil {
		return nil, "", ErrEmailTaken
	}

	// Step 2: Create user entity (validates name + email + password, hashes password)
	user, err := entities.NewUser(name, email, password)
	if err != nil {
		return nil, "", err
	}

	// Step 3: Persist
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", err
	}

	// Step 4: Generate JWT so the user is logged in immediately
	token, err := s.tokenProvider.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login verifies credentials and returns a JWT.
func (s *authService) Login(ctx context.Context, email, password string) (*entities.User, string, error) {
	// Step 1: Find user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists or not (security)
		return nil, "", ErrInvalidCredentials
	}

	// Step 2: Compare password with stored hash
	pwd := valueobjects.NewPasswordFromHash(user.PasswordHash)
	if err := pwd.Compare(password); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// Step 3: Generate JWT
	token, err := s.tokenProvider.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}
