package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"mride-backend/internal/models"

	"gorm.io/gorm"
)

type SubscriptionSvc struct {
	db              *gorm.DB
	notificationSvc NotificationSvcInterface
}

func NewSubscriptionSvc(db *gorm.DB, notificationSvc NotificationSvcInterface) *SubscriptionSvc {
	return &SubscriptionSvc{
		db:              db,
		notificationSvc: notificationSvc,
	}
}

func (s *SubscriptionSvc) CreateSubscription(userID uint, req models.CreateSubscriptionReq) (*models.SubscriptionResp, error) {
	if !s.isValidTimeFormat(req.DepartureTime) {
		return nil, fmt.Errorf("invalid departure time format, use HH:MM")
	}

	if err := s.validateRecurringDays(req.RecurringDays); err != nil {
		return nil, err
	}

	if req.StartDate.Before(time.Now().UTC().Truncate(24 * time.Hour)) {
		return nil, fmt.Errorf("start date cannot be in the past")
	}

	if req.EndDate != nil && req.EndDate.Before(req.StartDate) {
		return nil, fmt.Errorf("end date cannot be before start date")
	}

	if req.Price != nil && *req.Price < 0 {
		return nil, fmt.Errorf("price cannot be negative")
	}

	fromLat, fromLng, err := s.getCoordinates(req.FromLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinates for from location: %v", err)
	}

	toLat, toLng, err := s.getCoordinates(req.ToLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinates for to location: %v", err)
	}

	subscription := models.RideSubscription{
		UserID:           userID,
		Title:            req.Title,
		Description:      req.Description,
		CarNumber:        req.CarNumber,
		CarModel:         req.CarModel,
		PassengerCount:   req.PassengerCount,
		Price:            req.Price,
		FromLocation:     req.FromLocation,
		ToLocation:       req.ToLocation,
		FromLatitude:     fromLat,
		FromLongitude:    fromLng,
		ToLatitude:       toLat,
		ToLongitude:      toLng,
		DepartureTime:    req.DepartureTime,
		RecurringDays:    models.StringArray(req.RecurringDays),
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
		Status:           "active",
		MaxSubscribers:   req.MaxSubscribers,
		NotificationTime: req.NotificationTime,
	}

	if err := s.db.Create(&subscription).Error; err != nil {
		return nil, err
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	nextRides := s.getNextRideDates(subscription, 5)

	return &models.SubscriptionResp{
		Subscription:    subscription,
		Driver:          user,
		AvailableSlots:  subscription.MaxSubscribers,
		SubscriberCount: 0,
		NextRideDates:   nextRides,
	}, nil
}

func (s *SubscriptionSvc) GetSubscription(subscriptionID uint, userID *uint) (*models.SubscriptionResp, error) {
	var subscription models.RideSubscription
	if err := s.db.Preload("User").First(&subscription, subscriptionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, err
	}

	var subscriberCount int64
	s.db.Model(&models.SubscriptionSubscriber{}).
		Where("subscription_id = ? AND status = ?", subscriptionID, "active").
		Count(&subscriberCount)

	subscribers, err := s.getSubscribers(subscriptionID)
	if err != nil {
		subscribers = []models.User{}
	}

	isSubscribed := false
	if userID != nil {
		var count int64
		s.db.Model(&models.SubscriptionSubscriber{}).
			Where("subscription_id = ? AND subscriber_id = ? AND status = ?",
				subscriptionID, *userID, "active").
			Count(&count)
		isSubscribed = count > 0
	}

	nextRides := s.getNextRideDates(subscription, 5)

	availableSlots := subscription.MaxSubscribers - int(subscriberCount)
	if availableSlots < 0 {
		availableSlots = 0
	}

	return &models.SubscriptionResp{
		Subscription:    subscription,
		Driver:          subscription.User,
		Subscribers:     subscribers,
		AvailableSlots:  availableSlots,
		SubscriberCount: int(subscriberCount),
		NextRideDates:   nextRides,
		IsSubscribed:    isSubscribed,
	}, nil
}

func (s *SubscriptionSvc) UpdateSubscription(userID, subscriptionID uint, req models.UpdateSubscriptionReq) error {
	var subscription models.RideSubscription
	if err := s.db.First(&subscription, subscriptionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("subscription not found")
		}
		return err
	}

	if subscription.UserID != userID {
		return fmt.Errorf("you can only update your own subscriptions")
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}

	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if req.Price != nil {
		if *req.Price < 0 {
			return fmt.Errorf("price cannot be negative")
		}
		updates["price"] = *req.Price
	}

	if req.DepartureTime != nil {
		if !s.isValidTimeFormat(*req.DepartureTime) {
			return fmt.Errorf("invalid departure time format, use HH:MM")
		}
		updates["departure_time"] = *req.DepartureTime
	}

	if req.RecurringDays != nil {
		if err := s.validateRecurringDays(*req.RecurringDays); err != nil {
			return err
		}
		updates["recurring_days"] = models.StringArray(*req.RecurringDays)
	}

	if req.EndDate != nil {
		if req.EndDate.Before(subscription.StartDate) {
			return fmt.Errorf("end date cannot be before start date")
		}
		updates["end_date"] = *req.EndDate
	}

	if req.MaxSubscribers != nil {
		var currentCount int64
		s.db.Model(&models.SubscriptionSubscriber{}).
			Where("subscription_id = ? AND status = ?", subscriptionID, "active").
			Count(&currentCount)

		if *req.MaxSubscribers < int(currentCount) {
			return fmt.Errorf("cannot set max subscribers below current subscriber count (%d)", currentCount)
		}
		updates["max_subscribers"] = *req.MaxSubscribers
	}

	if req.NotificationTime != nil {
		updates["notification_time"] = *req.NotificationTime
	}

	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		return s.db.Model(&subscription).Updates(updates).Error
	}

	return nil
}

