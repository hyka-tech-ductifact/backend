package services

import (
	"context"
	"errors"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

// --- Application-level errors ---

var (
	ErrClientNotFound = repositories.ErrClientNotFound
	ErrClientNotOwned = repositories.ErrClientNotOwned
)

// clientService implements usecases.ClientService.
// Unexported struct: can only be created via NewClientService.
type clientService struct {
	clientRepo repositories.ClientRepository
	userRepo   repositories.UserRepository
}

// NewClientService creates a new ClientService.
// It receives both the client and user repositories (outbound ports).
// The user repository is needed to verify that the owning user exists.
func NewClientService(clientRepo repositories.ClientRepository, userRepo repositories.UserRepository) *clientService {
	return &clientService{
		clientRepo: clientRepo,
		userRepo:   userRepo,
	}
}

// CreateClient orchestrates client creation:
// 1. Verify the owning user exists.
// 2. Build the domain entity (which validates all fields).
// 3. Persist via repository.
func (s *clientService) CreateClient(ctx context.Context, params entities.CreateClientParams) (*entities.Client, error) {
	// Step 1: Verify the user exists
	_, err := s.userRepo.GetByID(ctx, params.UserID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Step 2: Domain entity validates its own invariants
	client, err := entities.NewClient(params)
	if err != nil {
		return nil, err
	}

	// Step 3: Persist
	if err := s.clientRepo.Create(ctx, client); err != nil {
		return nil, err
	}

	return client, nil
}

// GetClientByID retrieves a client by ID, ensuring it belongs to the given user.
func (s *clientService) GetClientByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Client, error) {
	return s.clientRepo.GetByIDForOwner(ctx, id, userID)
}

// ListClientsByUserID retrieves a paginated list of clients belonging to a user.
func (s *clientService) ListClientsByUserID(ctx context.Context, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.Client], error) {
	clients, totalItems, err := s.clientRepo.ListByUserID(ctx, userID, pg)
	if err != nil {
		return pagination.Result[*entities.Client]{}, err
	}

	return pagination.NewResult(clients, pg, totalItems), nil
}

// UpdateClient applies a partial update to an existing client.
// Only non-nil fields in params are updated. Ensures the client belongs to the given user.
func (s *clientService) UpdateClient(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdateClientParams) (*entities.Client, error) {
	client, err := s.clientRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Nothing to update
	if !params.HasChanges() {
		return client, nil
	}

	// Apply changes
	if params.Name != nil {
		if err := client.SetName(*params.Name); err != nil {
			return nil, err
		}
	}
	if params.Phone != nil {
		if err := client.SetPhone(*params.Phone); err != nil {
			return nil, err
		}
	}
	if params.Email != nil {
		if err := client.SetEmail(*params.Email); err != nil {
			return nil, err
		}
	}
	if params.Description != nil {
		if err := client.SetDescription(*params.Description); err != nil {
			return nil, err
		}
	}

	// Update timestamp and persist
	client.UpdatedAt = time.Now()
	if err := s.clientRepo.Update(ctx, client); err != nil {
		return nil, err
	}

	return client, nil
}

// DeleteClient removes a client, ensuring it belongs to the given user.
func (s *clientService) DeleteClient(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	client, err := s.clientRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	return s.clientRepo.Delete(ctx, client.ID)
}
