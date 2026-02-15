package http

import (
	"net/http"
	"strconv"

	"backend-go-blueprint/internal/application/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BlueprintHandler struct {
	blueprintUseCase *usecases.BlueprintUseCase
}

func NewBlueprintHandler(blueprintUseCase *usecases.BlueprintUseCase) *BlueprintHandler {
	return &BlueprintHandler{
		blueprintUseCase: blueprintUseCase,
	}
}