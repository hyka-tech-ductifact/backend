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
	ErrClientNotFound = errors.New("client not found")
	ErrClientNotOwned = errors.New("client does not belong to this user")
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
// 2. Build the domain entity (which validates name).
// 3. Persist via repository.
func (s *clientService) CreateClient(ctx context.Context, name string, userID uuid.UUID) (*entities.Client, error) {
	// Step 1: Verify the user exists
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Step 2: Domain entity validates its own invariants
	client, err := entities.NewClient(name, userID)
	if err != nil {
		return nil, err
	}

	// Step 3: Persist
	if err := s.clientRepo.Create(ctx, client); err != nil {
		return nil, err
	}

	return client, nil
}

// getOwnedClient fetches a client by ID and verifies it belongs to the given user.
func (s *clientService) getOwnedClient(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Client, error) {
	client, err := s.clientRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrClientNotFound
		}
		return nil, err
	}

	if client.UserID != userID {
		return nil, ErrClientNotOwned
	}

	return client, nil
}

// GetClientByID retrieves a client by ID, ensuring it belongs to the given user.
func (s *clientService) GetClientByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Client, error) {
	return s.getOwnedClient(ctx, id, userID)
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
// Only non-nil fields are updated. Ensures the client belongs to the given user.
func (s *clientService) UpdateClient(ctx context.Context, id uuid.UUID, userID uuid.UUID, name *string) (*entities.Client, error) {
	client, err := s.getOwnedClient(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Nothing to update
	if name == nil {
		return client, nil
	}

	// Apply changes
	if err := client.SetName(*name); err != nil {
		return nil, err
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
	client, err := s.getOwnedClient(ctx, id, userID)
	if err != nil {
		return err
	}

	return s.clientRepo.Delete(ctx, client.ID)
}
