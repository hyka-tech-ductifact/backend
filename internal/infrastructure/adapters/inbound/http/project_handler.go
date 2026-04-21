package http

import (
	"net/http"

	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- DTOs (HTTP-specific, not domain objects) ---

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Address     string `json:"address"`
	ManagerName string `json:"manager_name"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name" binding:"omitempty"`
	Address     *string `json:"address"`
	ManagerName *string `json:"manager_name"`
	Phone       *string `json:"phone"`
	Description *string `json:"description"`
}

type ProjectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	ManagerName string `json:"manager_name"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
	ClientID    string `json:"client_id"`
}

type ListProjectResponse struct {
	Data       []*ProjectResponse `json:"data"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalItems int64              `json:"total_items"`
	TotalPages int                `json:"total_pages"`
}

// --- Handler ---

type ProjectHandler struct {
	projectService usecases.ProjectService
}

func NewProjectHandler(projectService usecases.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

// CreateProject handles POST /clients/:client_id/projects
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	clientID, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID format"})
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project, err := h.projectService.CreateProject(c.Request.Context(), userID, entities.CreateProjectParams{
		Name:        req.Name,
		Address:     req.Address,
		ManagerName: req.ManagerName,
		Phone:       req.Phone,
		Description: req.Description,
		ClientID:    clientID,
	})
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toProjectResponse(project))
}

// ListProjects handles GET /clients/:client_id/projects?page=1&page_size=20
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	clientID, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID format"})
		return
	}

	pg, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.projectService.ListProjectsByClientID(c.Request.Context(), clientID, userID, pg)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response := make([]*ProjectResponse, len(result.Data))
	for i, project := range result.Data {
		response[i] = toProjectResponse(project)
	}

	c.JSON(http.StatusOK, &ListProjectResponse{
		Data:       response,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	})
}

// GetProject handles GET /projects/:project_id
func (h *ProjectHandler) GetProject(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID format"})
		return
	}

	project, err := h.projectService.GetProjectByID(c.Request.Context(), projectID, userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toProjectResponse(project))
}

// UpdateProject handles PUT /projects/:project_id
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID format"})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project, err := h.projectService.UpdateProject(c.Request.Context(), projectID, userID, entities.UpdateProjectParams{
		Name:        req.Name,
		Address:     req.Address,
		ManagerName: req.ManagerName,
		Phone:       req.Phone,
		Description: req.Description,
	})
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toProjectResponse(project))
}

// DeleteProject handles DELETE /projects/:project_id
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID format"})
		return
	}

	if err := h.projectService.DeleteProject(c.Request.Context(), projectID, userID); err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// --- Mapper: Domain → HTTP Response ---

func toProjectResponse(project *entities.Project) *ProjectResponse {
	return &ProjectResponse{
		ID:          project.ID.String(),
		Name:        project.Name,
		Address:     project.Address,
		ManagerName: project.ManagerName,
		Phone:       project.Phone,
		Description: project.Description,
		ClientID:    project.ClientID.String(),
	}
}
