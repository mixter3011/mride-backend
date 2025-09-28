package models

import (
	"time"
)

type RideSubscription struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	UserID           uint       `json:"user_id" gorm:"not null;index"`
	Title            string     `json:"title" gorm:"size:100;not null"`
	Description      string     `json:"description" gorm:"size:500"`
	CarNumber        string     `json:"car_number" gorm:"size:20;not null"`
	CarModel         string     `json:"car_model" gorm:"size:50;not null"`
	PassengerCount   int        `json:"passenger_count" gorm:"not null;check:passenger_count > 0 AND passenger_count <= 8"`
	Price            *float64   `json:"price" gorm:"check:price >= 0"`
	FromLocation     string     `json:"from_location" gorm:"size:255;not null"`
	ToLocation       string     `json:"to_location" gorm:"size:255;not null"`
	FromLatitude     float64    `json:"from_latitude" gorm:"not null"`
	FromLongitude    float64    `json:"from_longitude" gorm:"not null"`
	ToLatitude       float64    `json:"to_latitude" gorm:"not null"`
	ToLongitude      float64    `json:"to_longitude" gorm:"not null"`
	DepartureTime    string     `json:"departure_time" gorm:"size:5;not null"`
	RecurringDays    string     `json:"recurring_days" gorm:"size:255;not null"`
	StartDate        time.Time  `json:"start_date" gorm:"not null"`
	EndDate          *time.Time `json:"end_date,omitempty"`
	Status           string     `json:"status" gorm:"size:20;default:'active';index"`
	MaxSubscribers   int        `json:"max_subscribers" gorm:"default:0"`
	NotificationTime int        `json:"notification_time" gorm:"default:60"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	User        User                     `json:"driver,omitempty" gorm:"foreignKey:UserID"`
	Subscribers []SubscriptionSubscriber `json:"subscribers,omitempty" gorm:"foreignKey:SubscriptionID"`
}

type SubscriptionSubscriber struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	SubscriptionID uint      `json:"subscription_id" gorm:"not null;index"`
	SubscriberID   uint      `json:"subscriber_id" gorm:"not null;index"`
	Status         string    `json:"status" gorm:"size:20;default:'active'"`
	NotifyEnabled  bool      `json:"notify_enabled" gorm:"default:true"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Subscription RideSubscription `json:"subscription,omitempty" gorm:"foreignKey:SubscriptionID"`
	Subscriber   User             `json:"subscriber,omitempty" gorm:"foreignKey:SubscriberID"`
}

type SubscriptionNotification struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	SubscriptionID   uint      `json:"subscription_id" gorm:"not null;index"`
	SubscriberID     uint      `json:"subscriber_id" gorm:"not null;index"`
	RideDate         time.Time `json:"ride_date" gorm:"not null"`
	NotificationSent bool      `json:"notification_sent" gorm:"default:false"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateSubscriptionReq struct {
	Title            string     `json:"title" binding:"required,min=3,max=100"`
	Description      string     `json:"description" binding:"max=500"`
	CarNumber        string     `json:"car_number" binding:"required,min=1,max=20"`
	CarModel         string     `json:"car_model" binding:"required,min=1,max=50"`
	PassengerCount   int        `json:"passenger_count" binding:"required,min=1,max=8"`
	Price            *float64   `json:"price"`
	FromLocation     string     `json:"from_location" binding:"required,min=3,max=255"`
	ToLocation       string     `json:"to_location" binding:"required,min=3,max=255"`
	DepartureTime    string     `json:"departure_time" binding:"required,len=5"`
	RecurringDays    []string   `json:"recurring_days" binding:"required,min=1,max=7"`
	StartDate        time.Time  `json:"start_date" binding:"required"`
	EndDate          *time.Time `json:"end_date,omitempty"`
	MaxSubscribers   int        `json:"max_subscribers" binding:"min=0"`
	NotificationTime int        `json:"notification_time" binding:"min=1,max=1440"`
}

type UpdateSubscriptionReq struct {
	Title            *string    `json:"title,omitempty" binding:"omitempty,min=3,max=100"`
	Description      *string    `json:"description,omitempty" binding:"omitempty,max=500"`
	Price            *float64   `json:"price,omitempty"`
	DepartureTime    *string    `json:"departure_time,omitempty" binding:"omitempty,len=5"`
	RecurringDays    *[]string  `json:"recurring_days,omitempty" binding:"omitempty,min=1,max=7"`
	EndDate          *time.Time `json:"end_date,omitempty"`
	MaxSubscribers   *int       `json:"max_subscribers,omitempty" binding:"omitempty,min=0"`
	NotificationTime *int       `json:"notification_time,omitempty" binding:"omitempty,min=1,max=1440"`
	Status           *string    `json:"status,omitempty" binding:"omitempty,oneof=active inactive paused"`
}

type SubscriptionResp struct {
	Subscription    RideSubscription `json:"subscription"`
	Driver          User             `json:"driver"`
	Subscribers     []User           `json:"subscribers,omitempty"`
	AvailableSlots  int              `json:"available_slots"`
	SubscriberCount int              `json:"subscriber_count"`
	NextRideDates   []time.Time      `json:"next_ride_dates,omitempty"`
	IsSubscribed    bool             `json:"is_subscribed,omitempty"`
}

type SearchSubscriptionsReq struct {
	From          string `form:"from"`
	To            string `form:"to"`
	DepartureTime string `form:"departure_time"`
	WeekDay       string `form:"weekday"`
}

type NearbySubscriptionsReq struct {
	FromLatitude  float64 `json:"from_latitude" binding:"required"`
	FromLongitude float64 `json:"from_longitude" binding:"required"`
	ToLatitude    float64 `json:"to_latitude" binding:"required"`
	ToLongitude   float64 `json:"to_longitude" binding:"required"`
	RadiusKM      float64 `json:"radius_km"`
	DepartureTime string  `json:"departure_time,omitempty"`
	WeekDay       string  `json:"weekday,omitempty"`
}

type SubscriptionStatsResp struct {
	TotalSubscriptions  int `json:"total_subscriptions"`
	ActiveSubscriptions int `json:"active_subscriptions"`
	TotalSubscribers    int `json:"total_subscribers"`
	TodaysRides         int `json:"todays_rides"`
}
