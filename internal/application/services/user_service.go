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
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// UpdateUser applies a partial update to an existing user.
// Only non-nil fields are updated.
func (s *userService) UpdateUser(ctx context.Context, id uuid.UUID, name, email *string) (*entities.User, error) {
	// Step 1: Fetch existing
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Step 2: Nothing to update
	if name == nil && email == nil {
		return user, nil
	}

	// Step 3: Apply changes
	if name != nil {
		if err := user.SetName(*name); err != nil {
			return nil, err
		}
	}
	if email != nil {
		// If email changes, check uniqueness first (application concern)
		if *email != user.Email {
			existing, err := s.userRepo.GetByEmail(ctx, *email)
			if err != nil && !errors.Is(err, repositories.ErrNotFound) {
				return nil, err
			}
			if existing != nil {
				return nil, ErrEmailAlreadyInUse
			}
		}
		// Validate format and apply via entity (which uses the VO internally)
		if err := user.SetEmail(*email); err != nil {
			return nil, err
		}
	}

	// Step 4: Update timestamp and persist
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
