package handlers

import (
	"net/http"
	"strconv"

	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notificationSvc *services.NotificationSvc
}

func NewNotificationHandler(notificationSvc *services.NotificationSvc) *NotificationHandler {
	return &NotificationHandler{
		notificationSvc: notificationSvc,
	}
}

func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	resp, err := h.notificationSvc.GetUserNotifications(userID.(int), limit, offset)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch notifications")
		return
	}

	utils.SuccJSON(c, "Notifications retrieved successfully", resp)
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	notificationIDStr := c.Param("id")
	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	err = h.notificationSvc.MarkAsRead(userID.(int), notificationID)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to mark notification as read")
		return
	}

	utils.SuccJSON(c, "Notification marked as read", nil)
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err := h.notificationSvc.MarkAllAsRead(userID.(int))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to mark all notifications as read")
		return
	}

	utils.SuccJSON(c, "All notifications marked as read", nil)
}

func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	count, err := h.notificationSvc.GetUnreadCount(userID.(int))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to get unread count")
		return
	}

	utils.SuccJSON(c, "Unread count retrieved successfully", map[string]int{"unread_count": count})
}