func (s *SubscriptionSvc) DeleteSubscription(userID, subscriptionID uint) error {
	var subscription models.RideSubscription
	if err := s.db.First(&subscription, subscriptionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("subscription not found")
		}
		return err
	}

	if subscription.UserID != userID {
		return fmt.Errorf("you can only delete your own subscriptions")
	}

	var subscribers []models.SubscriptionSubscriber
	s.db.Where("subscription_id = ? AND status = ?", subscriptionID, "active").Find(&subscribers)

	subscriberIDs := make([]uint, len(subscribers))
	for i, sub := range subscribers {
		subscriberIDs[i] = sub.SubscriberID
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subscription_id = ?", subscriptionID).Delete(&models.SubscriptionSubscriber{}).Error; err != nil {
			return err
		}

		if err := tx.Where("subscription_id = ?", subscriptionID).Delete(&models.SubscriptionNotification{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&subscription).Error; err != nil {
			return err
		}

		if s.notificationSvc != nil {
			for _, subscriberID := range subscriberIDs {
				if err := s.notificationSvc.CreateSubscriptionDeletedNotification(subscriberID, subscriptionID, userID); err != nil {
					log.Printf("Failed to create subscription deleted notification: %v", err)
				}
			}
		}

		return nil
	})
}

func (s *SubscriptionSvc) SubscribeToRide(userID, subscriptionID uint) error {
	var subscription models.RideSubscription
	if err := s.db.First(&subscription, subscriptionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("subscription not found")
		}
		return err
	}

	if subscription.UserID == userID {
		return fmt.Errorf("cannot subscribe to your own ride subscription")
	}

	if subscription.Status != "active" {
		return fmt.Errorf("subscription is not active")
	}

	var existing models.SubscriptionSubscriber
	err := s.db.Where("subscription_id = ? AND subscriber_id = ? AND status = ?",
		subscriptionID, userID, "active").First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil {
		return fmt.Errorf("you are already subscribed to this ride")
	}

	if subscription.MaxSubscribers > 0 {
		var currentCount int64
		s.db.Model(&models.SubscriptionSubscriber{}).
			Where("subscription_id = ? AND status = ?", subscriptionID, "active").
			Count(&currentCount)

		if int(currentCount) >= subscription.MaxSubscribers {
			return fmt.Errorf("subscription is full")
		}
	}

	subscriber := models.SubscriptionSubscriber{
		SubscriptionID: subscriptionID,
		SubscriberID:   userID,
		Status:         "active",
		NotifyEnabled:  true,
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&subscriber).Error; err != nil {
			return err
		}

		var user models.User
		subscriberName := "Unknown User"
		if err := tx.First(&user, userID).Error; err == nil {
			subscriberName = user.FullName
		}

		if s.notificationSvc != nil {
			if err := s.notificationSvc.CreateSubscriptionJoinNotification(subscription.UserID, subscriptionID, userID, subscriberName); err != nil {
				log.Printf("Failed to create subscription join notification: %v", err)
			}
		}

		return nil
	})
}

