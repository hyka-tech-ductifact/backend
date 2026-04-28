package repositories
package repositories

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// OneTimeTokenRepository is the outbound port for one-time token persistence.
// Tokens are always scoped by type to prevent cross-purpose usage.
type OneTimeTokenRepository interface {
	Create(ctx context.Context, token *entities.OneTimeToken) error
	GetByToken(ctx context.Context, token string, tokenType entities.TokenType) (*entities.OneTimeToken, error)
	DeleteByUserIDAndType(ctx context.Context, userID uuid.UUID, tokenType entities.TokenType) error
}
