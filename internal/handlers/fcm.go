package handlers

import (
	"net/http"

	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type FCMHandler struct {
	notificationSvc *services.NotificationSvc
}

func NewFCMHandler(notificationSvc *services.NotificationSvc) *FCMHandler {
	return &FCMHandler{
		notificationSvc: notificationSvc,
	}
}

func (h *FCMHandler) SaveFCMToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req services.FCMTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	err := h.notificationSvc.SaveFCMToken(userID.(int), req)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to save FCM token")
		return
	}

	utils.SuccJSON(c, "FCM token saved successfully", nil)
}

func (h *FCMHandler) RemoveFCMToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	deviceID := c.Query("device_id")
	if deviceID == "" {
		utils.ErrJSON(c, http.StatusBadRequest, "Device ID is required")
		return
	}

	err := h.notificationSvc.RemoveFCMToken(userID.(int), deviceID)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to remove FCM token")
		return
	}

	utils.SuccJSON(c, "FCM token removed successfully", nil)
}