func (s *SubscriptionSvc) UnsubscribeFromRide(userID, subscriptionID uint) error {
	var subscription models.RideSubscription
	if err := s.db.First(&subscription, subscriptionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("subscription not found")
		}
		return err
	}

	var subscriber models.SubscriptionSubscriber
	err := s.db.Where("subscription_id = ? AND subscriber_id = ? AND status = ?",
		subscriptionID, userID, "active").First(&subscriber).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("you are not subscribed to this ride")
		}
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&subscriber).Error; err != nil {
			return err
		}

		var user models.User
		userName := "Unknown User"
		if err := tx.First(&user, userID).Error; err == nil {
			userName = user.FullName
		}

		if s.notificationSvc != nil {
			if err := s.notificationSvc.CreateSubscriptionLeaveNotification(subscription.UserID, subscriptionID, userID, userName); err != nil {
				log.Printf("Failed to create subscription leave notification: %v", err)
			}
		}

		return nil
	})
}

func (s *SubscriptionSvc) GetUserSubscriptions(userID uint) ([]models.SubscriptionResp, error) {
	var subscriptions []models.RideSubscription
	err := s.db.Preload("User").Where("user_id = ?", userID).Order("created_at DESC").Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}

	var responses []models.SubscriptionResp
	for _, sub := range subscriptions {
		var subscriberCount int64
		s.db.Model(&models.SubscriptionSubscriber{}).
			Where("subscription_id = ? AND status = ?", sub.ID, "active").
			Count(&subscriberCount)

		subscribers, _ := s.getSubscribers(sub.ID)
		if subscribers == nil {
			subscribers = []models.User{}
		}

		nextRides := s.getNextRideDates(sub, 3)

		availableSlots := sub.MaxSubscribers - int(subscriberCount)
		if availableSlots < 0 {
			availableSlots = 0
		}

		responses = append(responses, models.SubscriptionResp{
			Subscription:    sub,
			Driver:          sub.User,
			Subscribers:     subscribers,
			AvailableSlots:  availableSlots,
			SubscriberCount: int(subscriberCount),
			NextRideDates:   nextRides,
		})
	}

	return responses, nil
}

func (s *SubscriptionSvc) GetUserSubscribedRides(userID uint) ([]models.SubscriptionResp, error) {
	var subscriptions []models.RideSubscription
	err := s.db.Preload("User").
		Joins("JOIN subscription_subscribers ON ride_subscriptions.id = subscription_subscribers.subscription_id").
		Where("subscription_subscribers.subscriber_id = ? AND subscription_subscribers.status = ?", userID, "active").
		Order("ride_subscriptions.created_at DESC").
		Find(&subscriptions).Error

	if err != nil {
		return nil, err
	}

	var responses []models.SubscriptionResp
	for _, sub := range subscriptions {
		var subscriberCount int64
		s.db.Model(&models.SubscriptionSubscriber{}).
			Where("subscription_id = ? AND status = ?", sub.ID, "active").
			Count(&subscriberCount)

		nextRides := s.getNextRideDates(sub, 3)

		availableSlots := sub.MaxSubscribers - int(subscriberCount)
		if availableSlots < 0 {
			availableSlots = 0
		}

		responses = append(responses, models.SubscriptionResp{
			Subscription:    sub,
			Driver:          sub.User,
			AvailableSlots:  availableSlots,
			SubscriberCount: int(subscriberCount),
			NextRideDates:   nextRides,
			IsSubscribed:    true,
		})
	}

	return responses, nil
}

