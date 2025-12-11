package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
)

// TenantHandler handles tenant management HTTP requests
type TenantHandler struct {
	tenantService service.TenantService
}

// NewTenantHandler creates a new TenantHandler
func NewTenantHandler(tenantService service.TenantService) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

// Create handles tenant creation
// POST /api/v1/tenants
func (h *TenantHandler) Create(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	// Validate slug format
	if valid, msg := req.ValidateSlug(); !valid {
		c.JSON(http.StatusBadRequest, response.Error("INVALID_SLUG", msg))
		return
	}

	result, err := h.tenantService.Create(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrTenantAlreadyExists) {
			c.JSON(http.StatusConflict, response.Error("TENANT_EXISTS", "Tenant with this slug already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, response.Success(result))
}

// GetByID handles retrieving a tenant by ID
// GET /api/v1/tenants/:id
func (h *TenantHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("Tenant ID is required"))
		return
	}

	result, err := h.tenantService.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Tenant not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(result))
}

// GetBySlug handles retrieving a tenant by slug
// GET /api/v1/tenants/slug/:slug
func (h *TenantHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("Slug is required"))
		return
	}

	result, err := h.tenantService.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Tenant not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(result))
}

// List handles retrieving all tenants with pagination
// GET /api/v1/tenants
func (h *TenantHandler) List(c *gin.Context) {
	var query dto.ListTenantsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	result, err := h.tenantService.List(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(result))
}

// Update handles tenant update
// PUT /api/v1/tenants/:id
func (h *TenantHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("Tenant ID is required"))
		return
	}

	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	// Validate that at least one field is provided
	if valid, msg := req.Validate(); !valid {
		c.JSON(http.StatusBadRequest, response.Error("INVALID_UPDATE", msg))
		return
	}

	result, err := h.tenantService.Update(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Tenant not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(result))
}

// Delete handles tenant soft deletion
// DELETE /api/v1/tenants/:id
func (h *TenantHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("Tenant ID is required"))
		return
	}

	err := h.tenantService.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Tenant not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(gin.H{"message": "Tenant deleted successfully"}))
}
