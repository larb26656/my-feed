package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/larb26656/my-feed/backend/internal/modules/auth/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

// Handler exposes login, refresh, and logout routes.
type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/login", h.login)
	r.POST("/refresh", h.refresh)
	r.POST("/logout", h.logout)
}

// @Summary Login
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body loginRequest true "Credentials"
// @Success 200 {object} httpx.SuccessResponse[tokenResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	pair, err := h.svc.Login(c.Request.Context(), service.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toTokenResponse(pair)))
}

// @Summary Refresh Session
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body refreshRequest true "Refresh token"
// @Success 200 {object} httpx.SuccessResponse[tokenResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	pair, err := h.svc.Refresh(c.Request.Context(), service.RefreshInput{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toTokenResponse(pair)))
}

// @Summary Logout
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body logoutRequest true "Refresh token"
// @Success 200 {object} httpx.SuccessResponse[logoutResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *Handler) logout(c *gin.Context) {
	var req logoutRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.svc.Logout(c.Request.Context(), service.RefreshInput{
		RefreshToken: req.RefreshToken,
	}); err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(logoutResponse{OK: true}))
}
