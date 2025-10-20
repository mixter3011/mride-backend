package models

import (
	"time"
)

type RideChat struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RideID    uint      `gorm:"not null;index" json:"ride_id"`
	SenderID  uint      `gorm:"not null;index" json:"sender_id"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Sender User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Ride   Ride `gorm:"foreignKey:RideID" json:"ride,omitempty"`
}

func (RideChat) TableName() string {
	return "ride_chats"
}

type SendChatMessageReq struct {
	Message string `json:"message" binding:"required,min=1,max=500"`
}

type ChatMessageResp struct {
	ID        uint      `json:"id"`
	RideID    uint      `json:"ride_id"`
	SenderID  uint      `json:"sender_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	IsMine    bool      `json:"is_mine"`
	Sender    struct {
		ID       uint   `json:"id"`
		FullName string `json:"full_name"`
	} `json:"sender"`
}

type GetChatHistoryResp struct {
	Messages []ChatMessageResp `json:"messages"`
	Count    int               `json:"count"`
}

type ChatRoomInfo struct {
	RideID          uint      `json:"ride_id"`
	OtherUserID     uint      `json:"other_user_id"`
	OtherUserName   string    `json:"other_user_name"`
	LastMessage     string    `json:"last_message"`
	LastMessageTime time.Time `json:"last_message_time"`
	UnreadCount     int       `json:"unread_count"`
	RideStatus      string    `json:"ride_status"`
	FromLocation    string    `json:"from_location"`
	ToLocation      string    `json:"to_location"`
	DepartureTime   time.Time `json:"departure_time"`
	IsDriver        bool      `json:"is_driver"`
}
