package http

import (
	"backend-go-blueprint/internal/application/usecases"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(blueprintUseCase *usecases.BlueprintUseCase) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	return r
}
