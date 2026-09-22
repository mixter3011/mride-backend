package models

import (
	"time"
)

type Ride struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	UserID            uint       `json:"user_id" gorm:"not null;index"`
	CarNumber         string     `json:"car_number" gorm:"size:20;not null"`
	CarModel          string     `json:"car_model" gorm:"size:50;not null"`
	PassengerCount    int        `json:"passenger_count" gorm:"not null;check:passenger_count > 0 AND passenger_count <= 8"`
	Price             *float64   `json:"price" gorm:"check:price >= 0"`
	FromLocation      string     `json:"from_location" gorm:"size:255;not null"`
	ToLocation        string     `json:"to_location" gorm:"size:255;not null"`
	FromLatitude      float64    `json:"from_latitude" gorm:"not null"`
	FromLongitude     float64    `json:"from_longitude" gorm:"not null"`
	ToLatitude        float64    `json:"to_latitude" gorm:"not null"`
	ToLongitude       float64    `json:"to_longitude" gorm:"not null"`
	DepartureTime     time.Time  `json:"departure_time" gorm:"not null;index"`
	Status            string     `json:"status" gorm:"size:20;default:'active';index"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	EstimatedDuration int        `json:"estimated_duration,omitempty"`

	User       User            `json:"driver,omitempty" gorm:"foreignKey:UserID"`
	Passengers []RidePassenger `json:"passengers,omitempty" gorm:"foreignKey:RideID"`
}
type RidePassenger struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	RideID      uint      `json:"ride_id" gorm:"not null;index"`
	PassengerID uint      `json:"passenger_id" gorm:"not null;index"`
	Status      string    `json:"status" gorm:"size:20;default:'active'"`
	CreatedAt   time.Time `json:"joined_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Ride      Ride `json:"ride,omitempty" gorm:"foreignKey:RideID"`
	Passenger User `json:"passenger,omitempty" gorm:"foreignKey:PassengerID"`
}
type CreateRideReq struct {
	CarNumber      string    `json:"car_number" binding:"required,min=1,max=20"`
	CarModel       string    `json:"car_model" binding:"required,min=1,max=50"`
	PassengerCount int       `json:"passenger_count" binding:"required,min=1,max=8"`
	Price          *float64  `json:"price"`
	FromLocation   string    `json:"from_location" binding:"required,min=3,max=255"`
	ToLocation     string    `json:"to_location" binding:"required,min=3,max=255"`
	DepartureTime  time.Time `json:"departure_time" binding:"required"`
}
type SearchRidesReq struct {
	From string `form:"from"`
	To   string `form:"to"`
}

type SearchByLocationReq struct {
	FromLatitude  float64 `form:"from_lat" binding:"required"`
	FromLongitude float64 `form:"from_lng" binding:"required"`
	ToLatitude    float64 `form:"to_lat" binding:"required"`
	ToLongitude   float64 `form:"to_lng" binding:"required"`
	RadiusKM      float64 `form:"radius"`
}
type NearbyRidesReq struct {
	FromLatitude        float64 `json:"from_latitude" binding:"required"`
	FromLongitude       float64 `json:"from_longitude" binding:"required"`
	ToLatitude          float64 `json:"to_latitude" binding:"required"`
	ToLongitude         float64 `json:"to_longitude" binding:"required"`
	RadiusKM            float64 `json:"radius_km"`
	MaxRouteDeviationKM float64 `json:"max_route_deviation_km"`
}
type RideResp struct {
	Ride              Ride    `json:"ride"`
	User              User    `json:"driver"`
	JoinedUsers       []User  `json:"joined_users,omitempty"`
	AvailableSeats    int     `json:"available_seats"`
	HasPassengers     bool    `json:"has_passengers"`
	MatchScore        float64 `json:"match_score,omitempty"`
	PickupDistanceKm  float64 `json:"pickup_distance_km,omitempty"`
	DropoffDistanceKm float64 `json:"dropoff_distance_km,omitempty"`
}
type JoinedRidesResp struct {
	CreatedRides []RideResp `json:"created_rides"`
	JoinedRides  []RideResp `json:"joined_rides"`
}
type GeocodingResp struct {
	Results []struct {
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
}
type StartRideReq struct {
	EstimatedDuration int     `json:"estimated_duration" binding:"required,min=1,max=600"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
}
type RideStatusUpdateReq struct {
	RemainingTime int     `json:"remaining_time"`
	CurrentLat    float64 `json:"current_lat"`
	CurrentLng    float64 `json:"current_lng"`
}
type RideProgressResp struct {
	RideID            uint       `json:"ride_id"`
	Status            string     `json:"status"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	EstimatedDuration int        `json:"estimated_duration"`
	RemainingTime     int        `json:"remaining_time"`
	ProgressPercent   float64    `json:"progress_percent"`
	CurrentLocation   struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"current_location,omitempty"`
}
