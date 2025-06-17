package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"mride-backend/internal/models"
)

type RideSvc struct {
	db              *sql.DB
	notificationSvc *NotificationSvc
}

func NewRideSvc(db *sql.DB, notificationSvc *NotificationSvc) *RideSvc {
	return &RideSvc{
		db:              db,
		notificationSvc: notificationSvc,
	}
}

func (r *RideSvc) CreateRide(userID int, req models.CreateRideReq) (*models.RideResp, error) {
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

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var ride models.Ride
	query := `INSERT INTO rides (user_id, car_number, car_model, passenger_count, price, 
			  from_location, to_location, from_latitude, from_longitude, 
			  to_latitude, to_longitude, departure_time, status) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'active') 
			  RETURNING id, user_id, car_number, car_model, passenger_count, price, 
			  from_location, to_location, from_latitude, from_longitude, 
			  to_latitude, to_longitude, departure_time, status, created_at, updated_at`

	err = tx.QueryRow(query, userID, req.CarNumber, req.CarModel, req.PassengerCount,
		req.Price, req.FromLocation, req.ToLocation, fromLat, fromLng,
		toLat, toLng, req.DepartureTime).Scan(
		&ride.ID, &ride.UserID, &ride.CarNumber, &ride.CarModel,
		&ride.PassengerCount, &ride.Price, &ride.FromLocation, &ride.ToLocation,
		&ride.FromLatitude, &ride.FromLongitude, &ride.ToLatitude, &ride.ToLongitude,
		&ride.DepartureTime, &ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	var user models.User
	userQuery := `SELECT id, full_name, email, phone, phone_verified, created_at, updated_at 
				  FROM users WHERE id = $1`
	err = tx.QueryRow(userQuery, userID).Scan(
		&user.ID, &user.FullName, &user.Email, &user.Phone,
		&user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &models.RideResp{
		Ride:           ride,
		User:           user,
		AvailableSeats: ride.PassengerCount,
	}, nil
}

func (r *RideSvc) GetRidesByUser(userID int) ([]models.Ride, error) {
	query := `SELECT id, user_id, car_number, car_model, passenger_count, price, 
			  from_location, to_location, from_latitude, from_longitude, 
			  to_latitude, to_longitude, departure_time, status, created_at, updated_at 
			  FROM rides WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rides []models.Ride
	for rows.Next() {
		var ride models.Ride
		err := rows.Scan(&ride.ID, &ride.UserID, &ride.CarNumber, &ride.CarModel,
			&ride.PassengerCount, &ride.Price, &ride.FromLocation, &ride.ToLocation,
			&ride.FromLatitude, &ride.FromLongitude, &ride.ToLatitude, &ride.ToLongitude,
			&ride.DepartureTime, &ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rides = append(rides, ride)
	}

	return rides, nil
}

func (r *RideSvc) GetRideByID(rideID int) (*models.RideResp, error) {
	var ride models.Ride
	var user models.User

	query := `SELECT r.id, r.user_id, r.car_number, r.car_model, r.passenger_count, r.price, 
			  r.from_location, r.to_location, r.from_latitude, r.from_longitude, 
			  r.to_latitude, r.to_longitude, r.departure_time, r.status, r.created_at, r.updated_at,
			  u.id, u.full_name, u.email, u.phone, u.phone_verified, u.created_at, u.updated_at
			  FROM rides r 
			  JOIN users u ON r.user_id = u.id 
			  WHERE r.id = $1`

	err := r.db.QueryRow(query, rideID).Scan(
		&ride.ID, &ride.UserID, &ride.CarNumber, &ride.CarModel,
		&ride.PassengerCount, &ride.Price, &ride.FromLocation, &ride.ToLocation,
		&ride.FromLatitude, &ride.FromLongitude, &ride.ToLatitude, &ride.ToLongitude,
		&ride.DepartureTime, &ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
		&user.ID, &user.FullName, &user.Email, &user.Phone,
		&user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	var joinedCount int
	countQuery := `SELECT COUNT(*) FROM ride_passengers WHERE ride_id = $1 AND status = 'active'`
	err = r.db.QueryRow(countQuery, rideID).Scan(&joinedCount)
	if err != nil {
		joinedCount = 0
	}

	joinedUsers, err := r.getJoinedUsers(rideID)
	if err != nil {
		joinedUsers = []models.User{}
	}

	return &models.RideResp{
		Ride:           ride,
		User:           user,
		JoinedUsers:    joinedUsers,
		AvailableSeats: ride.PassengerCount - joinedCount,
	}, nil
}

func (r *RideSvc) SearchRides(from, to string) ([]models.RideResp, error) {
	query := `SELECT r.id, r.user_id, r.car_number, r.car_model, r.passenger_count, r.price, 
              r.from_location, r.to_location, r.from_latitude, r.from_longitude, 
              r.to_latitude, r.to_longitude, r.departure_time, r.status, r.created_at, r.updated_at,
              u.id, u.full_name, u.email, u.phone, u.phone_verified, u.created_at, u.updated_at
              FROM rides r 
              JOIN users u ON r.user_id = u.id 
              WHERE r.status = 'active' 
              AND r.departure_time > NOW()
              AND r.from_location ILIKE $1
              AND r.to_location ILIKE $2`

	rows, err := r.db.Query(query, "%"+from+"%", "%"+to+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rides := []models.RideResp{}
	for rows.Next() {
		var ride models.Ride
		var user models.User

		err := rows.Scan(&ride.ID, &ride.UserID, &ride.CarNumber, &ride.CarModel,
			&ride.PassengerCount, &ride.Price, &ride.FromLocation, &ride.ToLocation,
			&ride.FromLatitude, &ride.FromLongitude, &ride.ToLatitude, &ride.ToLongitude,
			&ride.DepartureTime, &ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
			&user.ID, &user.FullName, &user.Email, &user.Phone,
			&user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}

		var joinedCount int
		countQuery := `SELECT COUNT(*) FROM ride_passengers WHERE ride_id = $1 AND status = 'active'`
		err = r.db.QueryRow(countQuery, ride.ID).Scan(&joinedCount)
		if err != nil {
			joinedCount = 0
		}

		rides = append(rides, models.RideResp{
			Ride:           ride,
			User:           user,
			AvailableSeats: ride.PassengerCount - joinedCount,
		})
	}
	return rides, nil
}

func (r *RideSvc) GetNearbyRides(userID int, req models.NearbyRidesReq) ([]models.RideResp, error) {
	if req.RadiusKM <= 0 {
		req.RadiusKM = 10
	}

	query := `
	SELECT 
		r.id, r.user_id, r.car_number, r.car_model, r.passenger_count, r.price, 
		r.from_location, r.to_location, r.from_latitude, r.from_longitude, 
		r.to_latitude, r.to_longitude, r.departure_time, r.status, r.created_at, r.updated_at,
		u.id, u.full_name, u.email, u.phone, u.phone_verified, u.created_at, u.updated_at,
		COALESCE((SELECT COUNT(*) FROM ride_passengers WHERE ride_id = r.id AND status = 'active'), 0) as joined_count,
		6371 * acos(
			GREATEST(-1, LEAST(1,
				cos(radians($1)) * 
				cos(radians(r.from_latitude)) * 
				cos(radians(r.from_longitude) - radians($2)) + 
				sin(radians($1)) * 
				sin(radians(r.from_latitude))
			))
		) as from_distance,
		6371 * acos(
			GREATEST(-1, LEAST(1,
				cos(radians($3)) * 
				cos(radians(r.to_latitude)) * 
				cos(radians(r.to_longitude) - radians($4)) + 
				sin(radians($3)) * 
				sin(radians(r.to_latitude))
			))
		) as to_distance
	FROM rides r 
	JOIN users u ON r.user_id = u.id 
	ORDER BY r.created_at DESC`

	rows, err := r.db.Query(query, req.FromLatitude, req.FromLongitude,
		req.ToLatitude, req.ToLongitude)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rides := []models.RideResp{}
	for rows.Next() {
		var ride models.Ride
		var user models.User
		var joinedCount int
		var fromDistance, toDistance float64

		err := rows.Scan(
			&ride.ID, &ride.UserID, &ride.CarNumber, &ride.CarModel,
			&ride.PassengerCount, &ride.Price, &ride.FromLocation, &ride.ToLocation,
			&ride.FromLatitude, &ride.FromLongitude, &ride.ToLatitude, &ride.ToLongitude,
			&ride.DepartureTime, &ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
			&user.ID, &user.FullName, &user.Email, &user.Phone,
			&user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
			&joinedCount, &fromDistance, &toDistance,
		)
		if err != nil {
			return nil, err
		}

		fmt.Printf("Ride ID: %d, From Distance: %.2f km, To Distance: %.2f km, Status: %s, Departure: %v\n",
			ride.ID, fromDistance, toDistance, ride.Status, ride.DepartureTime)

		rides = append(rides, models.RideResp{
			Ride:           ride,
			User:           user,
			AvailableSeats: ride.PassengerCount - joinedCount,
		})
	}

	return rides, nil
}

func (r *RideSvc) JoinRide(userID, rideID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var ride models.Ride
	rideQuery := `SELECT id, user_id, passenger_count, status FROM rides WHERE id = $1`
	err = tx.QueryRow(rideQuery, rideID).Scan(&ride.ID, &ride.UserID, &ride.PassengerCount, &ride.Status)
	if err != nil {
		if err == sql.ErrNoRows {
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

	var existingID int
	checkQuery := `SELECT id FROM ride_passengers WHERE ride_id = $1 AND passenger_id = $2 AND status = 'active'`
	err = tx.QueryRow(checkQuery, rideID, userID).Scan(&existingID)
	if err == nil {
		return fmt.Errorf("you have already joined this ride")
	}

	var joinedCount int
	countQuery := `SELECT COUNT(*) FROM ride_passengers WHERE ride_id = $1 AND status = 'active'`
	err = tx.QueryRow(countQuery, rideID).Scan(&joinedCount)
	if err != nil {
		return err
	}

	if joinedCount >= ride.PassengerCount {
		return fmt.Errorf("no available seats")
	}

	insertQuery := `INSERT INTO ride_passengers (ride_id, passenger_id, status) VALUES ($1, $2, 'active')`
	_, err = tx.Exec(insertQuery, rideID, userID)
	if err != nil {
		return err
	}

	var passengerName string
	nameQuery := `SELECT full_name FROM users WHERE id = $1`
	err = tx.QueryRow(nameQuery, userID).Scan(&passengerName)
	if err != nil {
		passengerName = "Unknown User"
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	if r.notificationSvc != nil {
		r.notificationSvc.CreateRideJoinNotification(ride.UserID, rideID, userID, passengerName)
	}

	return nil
}

func (r *RideSvc) GetAllUserRides(userID int) (*models.JoinedRidesResp, error) {
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
	query := `SELECT 
		r.id, r.user_id, r.car_number, r.car_model, r.passenger_count, r.price, 
		r.from_location, r.to_location, r.from_latitude, r.from_longitude, 
		r.to_latitude, r.to_longitude, r.departure_time, r.status, r.created_at, r.updated_at,
		u.id, u.full_name, u.email, u.phone, u.phone_verified, u.created_at, u.updated_at,
		COALESCE((SELECT COUNT(*) FROM ride_passengers WHERE ride_id = r.id AND status = 'active'), 0) as joined_count
	FROM rides r 
	JOIN users u ON r.user_id = u.id 
	ORDER BY r.created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rides := []models.RideResp{}
	for rows.Next() {
		var ride models.Ride
		var user models.User
		var joinedCount int

		err := rows.Scan(
			&ride.ID, &ride.UserID, &ride.CarNumber, &ride.CarModel,
			&ride.PassengerCount, &ride.Price, &ride.FromLocation, &ride.ToLocation,
			&ride.FromLatitude, &ride.FromLongitude, &ride.ToLatitude, &ride.ToLongitude,
			&ride.DepartureTime, &ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
			&user.ID, &user.FullName, &user.Email, &user.Phone,
			&user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
			&joinedCount,
		)
		if err != nil {
			return nil, err
		}

		rides = append(rides, models.RideResp{
			Ride:           ride,
			User:           user,
			AvailableSeats: ride.PassengerCount - joinedCount,
		})
	}
	return rides, nil
}

func (r *RideSvc) getCreatedRidesDetailed(userID int) ([]models.RideResp, error) {
	query := `SELECT r.id, r.user_id, r.car_number, r.car_model, r.passenger_count, r.price, 
			  r.from_location, r.to_location, r.from_latitude, r.from_longitude, 
			  r.to_latitude, r.to_longitude, r.departure_time, r.status, r.created_at, r.updated_at,
			  u.id, u.full_name, u.email, u.phone, u.phone_verified, u.created_at, u.updated_at
			  FROM rides r 
			  JOIN users u ON r.user_id = u.id 
			  WHERE r.user_id = $1 
			  ORDER BY r.created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rides := []models.RideResp{}
	for rows.Next() {
		var ride models.Ride
		var user models.User

		err := rows.Scan(&ride.ID, &ride.UserID, &ride.CarNumber, &ride.CarModel,
			&ride.PassengerCount, &ride.Price, &ride.FromLocation, &ride.ToLocation,
			&ride.FromLatitude, &ride.FromLongitude, &ride.ToLatitude, &ride.ToLongitude,
			&ride.DepartureTime, &ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
			&user.ID, &user.FullName, &user.Email, &user.Phone,
			&user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		var joinedCount int
		countQuery := `SELECT COUNT(*) FROM ride_passengers WHERE ride_id = $1 AND status = 'active'`
		err = r.db.QueryRow(countQuery, ride.ID).Scan(&joinedCount)
		if err != nil {
			joinedCount = 0
		}

		joinedUsers, err := r.getJoinedUsers(ride.ID)
		if err != nil {
			joinedUsers = []models.User{}
		}

		rides = append(rides, models.RideResp{
			Ride:           ride,
			User:           user,
			JoinedUsers:    joinedUsers,
			AvailableSeats: ride.PassengerCount - joinedCount,
		})
	}

	return rides, nil
}

func (r *RideSvc) getJoinedRidesDetailed(userID int) ([]models.RideResp, error) {
	query := `SELECT r.id, r.user_id, r.car_number, r.car_model, r.passenger_count, r.price, 
			  r.from_location, r.to_location, r.from_latitude, r.from_longitude, 
			  r.to_latitude, r.to_longitude, r.departure_time, r.status, r.created_at, r.updated_at,
			  u.id, u.full_name, u.email, u.phone, u.phone_verified, u.created_at, u.updated_at
			  FROM rides r 
			  JOIN users u ON r.user_id = u.id 
			  JOIN ride_passengers rp ON r.id = rp.ride_id
			  WHERE rp.passenger_id = $1 AND rp.status = 'active'
			  ORDER BY r.created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rides []models.RideResp
	for rows.Next() {
		var ride models.Ride
		var user models.User

		err := rows.Scan(&ride.ID, &ride.UserID, &ride.CarNumber, &ride.CarModel,
			&ride.PassengerCount, &ride.Price, &ride.FromLocation, &ride.ToLocation,
			&ride.FromLatitude, &ride.FromLongitude, &ride.ToLatitude, &ride.ToLongitude,
			&ride.DepartureTime, &ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
			&user.ID, &user.FullName, &user.Email, &user.Phone,
			&user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		var joinedCount int
		countQuery := `SELECT COUNT(*) FROM ride_passengers WHERE ride_id = $1 AND status = 'active'`
		err = r.db.QueryRow(countQuery, ride.ID).Scan(&joinedCount)
		if err != nil {
			joinedCount = 0
		}

		rides = append(rides, models.RideResp{
			Ride:           ride,
			User:           user,
			AvailableSeats: ride.PassengerCount - joinedCount,
		})
	}

	return rides, nil
}

func (r *RideSvc) getJoinedUsers(rideID int) ([]models.User, error) {
	query := `SELECT u.id, u.full_name, u.email, u.phone, u.phone_verified, u.created_at, u.updated_at
			  FROM users u 
			  JOIN ride_passengers rp ON u.id = rp.passenger_id
			  WHERE rp.ride_id = $1 AND rp.status = 'active'`

	rows, err := r.db.Query(query, rideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.FullName, &user.Email, &user.Phone,
			&user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
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
