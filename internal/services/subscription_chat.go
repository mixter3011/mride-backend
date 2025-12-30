package services

import (
	"errors"
	"fmt"
	"log"
	"mride-backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type SubscriptionChatSvc struct {
	db              *gorm.DB
	webSocketSvc    WebSocketInterface
	notificationSvc NotificationSvcInterface
}

func NewSubscriptionChatSvc(db *gorm.DB, webSocketSvc WebSocketInterface, notificationSvc NotificationSvcInterface) *SubscriptionChatSvc {
	if err := db.AutoMigrate(&models.SubscriptionChat{}, &models.SubscriptionChatReadStatus{}); err != nil {
		log.Printf("Failed to auto-migrate subscription chat models: %v", err)
	}

	return &SubscriptionChatSvc{
		db:              db,
		webSocketSvc:    webSocketSvc,
		notificationSvc: notificationSvc,
	}
}

func (s *SubscriptionChatSvc) SendChatMessage(userID, subscriptionID uint, message string) (*models.ChatMessageResp, error) {
	if !s.isUserPartOfSubscription(userID, subscriptionID) {
		return nil, fmt.Errorf("you are not part of this subscription")
	}

	var subscription models.RideSubscription
	if err := s.db.First(&subscription, subscriptionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, err
	}

	if subscription.Status != "active" {
		return nil, fmt.Errorf("chat is only available for active subscriptions")
	}

	chatMessage := models.SubscriptionChat{
		SubscriptionID: subscriptionID,
		SenderID:       userID,
		Message:        message,
	}

	if err := s.db.Create(&chatMessage).Error; err != nil {
		return nil, err
	}

	s.updateReadStatus(userID, subscriptionID, chatMessage.ID)

	var sender models.User
	if err := s.db.First(&sender, userID).Error; err != nil {
		return nil, err
	}

	resp := &models.ChatMessageResp{
		ID:        chatMessage.ID,
		RideID:    chatMessage.SubscriptionID,
		SenderID:  chatMessage.SenderID,
		Message:   chatMessage.Message,
		CreatedAt: chatMessage.CreatedAt,
		IsMine:    true,
		IsRead:    false,
		Sender: struct {
			ID       uint   `json:"id"`
			FullName string `json:"full_name"`
		}{
			ID:       sender.ID,
			FullName: sender.FullName,
		},
	}

	s.broadcastChatMessage(subscriptionID, userID, *resp)

	return resp, nil
}

