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
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
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

// Register creates a new user with a hashed password and returns a token pair.
func (s *authService) Register(ctx context.Context, name, email, password string) (*entities.User, *ports.TokenPair, error) {
	// Step 1: Check if email is already taken
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, ErrEmailAlreadyInUse
	}

	// Step 2: Create user entity (validates name + email + password, hashes password)
	user, err := entities.NewUser(name, email, password)
	if err != nil {
		return nil, nil, err
	}

	// Step 3: Persist
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	// Step 4: Generate token pair so the user is logged in immediately
	tokens, err := s.tokenProvider.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// Login verifies credentials and returns a token pair.
func (s *authService) Login(ctx context.Context, email, password string) (*entities.User, *ports.TokenPair, error) {
	// Step 1: Find user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists or not (security)
		return nil, nil, ErrInvalidCredentials
	}

	// Step 2: Compare password with stored hash
	pwd := valueobjects.NewPasswordFromHash(user.PasswordHash)
	if err := pwd.Compare(password); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Step 3: Generate token pair
	tokens, err := s.tokenProvider.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// RefreshToken validates a refresh token and returns a new token pair.
// This implements JWT rotation: each refresh invalidates the old pair
// and issues a completely new access + refresh token pair.
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*ports.TokenPair, error) {
	// Step 1: Validate the refresh token
	claims, err := s.tokenProvider.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Step 2: Verify the user still exists (could have been deleted)
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Step 3: Generate a new token pair (rotation)
	tokens, err := s.tokenProvider.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
