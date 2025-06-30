package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"mride-backend/internal/models"

	"gorm.io/gorm"
)

type RideSvc struct {
	db              *gorm.DB
	notificationSvc *NotificationSvc
}

func NewRideSvc(db *gorm.DB, notificationSvc *NotificationSvc) *RideSvc {
	return &RideSvc{
		db:              db,
		notificationSvc: notificationSvc,
	}
}

func (r *RideSvc) CreateRide(userID uint, req models.CreateRideReq) (*models.RideResp, error) {
	if req.DepartureTime.Before(time.Now()) {
		return nil, fmt.Errorf("departure time must be in the future")
	}

	if req.Price != nil && *req.Price < 0 {
		return nil, fmt.Errorf("price cannot be negative")
	}

	fromLat, fromLng, err := r.getCoordinates(req.FromLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinates for from location: %v", err)
	}

	toLat, toLng, err := r.getCoordinates(req.ToLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinates for to location: %v", err)
	}

	ride := models.Ride{
		UserID:         userID,
		CarNumber:      req.CarNumber,
		CarModel:       req.CarModel,
		PassengerCount: req.PassengerCount,
		Price:          req.Price,
		FromLocation:   req.FromLocation,
		ToLocation:     req.ToLocation,
		FromLatitude:   fromLat,
		FromLongitude:  fromLng,
		ToLatitude:     toLat,
		ToLongitude:    toLng,
		DepartureTime:  req.DepartureTime,
		Status:         "active",
	}

	if err := r.db.Create(&ride).Error; err != nil {
		return nil, err
	}

	var user models.User
	if err := r.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	return &models.RideResp{
		Ride:           ride,
		User:           user,
		AvailableSeats: ride.PassengerCount,
	}, nil
}

func (r *RideSvc) DeleteRide(userID, rideID uint) error {
	var ride models.Ride
	if err := r.db.First(&ride, rideID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("ride not found")
		}
		return err
	}

	if ride.UserID != userID {
		return fmt.Errorf("you can only delete your own rides")
	}

	var passengers []models.RidePassenger
	r.db.Where("ride_id = ? AND status = ?", rideID, "active").Find(&passengers)

	passengerIDs := make([]uint, len(passengers))
	for i, p := range passengers {
		passengerIDs[i] = p.PassengerID
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ride_id = ?", rideID).Delete(&models.RidePassenger{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&ride).Error; err != nil {
			return err
		}

		if r.notificationSvc != nil {
			for _, passengerID := range passengerIDs {
				r.notificationSvc.CreateRideDeletedNotification(passengerID, rideID, userID)
			}
		}

		return nil
	})
}

func (r *RideSvc) GetRidesByUser(userID uint) ([]models.Ride, error) {
	var rides []models.Ride
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&rides).Error
	return rides, err
}

func (r *RideSvc) GetRideByID(rideID uint) (*models.RideResp, error) {
	var ride models.Ride
	if err := r.db.Preload("User").First(&ride, rideID).Error; err != nil {
		return nil, err
	}

	var joinedCount int64
	r.db.Model(&models.RidePassenger{}).Where("ride_id = ? AND status = ?", rideID, "active").Count(&joinedCount)

	joinedUsers, err := r.getJoinedUsers(rideID)
	if err != nil {
		joinedUsers = []models.User{}
	}

	return &models.RideResp{
		Ride:           ride,
		User:           ride.User,
		JoinedUsers:    joinedUsers,
		AvailableSeats: ride.PassengerCount - int(joinedCount),
	}, nil
}

func (r *RideSvc) SearchRides(from, to string) ([]models.RideResp, error) {
	var rides []models.Ride
	err := r.db.Preload("User").
		Where("status = ? AND departure_time > ? AND from_location ILIKE ? AND to_location ILIKE ?",
			"active", time.Now(), "%"+from+"%", "%"+to+"%").
		Find(&rides).Error

	if err != nil {
		return nil, err
	}

	var rideResponses []models.RideResp
	for _, ride := range rides {
		var joinedCount int64
		r.db.Model(&models.RidePassenger{}).Where("ride_id = ? AND status = ?", ride.ID, "active").Count(&joinedCount)

		rideResponses = append(rideResponses, models.RideResp{
			Ride:           ride,
			User:           ride.User,
			AvailableSeats: ride.PassengerCount - int(joinedCount),
		})
	}

	return rideResponses, nil
}

