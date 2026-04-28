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
	ErrEmailAlreadyInUse    = errors.New("email already in use")
	ErrUserNotFound         = errors.New("user not found")
	ErrHasAssociatedClients = errors.New("user has associated clients; set cascade=true to delete")
)

// userService implements usecases.UserService.
// Unexported struct: can only be created via NewUserService.
type userService struct {
	userRepo   repositories.UserRepository
	clientRepo repositories.ClientRepository
}

// NewUserService creates a new UserService.
// It receives the outbound ports (repository interfaces), not concrete implementations.
func NewUserService(userRepo repositories.UserRepository, clientRepo repositories.ClientRepository) *userService {
	return &userService{userRepo: userRepo, clientRepo: clientRepo}
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
func (s *userService) UpdateUser(
	ctx context.Context,
	id uuid.UUID,
	name, email, locale *string,
) (*entities.User, error) {
	// Step 1: Fetch existing
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Step 2: Nothing to update
	if name == nil && email == nil && locale == nil {
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
	if locale != nil {
		if err := user.SetLocale(*locale); err != nil {
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

// DeleteUser permanently removes a user and all associated data (GDPR).
// If cascade is false and the user has associated clients, it returns an error.
// The database cascades the deletion to clients, projects, orders, and pieces.
func (s *userService) DeleteUser(ctx context.Context, id uuid.UUID, cascade bool) error {
	// Step 1: Verify the user exists
	_, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// Step 2: Check for associated data
	count, err := s.clientRepo.CountByUserID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 && !cascade {
		return ErrHasAssociatedClients
	}

	// Step 3: Hard delete (cascades via FK)
	return s.userRepo.Delete(ctx, id)
}