func (s *SubscriptionSvc) SearchSubscriptions(req models.SearchSubscriptionsReq, userID *uint) ([]models.SubscriptionResp, error) {
	query := s.db.Preload("User").Where("status = ?", "active")

	if req.From != "" {
		query = query.Where("UPPER(from_location) LIKE UPPER(?)", "%"+req.From+"%")
	}

	if req.To != "" {
		query = query.Where("UPPER(to_location) LIKE UPPER(?)", "%"+req.To+"%")
	}

	if req.DepartureTime != "" {
		if s.isValidTimeFormat(req.DepartureTime) {
			query = query.Where("departure_time = ?", req.DepartureTime)
		}
	}

	if req.WeekDay != "" {
		weekday := strings.ToLower(req.WeekDay)
		query = query.Where("LOWER(recurring_days) LIKE ?", "%"+weekday+"%")
	}

	var subscriptions []models.RideSubscription
	if err := query.Find(&subscriptions).Error; err != nil {
		return nil, err
	}

	return s.buildSubscriptionResponses(subscriptions, userID)
}

func (s *SubscriptionSvc) GetNearbySubscriptions(req models.NearbySubscriptionsReq, userID *uint) ([]models.SubscriptionResp, error) {
	if req.RadiusKM <= 0 {
		req.RadiusKM = 10
	}
	if req.MaxRouteDeviationKM <= 0 {
		req.MaxRouteDeviationKM = 3.0
	}

	query := s.db.Preload("User").Where("status = ?", "active")

	if req.DepartureTime != "" && s.isValidTimeFormat(req.DepartureTime) {
		query = query.Where("departure_time = ?", req.DepartureTime)
	}

	if req.WeekDay != "" {
		weekday := strings.ToLower(req.WeekDay)
		query = query.Where("LOWER(recurring_days) LIKE ?", "%"+weekday+"%")
	}

	var subscriptions []models.RideSubscription
	if err := query.Find(&subscriptions).Error; err != nil {
		return nil, err
	}

	type scoredSub struct {
		sub   models.RideSubscription
		score MatchScore
	}

	var nearbySubs []scoredSub
	for _, sub := range subscriptions {
		fromDistance := s.calculateDistance(req.FromLatitude, req.FromLongitude, sub.FromLatitude, sub.FromLongitude)
		toDistance := s.calculateDistance(req.ToLatitude, req.ToLongitude, sub.ToLatitude, sub.ToLongitude)

		if fromDistance <= req.RadiusKM && toDistance <= req.RadiusKM {
			ms := CalculateMatchScore(
				req.FromLatitude, req.FromLongitude, req.ToLatitude, req.ToLongitude,
				sub.FromLatitude, sub.FromLongitude, sub.ToLatitude, sub.ToLongitude,
			)

			if ms.RouteDeviation > req.MaxRouteDeviationKM {
				continue
			}

			nearbySubs = append(nearbySubs, scoredSub{sub: sub, score: ms})
		}
	}

	sort.Slice(nearbySubs, func(i, j int) bool {
		return nearbySubs[i].score.Score < nearbySubs[j].score.Score
	})

	// Build response with scoring fields populated.
	responses := make([]models.SubscriptionResp, 0, len(nearbySubs))
	for _, ns := range nearbySubs {
		sub := ns.sub
		var subscriberCount int64
		s.db.Model(&models.SubscriptionSubscriber{}).
			Where("subscription_id = ? AND status = ?", sub.ID, "active").
			Count(&subscriberCount)

		isSubscribed := false
		if userID != nil {
			var count int64
			s.db.Model(&models.SubscriptionSubscriber{}).
				Where("subscription_id = ? AND subscriber_id = ? AND status = ?",
					sub.ID, *userID, "active").
				Count(&count)
			isSubscribed = count > 0
		}

		nextRides := s.getNextRideDates(sub, 3)

		availableSlots := sub.MaxSubscribers - int(subscriberCount)
		if availableSlots < 0 {
			availableSlots = 0
		}

		responses = append(responses, models.SubscriptionResp{
			Subscription:      sub,
			Driver:            sub.User,
			AvailableSlots:    availableSlots,
			SubscriberCount:   int(subscriberCount),
			NextRideDates:     nextRides,
			IsSubscribed:      isSubscribed,
			MatchScore:        ns.score.Score,
			PickupDistanceKm:  ns.score.PickupDistance,
			DropoffDistanceKm: ns.score.DropoffDistance,
		})
	}

	return responses, nil
}

func (s *SubscriptionSvc) GetAllSubscriptions(userID *uint) ([]models.SubscriptionResp, error) {
	var subscriptions []models.RideSubscription
	err := s.db.Preload("User").Where("status = ?", "active").Order("created_at DESC").Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}

	return s.buildSubscriptionResponses(subscriptions, userID)
}