func (s *SubscriptionChatSvc) GetChatHistory(userID, subscriptionID uint, limit, offset int) (*models.GetChatHistoryResp, error) {
	if !s.isUserPartOfSubscription(userID, subscriptionID) {
		return nil, fmt.Errorf("you are not part of this subscription")
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var chatMessages []models.SubscriptionChat
	err := s.db.Preload("Sender").
		Where("subscription_id = ?", subscriptionID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&chatMessages).Error

	if err != nil {
		return nil, err
	}

	otherUserID := s.getOtherUserID(userID, subscriptionID)
	var readStatus models.SubscriptionChatReadStatus
	s.db.Where("subscription_id = ? AND user_id = ?", subscriptionID, otherUserID).First(&readStatus)

	messages := make([]models.ChatMessageResp, len(chatMessages))
	for i, msg := range chatMessages {
		isRead := msg.SenderID == userID || msg.ID <= readStatus.LastReadMessageID

		messages[i] = models.ChatMessageResp{
			ID:        msg.ID,
			RideID:    msg.SubscriptionID,
			SenderID:  msg.SenderID,
			Message:   msg.Message,
			CreatedAt: msg.CreatedAt,
			IsMine:    msg.SenderID == userID,
			IsRead:    isRead,
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

func (s *SubscriptionChatSvc) MarkChatAsRead(userID, subscriptionID uint) error {
	if !s.isUserPartOfSubscription(userID, subscriptionID) {
		return fmt.Errorf("you are not part of this subscription")
	}

	var lastMessage models.SubscriptionChat
	err := s.db.Where("subscription_id = ?", subscriptionID).Order("created_at DESC").First(&lastMessage).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	s.updateReadStatus(userID, subscriptionID, lastMessage.ID)

	return nil
}

func (s *SubscriptionChatSvc) GetSubscriptionChats(userID uint) ([]models.SubscriptionChatRoomInfo, error) {
	var chatRooms []models.SubscriptionChatRoomInfo

	var createdSubs []models.RideSubscription
	if err := s.db.Where("user_id = ?", userID).
		Find(&createdSubs).Error; err != nil {
		return nil, err
	}

	for _, sub := range createdSubs {
		chatRoom := s.buildChatRoomInfo(userID, sub, true)
		if chatRoom != nil {
			chatRooms = append(chatRooms, *chatRoom)
		}
	}

	var subscribedSubs []models.SubscriptionSubscriber
	if err := s.db.Preload("Subscription").Preload("Subscription.User").
		Where("subscriber_id = ?", userID).
		Find(&subscribedSubs).Error; err != nil {
		return nil, err
	}

	for _, ss := range subscribedSubs {
		chatRoom := s.buildChatRoomInfo(userID, ss.Subscription, false)
		if chatRoom != nil {
			chatRooms = append(chatRooms, *chatRoom)
		}
	}

	return chatRooms, nil
}

func (s *SubscriptionChatSvc) buildChatRoomInfo(userID uint, sub models.RideSubscription, isDriver bool) *models.SubscriptionChatRoomInfo {
	var messageCount int64
	s.db.Model(&models.SubscriptionChat{}).Where("subscription_id = ?", sub.ID).Count(&messageCount)

	var otherUserID uint
	var otherUserName string

	if isDriver {
		var subscriber models.SubscriptionSubscriber
		err := s.db.Preload("Subscriber").
			Where("subscription_id = ? AND status = ?", sub.ID, "active").
			First(&subscriber).Error

		if err != nil {
			return nil
		}

		otherUserID = subscriber.SubscriberID
		otherUserName = subscriber.Subscriber.FullName
	} else {
		if sub.User.ID == 0 {
			s.db.Preload("User").First(&sub, sub.ID)
		}
		otherUserID = sub.UserID
		otherUserName = sub.User.FullName
	}

	var lastMessage string
	var lastMessageTime time.Time
	var lastMessageRead bool
	var unreadCount int64

	if messageCount == 0 {
		lastMessage = "No messages yet"
		lastMessageTime = sub.CreatedAt
		lastMessageRead = true
		unreadCount = 0
	} else {
		var lastMsg models.SubscriptionChat
		s.db.Where("subscription_id = ?", sub.ID).Order("created_at DESC").First(&lastMsg)

		lastMessage = lastMsg.Message
		lastMessageTime = lastMsg.CreatedAt

		var readStatus models.SubscriptionChatReadStatus
		s.db.Where("subscription_id = ? AND user_id = ?", sub.ID, userID).First(&readStatus)

		lastMessageRead = lastMsg.SenderID == userID || lastMsg.ID <= readStatus.LastReadMessageID

		s.db.Model(&models.SubscriptionChat{}).
			Where("subscription_id = ? AND sender_id = ? AND id > ?", sub.ID, otherUserID, readStatus.LastReadMessageID).
			Count(&unreadCount)
	}

	return &models.SubscriptionChatRoomInfo{
		SubscriptionID:  sub.ID,
		OtherUserID:     otherUserID,
		OtherUserName:   otherUserName,
		LastMessage:     lastMessage,
		LastMessageTime: lastMessageTime,
		LastMessageRead: lastMessageRead,
		UnreadCount:     int(unreadCount),
		Title:           sub.Title,
		FromLocation:    sub.FromLocation,
		ToLocation:      sub.ToLocation,
		DepartureTime:   sub.DepartureTime,
		IsDriver:        isDriver,
		Status:          sub.Status,
	}
}

func (s *SubscriptionChatSvc) updateReadStatus(userID, subscriptionID, messageID uint) {
	var readStatus models.SubscriptionChatReadStatus
	err := s.db.Where("subscription_id = ? AND user_id = ?", subscriptionID, userID).First(&readStatus).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		readStatus = models.SubscriptionChatReadStatus{
			SubscriptionID:    subscriptionID,
			UserID:            userID,
			LastReadMessageID: messageID,
		}
		s.db.Create(&readStatus)
	} else if err == nil && messageID > readStatus.LastReadMessageID {
		s.db.Model(&readStatus).Update("last_read_message_id", messageID)
	}
}

func (s *SubscriptionChatSvc) getOtherUserID(userID, subscriptionID uint) uint {
	var sub models.RideSubscription
	s.db.First(&sub, subscriptionID)

	if sub.UserID == userID {
		var subscriber models.SubscriptionSubscriber
		s.db.Where("subscription_id = ? AND status = ?", subscriptionID, "active").First(&subscriber)
		return subscriber.SubscriberID
	}

	return sub.UserID
}

func (s *SubscriptionChatSvc) isUserPartOfSubscription(userID, subscriptionID uint) bool {
	var sub models.RideSubscription
	if err := s.db.First(&sub, subscriptionID).Error; err != nil {
		return false
	}

	if sub.UserID == userID {
		return true
	}

	var subscriber models.SubscriptionSubscriber
	err := s.db.Where("subscription_id = ? AND subscriber_id = ? AND status = ?", subscriptionID, userID, "active").First(&subscriber).Error

	return err == nil
}

func (s *SubscriptionChatSvc) broadcastChatMessage(subscriptionID, senderID uint, message models.ChatMessageResp) {
	participants := s.getSubscriptionParticipants(subscriptionID)

	wsMessage := WSMessage{
		Type:    "subscription_chat_message",
		Title:   "New Subscription Chat",
		Message: fmt.Sprintf("New message in subscription #%d", subscriptionID),
		Data:    message,
	}

	for _, participantID := range participants {
		if participantID != senderID {
			if s.webSocketSvc != nil {
				if err := s.webSocketSvc.SendToUser(int(participantID), wsMessage); err != nil {
					log.Printf("Failed to send subscription chat via WebSocket to user %d: %v", participantID, err)
				}
			}

			if s.notificationSvc != nil {
				if err := s.notificationSvc.CreateChatNotification(participantID, subscriptionID, senderID, message.Sender.FullName); err != nil {
					log.Printf("Failed to create subscription chat notification for user %d: %v", participantID, err)
				}
			}
		}
	}
}

func (s *SubscriptionChatSvc) getSubscriptionParticipants(subscriptionID uint) []uint {
	var participants []uint

	var sub models.RideSubscription
	if err := s.db.Select("user_id").First(&sub, subscriptionID).Error; err == nil {
		participants = append(participants, sub.UserID)
	}

	var subscribers []models.SubscriptionSubscriber
	if err := s.db.Select("subscriber_id").
		Where("subscription_id = ? AND status = ?", subscriptionID, "active").
		Find(&subscribers).Error; err == nil {
		for _, subscriber := range subscribers {
			participants = append(participants, subscriber.SubscriberID)
		}
	}

	return participants
}

func (s *SubscriptionChatSvc) CanSendMessage(userID, subscriptionID uint) (bool, string) {
	var sub models.RideSubscription
	if err := s.db.First(&sub, subscriptionID).Error; err != nil {
		return false, "Subscription not found"
	}

	if sub.Status != "active" {
		return false, "Subscription is not active"
	}

	if !s.isUserPartOfSubscription(userID, subscriptionID) {
		return false, "You are not part of this subscription"
	}

	return true, ""
}
