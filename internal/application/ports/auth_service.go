package ports

import (
	"context"

	"ductifact/internal/domain/entities"
)

// AuthService is the inbound port for authentication operations.
type AuthService interface {
	Register(ctx context.Context, name, email, password string) (*entities.User, string, error)
	Login(ctx context.Context, email, password string) (*entities.User, string, error)
}