func (s *SubscriptionSvc) SendDailyNotifications() error {
	today := time.Now().UTC()
	weekday := strings.ToLower(today.Weekday().String())

	var subscriptions []models.RideSubscription
	err := s.db.Where("status = ? AND LOWER(recurring_days) LIKE ?", "active", "%"+weekday+"%").Find(&subscriptions).Error
	if err != nil {
		return err
	}

	for _, sub := range subscriptions {
		if !s.isDateInSubscriptionPeriod(today, sub) {
			continue
		}

		departureDateTime, err := s.getDepartureDateTimeForToday(sub, today)
		if err != nil {
			log.Printf("Failed to get departure time for subscription %d: %v", sub.ID, err)
			continue
		}

		notificationTime := departureDateTime.Add(-time.Duration(sub.NotificationTime) * time.Minute)

		now := time.Now().UTC()
		if now.Before(notificationTime) || now.After(departureDateTime) {
			continue
		}

		var subscribers []models.SubscriptionSubscriber
		s.db.Where("subscription_id = ? AND status = ? AND notify_enabled = ?",
			sub.ID, "active", true).Find(&subscribers)

		if s.notificationSvc != nil {
			for _, subscriber := range subscribers {
				// Check if notification already sent today
				var existingNotif models.SubscriptionNotification
				err := s.db.Where("subscription_id = ? AND subscriber_id = ? AND DATE(ride_date) = DATE(?) AND notification_sent = ?",
					sub.ID, subscriber.SubscriberID, today, true).First(&existingNotif).Error

				if err == gorm.ErrRecordNotFound {
					notif := models.SubscriptionNotification{
						SubscriptionID:   sub.ID,
						SubscriberID:     subscriber.SubscriberID,
						RideDate:         today,
						NotificationSent: true,
					}
					s.db.Create(&notif)

					if err := s.notificationSvc.CreateSubscriptionRideNotification(
						subscriber.SubscriberID, sub.ID, sub.UserID, departureDateTime); err != nil {
						log.Printf("Failed to send ride notification: %v", err)
					}
				}
			}
		}
	}

	return nil
}

func (s *SubscriptionSvc) isValidTimeFormat(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

func (s *SubscriptionSvc) validateRecurringDays(days []string) error {
	validDays := map[string]bool{
		"monday":    true,
		"tuesday":   true,
		"wednesday": true,
		"thursday":  true,
		"friday":    true,
		"saturday":  true,
		"sunday":    true,
	}

	for _, day := range days {
		if !validDays[strings.ToLower(day)] {
			return fmt.Errorf("invalid day: %s", day)
		}
	}

	return nil
}

func (s *SubscriptionSvc) getCoordinates(address string) (float64, float64, error) {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		return 0, 0, fmt.Errorf("google Maps API key not configured")
	}

	baseURL := "https://maps.googleapis.com/maps/api/geocode/json"
	params := url.Values{}
	params.Add("address", address)
	params.Add("key", apiKey)
	params.Add("region", "in")

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Results []struct {
			Geometry struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, err
	}

	if result.Status != "OK" || len(result.Results) == 0 {
		return 0, 0, fmt.Errorf("location not found: %s", address)
	}

	location := result.Results[0].Geometry.Location
	return location.Lat, location.Lng, nil
}

func (s *SubscriptionSvc) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	return haversineDistance(lat1, lon1, lat2, lon2)
}

func (s *SubscriptionSvc) getSubscribers(subscriptionID uint) ([]models.User, error) {
	var users []models.User
	err := s.db.Joins("JOIN subscription_subscribers ON users.id = subscription_subscribers.subscriber_id").
		Where("subscription_subscribers.subscription_id = ? AND subscription_subscribers.status = ?",
			subscriptionID, "active").
		Find(&users).Error

	return users, err
}

