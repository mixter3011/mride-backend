package models

import (
	"encoding/json"
	"time"
)

type Notification struct {
	ID        int             `json:"id" db:"id"`
	UserID    int             `json:"user_id" db:"user_id"`
	Type      string          `json:"type" db:"type"`
	Title     string          `json:"title" db:"title"`
	Message   string          `json:"message" db:"message"`
	Data      json.RawMessage `json:"data" db:"data"`
	Read      bool            `json:"read" db:"read"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

type NotificationData struct {
	RideID     int    `json:"ride_id,omitempty"`
	UserID     int    `json:"user_id,omitempty"`
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
