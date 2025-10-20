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

	var passengerCount int64
	c.db.Model(&models.RidePassenger{}).
		Where("ride_id = ? AND status = ?", rideID, "active").
		Count(&passengerCount)

	if passengerCount == 0 && ride.UserID == userID {
		return nil, fmt.Errorf("cannot chat when there are no passengers in the ride")
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
		IsMine:    true,
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
			IsMine:    msg.SenderID == userID,
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
	err := c.db.Where("ride_id = ? AND passenger_id = ?", rideID, userID).First(&passenger).Error

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
		Where("ride_id = ?", rideID).
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

func (c *ChatSvc) HasChatMessages(rideID uint) (bool, error) {
	var count int64
	err := c.db.Model(&models.RideChat{}).Where("ride_id = ?", rideID).Count(&count).Error
	return count > 0, err
}

func (c *ChatSvc) GetActiveRidesWithChats(userID uint) ([]models.ChatRoomInfo, error) {
	var chatRooms []models.ChatRoomInfo

	var createdRides []models.Ride
	if err := c.db.Where("user_id = ? AND status IN ?", userID, []string{"active", "started"}).
		Find(&createdRides).Error; err != nil {
		return nil, err
	}

	for _, ride := range createdRides {
		var messageCount int64
		c.db.Model(&models.RideChat{}).Where("ride_id = ?", ride.ID).Count(&messageCount)

		if messageCount > 0 {
			var passenger models.RidePassenger
			if err := c.db.Preload("Passenger").
				Where("ride_id = ? AND status = ?", ride.ID, "active").
				First(&passenger).Error; err == nil {

				var lastMessage models.RideChat
				c.db.Where("ride_id = ?", ride.ID).Order("created_at DESC").First(&lastMessage)

				chatRooms = append(chatRooms, models.ChatRoomInfo{
					RideID:          ride.ID,
					OtherUserID:     passenger.PassengerID,
					OtherUserName:   passenger.Passenger.FullName,
					LastMessage:     lastMessage.Message,
					LastMessageTime: lastMessage.CreatedAt,
					UnreadCount:     0,
					RideStatus:      ride.Status,
					FromLocation:    ride.FromLocation,
					ToLocation:      ride.ToLocation,
					DepartureTime:   ride.DepartureTime,
					IsDriver:        true,
				})
			}
		}
	}

	var passengerRides []models.RidePassenger
	if err := c.db.Preload("Ride").Preload("Ride.User").
		Where("passenger_id = ? AND status = ?", userID, "active").
		Find(&passengerRides).Error; err != nil {
		return nil, err
	}

	for _, pr := range passengerRides {
		if pr.Ride.Status != "active" && pr.Ride.Status != "started" {
			continue
		}

		var messageCount int64
		c.db.Model(&models.RideChat{}).Where("ride_id = ?", pr.RideID).Count(&messageCount)

		if messageCount > 0 {
			var lastMessage models.RideChat
			c.db.Where("ride_id = ?", pr.RideID).Order("created_at DESC").First(&lastMessage)

			chatRooms = append(chatRooms, models.ChatRoomInfo{
				RideID:          pr.RideID,
				OtherUserID:     pr.Ride.UserID,
				OtherUserName:   pr.Ride.User.FullName,
				LastMessage:     lastMessage.Message,
				LastMessageTime: lastMessage.CreatedAt,
				UnreadCount:     0,
				RideStatus:      pr.Ride.Status,
				FromLocation:    pr.Ride.FromLocation,
				ToLocation:      pr.Ride.ToLocation,
				DepartureTime:   pr.Ride.DepartureTime,
				IsDriver:        false,
			})
		}
	}

	return chatRooms, nil
}

func (c *ChatSvc) GetExpiredRidesWithChats(userID uint) ([]models.ChatRoomInfo, error) {
	var chatRooms []models.ChatRoomInfo

	var createdRides []models.Ride
	if err := c.db.Where("user_id = ? AND status IN ?", userID, []string{"expired", "completed"}).
		Find(&createdRides).Error; err != nil {
		return nil, err
	}

	for _, ride := range createdRides {
		var messageCount int64
		c.db.Model(&models.RideChat{}).Where("ride_id = ?", ride.ID).Count(&messageCount)

		if messageCount > 0 {
			var passenger models.RidePassenger
			if err := c.db.Preload("Passenger").
				Where("ride_id = ?", ride.ID).
				First(&passenger).Error; err == nil {

				var lastMessage models.RideChat
				c.db.Where("ride_id = ?", ride.ID).Order("created_at DESC").First(&lastMessage)

				chatRooms = append(chatRooms, models.ChatRoomInfo{
					RideID:          ride.ID,
					OtherUserID:     passenger.PassengerID,
					OtherUserName:   passenger.Passenger.FullName,
					LastMessage:     lastMessage.Message,
					LastMessageTime: lastMessage.CreatedAt,
					UnreadCount:     0,
					RideStatus:      ride.Status,
					FromLocation:    ride.FromLocation,
					ToLocation:      ride.ToLocation,
					DepartureTime:   ride.DepartureTime,
					IsDriver:        true,
				})
			}
		}
	}

	var passengerRides []models.RidePassenger
	if err := c.db.Preload("Ride").Preload("Ride.User").
		Where("passenger_id = ?", userID).
		Find(&passengerRides).Error; err != nil {
		return nil, err
	}

	for _, pr := range passengerRides {
		if pr.Ride.Status != "expired" && pr.Ride.Status != "completed" {
			continue
		}

		var messageCount int64
		c.db.Model(&models.RideChat{}).Where("ride_id = ?", pr.RideID).Count(&messageCount)

		if messageCount > 0 {
			var lastMessage models.RideChat
			c.db.Where("ride_id = ?", pr.RideID).Order("created_at DESC").First(&lastMessage)

			chatRooms = append(chatRooms, models.ChatRoomInfo{
				RideID:          pr.RideID,
				OtherUserID:     pr.Ride.UserID,
				OtherUserName:   pr.Ride.User.FullName,
				LastMessage:     lastMessage.Message,
				LastMessageTime: lastMessage.CreatedAt,
				UnreadCount:     0,
				RideStatus:      pr.Ride.Status,
				FromLocation:    pr.Ride.FromLocation,
				ToLocation:      pr.Ride.ToLocation,
				DepartureTime:   pr.Ride.DepartureTime,
				IsDriver:        false,
			})
		}
	}

	return chatRooms, nil
}

func (c *ChatSvc) GetRidesWithChats(userID uint) ([]uint, error) {
	chatRooms, err := c.GetActiveRidesWithChats(userID)
	if err != nil {
		return nil, err
	}

	rideIDs := make([]uint, len(chatRooms))
	for i, room := range chatRooms {
		rideIDs[i] = room.RideID
	}
	return rideIDs, nil
}

func (c *ChatSvc) GetFirstMessageSender(rideID uint) (*models.User, error) {
	var firstChat models.RideChat
	err := c.db.Where("ride_id = ?", rideID).
		Order("created_at ASC").
		First(&firstChat).Error

	if err != nil {
		return nil, err
	}

	var sender models.User
	err = c.db.First(&sender, firstChat.SenderID).Error
	return &sender, err
}

func (c *ChatSvc) CanSendMessage(userID, rideID uint) (bool, string) {
	var ride models.Ride
	if err := c.db.First(&ride, rideID).Error; err != nil {
		return false, "Ride not found"
	}

	if ride.Status != "active" && ride.Status != "started" {
		return false, "Cannot send messages to expired or completed rides"
	}

	if !c.isUserPartOfRide(userID, rideID) {
		return false, "You are not part of this ride"
	}

	if ride.UserID == userID {
		var passengerCount int64
		c.db.Model(&models.RidePassenger{}).
			Where("ride_id = ? AND status = ?", rideID, "active").
			Count(&passengerCount)

		if passengerCount == 0 {
			return false, "Cannot chat when there are no passengers"
		}
	}

	return true, ""
}