func (s *SubscriptionSvc) getNextRideDates(subscription models.RideSubscription, limit int) []time.Time {
	var dates []time.Time
	recurringDays := []string(subscription.RecurringDays)

	dayMap := make(map[time.Weekday]bool)
	for _, day := range recurringDays {
		switch strings.ToLower(day) {
		case "sunday":
			dayMap[time.Sunday] = true
		case "monday":
			dayMap[time.Monday] = true
		case "tuesday":
			dayMap[time.Tuesday] = true
		case "wednesday":
			dayMap[time.Wednesday] = true
		case "thursday":
			dayMap[time.Thursday] = true
		case "friday":
			dayMap[time.Friday] = true
		case "saturday":
			dayMap[time.Saturday] = true
		}
	}

	currentDate := time.Now().UTC().Truncate(24 * time.Hour)
	if currentDate.Before(subscription.StartDate) {
		currentDate = subscription.StartDate
	}

	for len(dates) < limit {
		if subscription.EndDate != nil && currentDate.After(*subscription.EndDate) {
			break
		}

		if dayMap[currentDate.Weekday()] {
			dates = append(dates, currentDate)
		}

		currentDate = currentDate.Add(24 * time.Hour)
	}

	return dates
}

func (s *SubscriptionSvc) isDateInSubscriptionPeriod(date time.Time, subscription models.RideSubscription) bool {
	if date.Before(subscription.StartDate) {
		return false
	}

	if subscription.EndDate != nil && date.After(*subscription.EndDate) {
		return false
	}

	return true
}

func (s *SubscriptionSvc) getDepartureDateTimeForToday(subscription models.RideSubscription, today time.Time) (time.Time, error) {
	departureTime, err := time.Parse("15:04", subscription.DepartureTime)
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(
		today.Year(), today.Month(), today.Day(),
		departureTime.Hour(), departureTime.Minute(), 0, 0,
		today.Location(),
	), nil
}

func (s *SubscriptionSvc) buildSubscriptionResponses(subscriptions []models.RideSubscription, userID *uint) ([]models.SubscriptionResp, error) {
	responses := make([]models.SubscriptionResp, 0)

	for _, sub := range subscriptions {
		var subscriberCount int64
		s.db.Model(&models.SubscriptionSubscriber{}).
			Where("subscription_id = ? AND status = ?", sub.ID, "active").
			Count(&subscriberCount)

		isSubscribed := false
		if userID != nil {
			var count int64
			s.db.Model(&models.SubscriptionSubscriber{}).
				Where("subscription_id = ? AND subscriber_id = ? AND status = ?",
					sub.ID, *userID, "active").
				Count(&count)
			isSubscribed = count > 0
		}

		nextRides := s.getNextRideDates(sub, 3)

		availableSlots := sub.MaxSubscribers - int(subscriberCount)
		if availableSlots < 0 {
			availableSlots = 0
		}

		responses = append(responses, models.SubscriptionResp{
			Subscription:    sub,
			Driver:          sub.User,
			AvailableSlots:  availableSlots,
			SubscriberCount: int(subscriberCount),
			NextRideDates:   nextRides,
			IsSubscribed:    isSubscribed,
		})
	}

	return responses, nil
}

func (s *SubscriptionSvc) GetSubscriptionStats() (*models.SubscriptionStatsResp, error) {
	var totalSubs, activeSubs, totalSubscribers int64
	var todaysRides int64

	s.db.Model(&models.RideSubscription{}).Count(&totalSubs)

	s.db.Model(&models.RideSubscription{}).Where("status = ?", "active").Count(&activeSubs)

	s.db.Model(&models.SubscriptionSubscriber{}).Where("status = ?", "active").Count(&totalSubscribers)

	today := strings.ToLower(time.Now().Weekday().String())
	s.db.Model(&models.RideSubscription{}).
		Where("status = ? AND LOWER(recurring_days) LIKE ?", "active", "%"+today+"%").
		Count(&todaysRides)

	return &models.SubscriptionStatsResp{
		TotalSubscriptions:  int(totalSubs),
		ActiveSubscriptions: int(activeSubs),
		TotalSubscribers:    int(totalSubscribers),
		TodaysRides:         int(todaysRides),
	}, nil
}

func (s *SubscriptionSvc) ToggleNotifications(userID, subscriptionID uint, enabled bool) error {
	var subscriber models.SubscriptionSubscriber
	err := s.db.Where("subscription_id = ? AND subscriber_id = ? AND status = ?",
		subscriptionID, userID, "active").First(&subscriber).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("you are not subscribed to this ride")
		}
		return err
	}

	return s.db.Model(&subscriber).Update("notify_enabled", enabled).Error
}
