package handlers

import (
	"net/http"

	"mride-backend/internal/models"
	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *services.AuthSvc
}

func NewAuthHandler(authSvc *services.AuthSvc) *AuthHandler {
	return &AuthHandler{
		authSvc: authSvc,
	}
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var req models.SignUpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	if !utils.ValidEmail(req.Email) {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid email format")
		return
	}

	if !utils.ValidPwd(req.Password) {
		utils.ErrJSON(c, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	resp, err := h.authSvc.SignUp(req)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Account created successfully", resp)
}

func (h *AuthHandler) SignIn(c *gin.Context) {
	var req models.SignInReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	if !utils.ValidEmail(req.Email) {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid email format")
		return
	}

	if !utils.ValidPhone(req.Phone) {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid phone format")
		return
	}

	resp, err := h.authSvc.SignIn(req)
	if err != nil {
		utils.ErrJSON(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.SuccJSON(c, "Signed in successfully", resp)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	user, err := h.authSvc.GetUserByID(userID.(int))
	if err != nil {
		utils.ErrJSON(c, http.StatusNotFound, "User not found")
		return
	}

	utils.SuccJSON(c, "Profile retrieved successfully", user)
}
