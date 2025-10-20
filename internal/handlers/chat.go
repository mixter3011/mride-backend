package handlers

import (
	"net/http"
	"strconv"

	"mride-backend/internal/models"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type ChatService interface {
	SendChatMessage(userID, rideID uint, message string) (*models.ChatMessageResp, error)
	GetChatHistory(userID, rideID uint, limit, offset int) (*models.GetChatHistoryResp, error)
	GetUnreadChatCount(userID uint) (int, error)
	GetRidesWithChats(userID uint) ([]uint, error)
	GetActiveRidesWithChats(userID uint) ([]models.ChatRoomInfo, error)
	GetExpiredRidesWithChats(userID uint) ([]models.ChatRoomInfo, error)
	CanSendMessage(userID, rideID uint) (bool, string)
}

type ChatHandler struct {
	chatSvc ChatService
}

func NewChatHandler(chatSvc ChatService) *ChatHandler {
	return &ChatHandler{
		chatSvc: chatSvc,
	}
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, err := strconv.Atoi(rideIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid ride ID")
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

	resp, err := h.chatSvc.SendChatMessage(uint(userID.(int)), uint(rideID), req.Message)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Message sent successfully", resp)
}

func (h *ChatHandler) GetChatHistory(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, err := strconv.Atoi(rideIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid ride ID")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	resp, err := h.chatSvc.GetChatHistory(uint(userID.(int)), uint(rideID), limit, offset)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Chat history retrieved successfully", resp)
}

func (h *ChatHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	count, err := h.chatSvc.GetUnreadChatCount(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to get unread count")
		return
	}

	utils.SuccJSON(c, "Unread count retrieved successfully", map[string]interface{}{
		"unread_count": count,
	})
}

func (h *ChatHandler) GetActiveChats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	chatRooms, err := h.chatSvc.GetActiveRidesWithChats(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to get active chats")
		return
	}

	utils.SuccJSON(c, "Active chats retrieved successfully", map[string]interface{}{
		"chat_rooms": chatRooms,
		"count":      len(chatRooms),
	})
}

func (h *ChatHandler) GetExpiredChats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	chatRooms, err := h.chatSvc.GetExpiredRidesWithChats(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to get expired chats")
		return
	}

	utils.SuccJSON(c, "Expired chats retrieved successfully", map[string]interface{}{
		"chat_rooms": chatRooms,
		"count":      len(chatRooms),
	})
}

func (h *ChatHandler) GetRidesWithChats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	rideIDs, err := h.chatSvc.GetRidesWithChats(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to get rides with chats")
		return
	}

	utils.SuccJSON(c, "Rides with chats retrieved successfully", map[string]interface{}{
		"ride_ids": rideIDs,
	})
}

func (h *ChatHandler) CheckCanSendMessage(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, err := strconv.Atoi(rideIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid ride ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	canSend, reason := h.chatSvc.CanSendMessage(uint(userID.(int)), uint(rideID))

	utils.SuccJSON(c, "Chat status retrieved", map[string]interface{}{
		"can_send": canSend,
		"reason":   reason,
	})
}