func (r *RideSvc) GetNearbyRides(userID uint, req models.NearbyRidesReq) ([]models.RideResp, error) {
	if req.RadiusKM <= 0 {
		req.RadiusKM = 10
	}

	var rides []models.Ride
	err := r.db.Preload("User").Order("created_at DESC").Find(&rides).Error
	if err != nil {
		return nil, err
	}

	var rideResponses []models.RideResp
	for _, ride := range rides {
		var joinedCount int64
		r.db.Model(&models.RidePassenger{}).Where("ride_id = ? AND status = ?", ride.ID, "active").Count(&joinedCount)

		fromDistance := r.calculateDistance(req.FromLatitude, req.FromLongitude, ride.FromLatitude, ride.FromLongitude)
		toDistance := r.calculateDistance(req.ToLatitude, req.ToLongitude, ride.ToLatitude, ride.ToLongitude)

		fmt.Printf("Ride ID: %d, From Distance: %.2f km, To Distance: %.2f km, Status: %s, Departure: %v\n",
			ride.ID, fromDistance, toDistance, ride.Status, ride.DepartureTime)

		rideResponses = append(rideResponses, models.RideResp{
			Ride:           ride,
			User:           ride.User,
			AvailableSeats: ride.PassengerCount - int(joinedCount),
		})
	}

	return rideResponses, nil
}

func (r *RideSvc) JoinRide(userID, rideID uint) error {
	var ride models.Ride
	if err := r.db.First(&ride, rideID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("ride not found")
		}
		return err
	}

	if ride.UserID == userID {
		return fmt.Errorf("cannot join your own ride")
	}

	if ride.Status != "active" {
		return fmt.Errorf("ride is not active")
	}

	var existing models.RidePassenger
	if err := r.db.Where("ride_id = ? AND passenger_id = ? AND status = ?", rideID, userID, "active").First(&existing).Error; err == nil {
		return fmt.Errorf("you have already joined this ride")
	}

	var joinedCount int64
	r.db.Model(&models.RidePassenger{}).Where("ride_id = ? AND status = ?", rideID, "active").Count(&joinedCount)

	if int(joinedCount) >= ride.PassengerCount {
		return fmt.Errorf("no available seats")
	}

	passenger := models.RidePassenger{
		RideID:      rideID,
		PassengerID: userID,
		Status:      "active",
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&passenger).Error; err != nil {
			return err
		}

		var user models.User
		passengerName := "Unknown User"
		if err := tx.First(&user, userID).Error; err == nil {
			passengerName = user.FullName
		}

		if r.notificationSvc != nil {
			r.notificationSvc.CreateRideJoinNotification(ride.UserID, rideID, userID, passengerName)
		}

		return nil
	})
}

func (r *RideSvc) LeaveRide(userID, rideID uint) error {
	var ride models.Ride
	if err := r.db.First(&ride, rideID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("ride not found")
		}
		return err
	}

	if ride.UserID == userID {
		return fmt.Errorf("you cannot leave your own ride, use delete instead")
	}

	var passenger models.RidePassenger
	if err := r.db.Where("ride_id = ? AND passenger_id = ? AND status = ?", rideID, userID, "active").First(&passenger).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("you have not joined this ride")
		}
		return err
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&passenger).Error; err != nil {
			return err
		}

		var user models.User
		userName := "Unknown User"
		if err := tx.First(&user, userID).Error; err == nil {
			userName = user.FullName
		}

		if r.notificationSvc != nil {
			r.notificationSvc.CreateRideLeaveNotification(ride.UserID, rideID, userID, userName)
		}

		return nil
	})
}

func (r *RideSvc) GetAllUserRides(userID uint) (*models.JoinedRidesResp, error) {
	createdRides, err := r.getCreatedRidesDetailed(userID)
	if err != nil {
		return nil, err
	}

	joinedRides, err := r.getJoinedRidesDetailed(userID)
	if err != nil {
		return nil, err
	}

	return &models.JoinedRidesResp{
		CreatedRides: createdRides,
		JoinedRides:  joinedRides,
	}, nil
}

