package models

import (
	"encoding/json"
	"time"
)

type Notification struct {
	ID        uint            `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint            `json:"user_id" gorm:"not null;index"`
	Type      string          `json:"type" gorm:"not null;size:50"`
	Title     string          `json:"title" gorm:"not null;size:255"`
	Message   string          `json:"message" gorm:"not null;type:text"`
	Data      json.RawMessage `json:"data" gorm:"type:jsonb"`
	Read      bool            `json:"read" gorm:"default:false;index"`
	CreatedAt time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

type NotificationData struct {
	RideID     uint   `json:"ride_id,omitempty"`
	UserID     uint   `json:"user_id,omitempty"`
	UserName   string `json:"user_name,omitempty"`
	ActionType string `json:"action_type,omitempty"`
}

type NotificationsResp struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unread_count"`
}

const (
	NotificationTypeRideJoin   = "ride_join"
	NotificationTypeRideDelete = "ride_delete"
	NotificationTypeRideLeave  = "ride_leave"
)
