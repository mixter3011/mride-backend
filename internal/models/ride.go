package models

import (
	"time"
)

type Ride struct {
	ID             int       `json:"id" db:"id"`
	UserID         int       `json:"user_id" db:"user_id"`
	CarNumber      string    `json:"car_number" db:"car_number"`
	CarModel       string    `json:"car_model" db:"car_model"`
	PassengerCount int       `json:"passenger_count" db:"passenger_count"`
	Price          *float64  `json:"price" db:"price"`
	FromLocation   string    `json:"from_location" db:"from_location"`
	ToLocation     string    `json:"to_location" db:"to_location"`
	FromLatitude   float64   `json:"from_latitude" db:"from_latitude"`
	FromLongitude  float64   `json:"from_longitude" db:"from_longitude"`
	ToLatitude     float64   `json:"to_latitude" db:"to_latitude"`
	ToLongitude    float64   `json:"to_longitude" db:"to_longitude"`
	DepartureTime  time.Time `json:"departure_time" db:"departure_time"`
	Status         string    `json:"status" db:"status"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type RidePassenger struct {
	ID          int       `json:"id" db:"id"`
	RideID      int       `json:"ride_id" db:"ride_id"`
	PassengerID int       `json:"passenger_id" db:"passenger_id"`
	JoinedAt    time.Time `json:"joined_at" db:"joined_at"`
	Status      string    `json:"status" db:"status"`
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

type NearbyRidesReq struct {
	FromLatitude  float64 `json:"from_latitude" binding:"required"`
	FromLongitude float64 `json:"from_longitude" binding:"required"`
	ToLatitude    float64 `json:"to_latitude" binding:"required"`
	ToLongitude   float64 `json:"to_longitude" binding:"required"`
	RadiusKM      float64 `json:"radius_km"`
}

type RideResp struct {
	Ride           Ride            `json:"ride"`
	User           User            `json:"driver"`
	Passengers     []RidePassenger `json:"passengers,omitempty"`
	JoinedUsers    []User          `json:"joined_users,omitempty"`
	AvailableSeats int             `json:"available_seats"`
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