func (r *RideSvc) GetAllRides() ([]models.RideResp, error) {
	var rides []models.Ride
	err := r.db.Preload("User").Order("created_at DESC").Find(&rides).Error
	if err != nil {
		return nil, err
	}

	var rideResponses []models.RideResp
	for _, ride := range rides {
		var joinedCount int64
		r.db.Model(&models.RidePassenger{}).Where("ride_id = ? AND status = ?", ride.ID, "active").Count(&joinedCount)

		rideResponses = append(rideResponses, models.RideResp{
			Ride:           ride,
			User:           ride.User,
			AvailableSeats: ride.PassengerCount - int(joinedCount),
		})
	}

	return rideResponses, nil
}

func (r *RideSvc) getCreatedRidesDetailed(userID uint) ([]models.RideResp, error) {
	var rides []models.Ride
	err := r.db.Preload("User").Where("user_id = ?", userID).Order("created_at DESC").Find(&rides).Error
	if err != nil {
		return nil, err
	}

	var rideResponses []models.RideResp
	for _, ride := range rides {
		var joinedCount int64
		r.db.Model(&models.RidePassenger{}).Where("ride_id = ? AND status = ?", ride.ID, "active").Count(&joinedCount)

		joinedUsers, err := r.getJoinedUsers(ride.ID)
		if err != nil {
			joinedUsers = []models.User{}
		}

		rideResponses = append(rideResponses, models.RideResp{
			Ride:           ride,
			User:           ride.User,
			JoinedUsers:    joinedUsers,
			AvailableSeats: ride.PassengerCount - int(joinedCount),
		})
	}

	return rideResponses, nil
}

func (r *RideSvc) getJoinedRidesDetailed(userID uint) ([]models.RideResp, error) {
	var rides []models.Ride
	err := r.db.Preload("User").
		Joins("JOIN ride_passengers ON rides.id = ride_passengers.ride_id").
		Where("ride_passengers.passenger_id = ? AND ride_passengers.status = ?", userID, "active").
		Order("rides.created_at DESC").
		Find(&rides).Error

	if err != nil {
		return nil, err
	}

	var rideResponses []models.RideResp
	for _, ride := range rides {
		var joinedCount int64
		r.db.Model(&models.RidePassenger{}).Where("ride_id = ? AND status = ?", ride.ID, "active").Count(&joinedCount)

		rideResponses = append(rideResponses, models.RideResp{
			Ride:           ride,
			User:           ride.User,
			AvailableSeats: ride.PassengerCount - int(joinedCount),
		})
	}

	return rideResponses, nil
}

func (r *RideSvc) getJoinedUsers(rideID uint) ([]models.User, error) {
	var users []models.User
	err := r.db.Joins("JOIN ride_passengers ON users.id = ride_passengers.passenger_id").
		Where("ride_passengers.ride_id = ? AND ride_passengers.status = ?", rideID, "active").
		Find(&users).Error

	return users, err
}

func (r *RideSvc) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371

	lat1Rad := lat1 * (3.14159265359 / 180)
	lon1Rad := lon1 * (3.14159265359 / 180)
	lat2Rad := lat2 * (3.14159265359 / 180)
	lon2Rad := lon2 * (3.14159265359 / 180)

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := 0.5 - 0.5*((deltaLat*0.5)*(deltaLat*0.5)+(deltaLon*0.5)*(deltaLon*0.5))
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}

	return earthRadius * 2 * (a * a)
}

func (r *RideSvc) getCoordinates(address string) (float64, float64, error) {
	baseURL := "https://nominatim.openstreetmap.org/search"
	params := url.Values{}
	params.Add("q", address)
	params.Add("format", "json")
	params.Add("limit", "1")

	client := &http.Client{}
	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return 0, 0, err
	}

	req.Header.Set("User-Agent", "CarPoolApp/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, err
	}

	if len(results) == 0 {
		return 0, 0, fmt.Errorf("location not found: %s", address)
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return 0, 0, err
	}

	lng, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return 0, 0, err
	}

	return lat, lng, nil
}
