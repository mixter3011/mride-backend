package handlers

import (
	"net/http"

	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type PushNotificationHandler struct {
	pushNotificationSvc services.PushNotificationInterface
}

func NewPushNotificationHandler(pushNotificationSvc services.PushNotificationInterface) *PushNotificationHandler {
	return &PushNotificationHandler{
		pushNotificationSvc: pushNotificationSvc,
	}
}

func (h *PushNotificationHandler) RegisterDeviceToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req struct {
		Token    string `json:"token" binding:"required"`
		Platform string `json:"platform" binding:"required,oneof=ios android web"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	err := h.pushNotificationSvc.RegisterDeviceToken(uint(userID.(int)), req.Token, req.Platform)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to register device token")
		return
	}

	utils.SuccJSON(c, "Device token registered successfully", nil)
}

func (h *PushNotificationHandler) UnregisterDeviceToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	err := h.pushNotificationSvc.UnregisterDeviceToken(uint(userID.(int)), req.Token)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to unregister device token")
		return
	}

	utils.SuccJSON(c, "Device token unregistered successfully", nil)
}

func (h *PushNotificationHandler) GetDeviceTokens(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	tokens, err := h.pushNotificationSvc.GetUserDeviceTokens(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to retrieve device tokens")
		return
	}

	utils.SuccJSON(c, "Device tokens retrieved successfully", map[string]interface{}{
		"tokens": tokens,
		"count":  len(tokens),
	})
}

func (h *PushNotificationHandler) UpdateNotificationPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var prefs services.NotificationPreference
	if err := c.ShouldBindJSON(&prefs); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	err := h.pushNotificationSvc.UpdateNotificationPreferences(uint(userID.(int)), &prefs)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to update notification preferences")
		return
	}

	utils.SuccJSON(c, "Notification preferences updated successfully", prefs)
}

func (h *PushNotificationHandler) GetNotificationPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	prefs, err := h.pushNotificationSvc.GetNotificationPreferences(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to retrieve notification preferences")
		return
	}

	utils.SuccJSON(c, "Notification preferences retrieved successfully", prefs)
}

func (h *PushNotificationHandler) GetPushNotificationStats(c *gin.Context) {
	stats, err := h.pushNotificationSvc.GetPushNotificationStats()
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to retrieve push notification stats")
		return
	}

	utils.SuccJSON(c, "Push notification stats retrieved successfully", stats)
}

func (h *PushNotificationHandler) TestPushNotification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	notification := &struct {
		ID      uint   `json:"id"`
		UserID  uint   `json:"user_id"`
		Type    string `json:"type"`
		Title   string `json:"title"`
		Message string `json:"message"`
		Data    []byte `json:"data"`
	}{
		ID:      0,
		UserID:  uint(userID.(int)),
		Type:    "test",
		Title:   req.Title,
		Message: req.Message,
		Data:    []byte(`{"test": true}`),
	}

	utils.SuccJSON(c, "Test notification will be sent", notification)
}
