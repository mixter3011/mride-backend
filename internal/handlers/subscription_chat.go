package handlers

import (
	"net/http"
	"strconv"

	"mride-backend/internal/models"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type SubscriptionChatService interface {
	SendChatMessage(userID, subscriptionID uint, message string) (*models.ChatMessageResp, error)
	GetChatHistory(userID, subscriptionID uint, limit, offset int) (*models.GetChatHistoryResp, error)
	MarkChatAsRead(userID, subscriptionID uint) error
	GetSubscriptionChats(userID uint) ([]models.SubscriptionChatRoomInfo, error)
	CanSendMessage(userID, subscriptionID uint) (bool, string)
}

type SubscriptionChatHandler struct {
	chatSvc SubscriptionChatService
}

func NewSubscriptionChatHandler(chatSvc SubscriptionChatService) *SubscriptionChatHandler {
	return &SubscriptionChatHandler{
		chatSvc: chatSvc,
	}
}

func (h *SubscriptionChatHandler) SendMessage(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	var req models.SendChatMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	resp, err := h.chatSvc.SendChatMessage(uint(userID.(int)), uint(subscriptionID), req.Message)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Message sent successfully", resp)
}

func (h *SubscriptionChatHandler) GetChatHistory(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	resp, err := h.chatSvc.GetChatHistory(uint(userID.(int)), uint(subscriptionID), limit, offset)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Chat history retrieved successfully", resp)
}

func (h *SubscriptionChatHandler) MarkChatAsRead(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err = h.chatSvc.MarkChatAsRead(uint(userID.(int)), uint(subscriptionID))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Chat marked as read", nil)
}

func (h *SubscriptionChatHandler) GetSubscriptionChats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	chatRooms, err := h.chatSvc.GetSubscriptionChats(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to get subscription chats")
		return
	}

	utils.SuccJSON(c, "Subscription chats retrieved successfully", map[string]interface{}{
		"chat_rooms": chatRooms,
		"count":      len(chatRooms),
	})
}

func (h *SubscriptionChatHandler) CheckCanSendMessage(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	canSend, reason := h.chatSvc.CanSendMessage(uint(userID.(int)), uint(subscriptionID))

	utils.SuccJSON(c, "Chat status retrieved", map[string]interface{}{
		"can_send": canSend,
		"reason":   reason,
	})
}
