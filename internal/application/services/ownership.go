package services

import (
	"context"
	"errors"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

// ownership.go contains shared authorization checks used across services.
// Each function verifies a link in the ownership chain: User → Client → Project → Order.
// They are package-level functions (not struct methods) so any service can call them
// without duplicating logic.

// verifyClientOwnership fetches a client by ID and ensures it belongs to the given user.
// Returns the client or an application-level error.
func verifyClientOwnership(ctx context.Context, clientRepo repositories.ClientRepository, clientID uuid.UUID, userID uuid.UUID) (*entities.Client, error) {
	client, err := clientRepo.GetByID(ctx, clientID)
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

// verifyProjectOwnership verifies the full ownership chain: User → Client → Project.
// It reuses verifyClientOwnership for the first link.
// Returns the project or an application-level error.
func verifyProjectOwnership(ctx context.Context, clientRepo repositories.ClientRepository, projectRepo repositories.ProjectRepository, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID) (*entities.Project, error) {
	// Step 1: Verify client exists and belongs to user
	_, err := verifyClientOwnership(ctx, clientRepo, clientID, userID)
	if err != nil {
		return nil, err
	}

	// Step 2: Verify project exists and belongs to client
	project, err := projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	if project.ClientID != clientID {
		return nil, ErrProjectNotOwned
	}

	return project, nil
}

// verifyOrderOwnership verifies the full ownership chain: User → Client → Project → Order.
// It reuses verifyProjectOwnership for the first three links.
// Returns the order or an application-level error.
func verifyOrderOwnership(ctx context.Context, clientRepo repositories.ClientRepository, projectRepo repositories.ProjectRepository, orderRepo repositories.OrderRepository, orderID uuid.UUID, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID) (*entities.Order, error) {
	// Step 1: Verify User → Client → Project chain
	_, err := verifyProjectOwnership(ctx, clientRepo, projectRepo, projectID, clientID, userID)
	if err != nil {
		return nil, err
	}

	// Step 2: Verify order exists and belongs to project
	order, err := orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	if order.ProjectID != projectID {
		return nil, ErrOrderNotOwned
	}

	return order, nil
}

// verifyPieceDefAccess fetches a piece definition by ID and ensures the user can see it.
// Predefined definitions are visible to everyone. Custom definitions are only
// visible to their creator. Returns ErrPieceDefNotFound if the definition does
// not exist or the user cannot access it.
func verifyPieceDefAccess(ctx context.Context, pieceDefRepo repositories.PieceDefinitionRepository, defID uuid.UUID, userID uuid.UUID) (*entities.PieceDefinition, error) {
	def, err := pieceDefRepo.GetByID(ctx, defID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrPieceDefNotFound
		}
		return nil, err
	}

	if def.Predefined {
		return def, nil
	}

	if def.UserID == nil || *def.UserID != userID {
		return nil, ErrPieceDefNotFound
	}

	return def, nil
}

// verifyPieceDefOwnership fetches a piece definition by ID and ensures it belongs
// to the given user and can be modified. Returns ErrPieceDefPredefined for predefined
// definitions and ErrPieceDefNotOwned for custom definitions not owned by the user.
func verifyPieceDefOwnership(ctx context.Context, pieceDefRepo repositories.PieceDefinitionRepository, defID uuid.UUID, userID uuid.UUID) (*entities.PieceDefinition, error) {
	def, err := pieceDefRepo.GetByID(ctx, defID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrPieceDefNotFound
		}
		return nil, err
	}

	if def.Predefined {
		return nil, ErrPieceDefPredefined
	}

	if def.UserID == nil || *def.UserID != userID {
		return nil, ErrPieceDefNotOwned
	}

	return def, nil
}
