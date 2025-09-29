package services

import (
	"errors"
	"fmt"
	"log"
	"mride-backend/internal/models"

	"gorm.io/gorm"
)

type ChatSvc struct {
	db              *gorm.DB
	webSocketSvc    WebSocketInterface
	notificationSvc NotificationSvcInterface
}

func NewChatSvc(db *gorm.DB, webSocketSvc WebSocketInterface, notificationSvc NotificationSvcInterface) *ChatSvc {
	if err := db.AutoMigrate(&models.RideChat{}); err != nil {
		log.Printf("Failed to auto-migrate chat model: %v", err)
	}

	return &ChatSvc{
		db:              db,
		webSocketSvc:    webSocketSvc,
		notificationSvc: notificationSvc,
	}
}

func (c *ChatSvc) SendChatMessage(userID, rideID uint, message string) (*models.ChatMessageResp, error) {
	if !c.isUserPartOfRide(userID, rideID) {
		return nil, fmt.Errorf("you are not part of this ride")
	}

	var ride models.Ride
	if err := c.db.First(&ride, rideID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("ride not found")
		}
		return nil, err
	}

	if ride.Status != "active" && ride.Status != "started" {
		return nil, fmt.Errorf("chat is only available for active rides")
	}

	chatMessage := models.RideChat{
		RideID:   rideID,
		SenderID: userID,
		Message:  message,
	}

	if err := c.db.Create(&chatMessage).Error; err != nil {
		return nil, err
	}

	var sender models.User
	if err := c.db.First(&sender, userID).Error; err != nil {
		return nil, err
	}

	resp := &models.ChatMessageResp{
		ID:        chatMessage.ID,
		RideID:    chatMessage.RideID,
		SenderID:  chatMessage.SenderID,
		Message:   chatMessage.Message,
		CreatedAt: chatMessage.CreatedAt,
		Sender: struct {
			ID       uint   `json:"id"`
			FullName string `json:"full_name"`
		}{
			ID:       sender.ID,
			FullName: sender.FullName,
		},
	}

	c.broadcastChatMessage(rideID, userID, *resp)

	return resp, nil
}

func (c *ChatSvc) GetChatHistory(userID, rideID uint, limit, offset int) (*models.GetChatHistoryResp, error) {
	if !c.isUserPartOfRide(userID, rideID) {
		return nil, fmt.Errorf("you are not part of this ride")
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var chatMessages []models.RideChat
	err := c.db.Preload("Sender").
		Where("ride_id = ?", rideID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&chatMessages).Error

	if err != nil {
		return nil, err
	}

	messages := make([]models.ChatMessageResp, len(chatMessages))
	for i, msg := range chatMessages {
		messages[i] = models.ChatMessageResp{
			ID:        msg.ID,
			RideID:    msg.RideID,
			SenderID:  msg.SenderID,
			Message:   msg.Message,
			CreatedAt: msg.CreatedAt,
			Sender: struct {
				ID       uint   `json:"id"`
				FullName string `json:"full_name"`
			}{
				ID:       msg.Sender.ID,
				FullName: msg.Sender.FullName,
			},
		}
	}

	for i := len(messages)/2 - 1; i >= 0; i-- {
		opp := len(messages) - 1 - i
		messages[i], messages[opp] = messages[opp], messages[i]
	}

	return &models.GetChatHistoryResp{
		Messages: messages,
		Count:    len(messages),
	}, nil
}

func (c *ChatSvc) isUserPartOfRide(userID, rideID uint) bool {
	var ride models.Ride
	if err := c.db.First(&ride, rideID).Error; err != nil {
		return false
	}

	if ride.UserID == userID {
		return true
	}

	var passenger models.RidePassenger
	err := c.db.Where("ride_id = ? AND passenger_id = ? AND status = ?",
		rideID, userID, "active").First(&passenger).Error

	return err == nil
}

func (c *ChatSvc) broadcastChatMessage(rideID, senderID uint, message models.ChatMessageResp) {
	participants := c.getRideParticipants(rideID)

	wsMessage := WSMessage{
		Type:    "chat_message",
		Title:   "New Chat Message",
		Message: fmt.Sprintf("New message in ride #%d", rideID),
		Data:    message,
	}

	for _, participantID := range participants {
		if participantID != senderID {
			if c.webSocketSvc != nil {
				if err := c.webSocketSvc.SendToUser(int(participantID), wsMessage); err != nil {
					log.Printf("Failed to send chat message via WebSocket to user %d: %v", participantID, err)
				}
			}

			if c.notificationSvc != nil {
				if err := c.notificationSvc.CreateChatNotification(participantID, rideID, senderID, message.Sender.FullName); err != nil {
					log.Printf("Failed to create chat notification for user %d: %v", participantID, err)
				}
			}
		}
	}
}

func (c *ChatSvc) getRideParticipants(rideID uint) []uint {
	var participants []uint

	var ride models.Ride
	if err := c.db.Select("user_id").First(&ride, rideID).Error; err == nil {
		participants = append(participants, ride.UserID)
	}

	var passengers []models.RidePassenger
	if err := c.db.Select("passenger_id").
		Where("ride_id = ? AND status = ?", rideID, "active").
		Find(&passengers).Error; err == nil {
		for _, passenger := range passengers {
			participants = append(participants, passenger.PassengerID)
		}
	}

	return participants
}

func (c *ChatSvc) DeleteChatHistory(rideID uint) error {
	return c.db.Where("ride_id = ?", rideID).Delete(&models.RideChat{}).Error
}

func (c *ChatSvc) GetUnreadChatCount(userID uint) (int, error) {
	return 0, nil
}
