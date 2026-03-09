package services

import (
	"context"
	"errors"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

// --- Application-level errors ---

var (
	ErrEmailAlreadyInUse = errors.New("email already in use")
	ErrUserNotFound      = errors.New("user not found")
)

// userService implements usecases.UserService.
// Unexported struct: can only be created via NewUserService.
type userService struct {
	userRepo repositories.UserRepository
}

// NewUserService creates a new UserService.
// It receives the outbound port (repository interface), not a concrete implementation.
func NewUserService(userRepo repositories.UserRepository) *userService {
	// 	"Accept interfaces, return structs"
	// — 	Proverbio de Go
	return &userService{userRepo: userRepo}
}

// GetUserByID retrieves a user by ID.
func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateUser applies a partial update to an existing user.
// Only non-nil fields are updated.
func (s *userService) UpdateUser(ctx context.Context, id uuid.UUID, name, email *string) (*entities.User, error) {
	// Step 1: Fetch existing
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Step 2: Apply changes
	if name != nil {
		user.Name = *name
	}
	if email != nil {
		// If email changes, check uniqueness
		if *email != user.Email {
			existing, _ := s.userRepo.GetByEmail(ctx, *email)
			if existing != nil {
				return nil, ErrEmailAlreadyInUse
			}
		}
		user.Email = *email
	}

	// Step 3: Update timestamp and persist
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
