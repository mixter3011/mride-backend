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
	otpSvc  *services.OTPSvc
}

func NewAuthHandler(authSvc *services.AuthSvc, otpSvc *services.OTPSvc) *AuthHandler {
	return &AuthHandler{
		authSvc: authSvc,
		otpSvc:  otpSvc,
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

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Email != "" && !utils.ValidEmail(req.Email) {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid email format")
		return
	}

	err := h.authSvc.UpdateProfile(userID.(int), req)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Profile updated successfully", nil)
}

func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.UpdatePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	if !utils.ValidPwd(req.NewPassword) {
		utils.ErrJSON(c, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		utils.ErrJSON(c, http.StatusBadRequest, "Passwords don't match")
		return
	}

	err := h.authSvc.UpdatePassword(userID.(int), req.CurrentPassword, req.NewPassword)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Password updated successfully", nil)
}

func (h *AuthHandler) RequestPhoneUpdate(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.PhoneUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	if !utils.ValidPhone(req.Phone) {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid phone format")
		return
	}

	exists, err := h.authSvc.PhoneExists(req.Phone)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Database error")
		return
	}
	if exists {
		utils.ErrJSON(c, http.StatusBadRequest, "Phone number already registered")
		return
	}

	code := h.otpSvc.GenCode()
	if err := h.otpSvc.SaveOTP(req.Phone, code); err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to generate OTP")
		return
	}

	err = h.authSvc.StorePendingPhoneUpdate(userID.(int), req.Phone)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	err = h.otpSvc.SendOTP(req.Phone, code)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to send OTP")
		return
	}

	utils.SuccJSON(c, "OTP sent to new phone number", nil)
}

func (h *AuthHandler) ConfirmPhoneUpdate(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.ConfirmPhoneUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	pendingPhone, err := h.authSvc.GetPendingPhoneUpdate(userID.(int))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "No pending phone update found")
		return
	}

	err = h.otpSvc.VerifyOTP(pendingPhone, req.OTP)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.authSvc.UpdatePhoneVerified(userID.(int), pendingPhone)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.authSvc.ClearPendingPhoneUpdate(userID.(int))

	utils.SuccJSON(c, "Phone number updated successfully", nil)
}

func (h *AuthHandler) UpdateLocation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.UpdateLocationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Latitude < -90 || req.Latitude > 90 {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid latitude")
		return
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid longitude")
		return
	}

	err := h.authSvc.UpdateLocation(userID.(int), req.Latitude, req.Longitude)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccJSON(c, "Location updated successfully", nil)
}
