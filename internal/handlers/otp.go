package handlers

import (
	"net/http"

	"mride-backend/internal/models"
	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type OTPHandler struct {
	otpSvc  services.OTPService
	authSvc *services.AuthSvc
}

func NewOTPHandler(otpSvc services.OTPService, authSvc *services.AuthSvc) *OTPHandler {
	return &OTPHandler{
		otpSvc:  otpSvc,
		authSvc: authSvc,
	}
}

func (h *OTPHandler) SendOTP(c *gin.Context) {
	var req models.PhoneVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	if !utils.ValidPhone(req.Phone) {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid phone format")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	phoneExists, err := h.authSvc.PhoneExists(req.Phone)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Database error")
		return
	}
	if phoneExists {
		utils.ErrJSON(c, http.StatusBadRequest, "Phone number already registered")
		return
	}

	if err := h.authSvc.UpdatePhone(userID.(int), req.Phone); err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to update phone")
		return
	}

	code := h.otpSvc.GenCode()
	if err := h.otpSvc.SaveOTP(req.Phone, code); err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to generate OTP")
		return
	}

	if err := h.otpSvc.SendOTP(req.Phone, code); err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to send OTP")
		return
	}

	utils.SuccJSON(c, "OTP sent successfully", nil)
}

func (h *OTPHandler) VerifyOTP(c *gin.Context) {
	var req models.OTPVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	if err := h.otpSvc.VerifyOTP(req.Phone, req.Code); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authSvc.VerifyPhone(userID.(int)); err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to verify phone")
		return
	}

	utils.SuccJSON(c, "Phone verified successfully", nil)
}
