package usecases

import (
	"context"

	"backend-go-blueprint/internal/domain/repositories"
)

type BlueprintUseCase struct {
	userRepo repositories.UserRepository
}

func NewBlueprintUseCase(userRepo repositories.UserRepository) *BlueprintUseCase {
	return &BlueprintUseCase{
		userRepo: userRepo,
	}
}