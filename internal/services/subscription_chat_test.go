package services

import (
	"testing"
	"time"

	"mride-backend/internal/models"
	"mride-backend/internal/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubscriptionChatSvc_SendChatMessage_Success(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.User{}, &models.RideSubscription{}, &models.SubscriptionSubscriber{}, &models.SubscriptionChat{}, &models.SubscriptionChatReadStatus{})
	mockWS := new(MockWebSocketSvc)
	mockNotif := new(MockNotificationSvc)

	svc := NewSubscriptionChatSvc(db, mockWS, mockNotif)

	driver := models.User{ID: 1, FullName: "Driver", Email: "driver@test.com"}
	subscriber := models.User{ID: 2, FullName: "Subscriber", Email: "sub@test.com"}
	db.Create(&driver)
	db.Create(&subscriber)

	subscription := models.RideSubscription{
		ID:               1,
		UserID:           driver.ID,
		Title:            "Test Ride",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    models.StringArray{"Monday"},
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "active",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	subscriptionSub := models.SubscriptionSubscriber{
		SubscriptionID: subscription.ID,
		SubscriberID:   subscriber.ID,
		Status:         "active",
		NotifyEnabled:  true,
	}
	db.Create(&subscriptionSub)

	mockWS.On("SendToUser", int(driver.ID), mock.AnythingOfType("WSMessage")).Return(nil)
	mockNotif.On("CreateChatNotification", driver.ID, subscription.ID, subscriber.ID, subscriber.FullName).Return(nil)

	resp, err := svc.SendChatMessage(subscriber.ID, subscription.ID, "Hello driver!")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Hello driver!", resp.Message)
	assert.Equal(t, subscriber.ID, resp.SenderID)
	assert.True(t, resp.IsMine)

	mockWS.AssertExpectations(t)
	mockNotif.AssertExpectations(t)
}

func TestSubscriptionChatSvc_SendChatMessage_NotPartOfSubscription(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.User{}, &models.RideSubscription{}, &models.SubscriptionSubscriber{}, &models.SubscriptionChat{})
	svc := NewSubscriptionChatSvc(db, nil, nil)

	driver := models.User{ID: 1, FullName: "Driver", Email: "driver@test.com"}
	db.Create(&driver)

	subscription := models.RideSubscription{
		ID:               1,
		UserID:           driver.ID,
		Title:            "Test Ride",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    models.StringArray{"Monday"},
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "active",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	_, err := svc.SendChatMessage(999, subscription.ID, "Hello!")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not part of this subscription")
}

func TestSubscriptionChatSvc_SendChatMessage_InactiveSubscription(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.User{}, &models.RideSubscription{}, &models.SubscriptionSubscriber{}, &models.SubscriptionChat{})
	svc := NewSubscriptionChatSvc(db, nil, nil)

	driver := models.User{ID: 1, FullName: "Driver", Email: "driver@test.com"}
	db.Create(&driver)

	subscription := models.RideSubscription{
		ID:               1,
		UserID:           driver.ID,
		Title:            "Test Ride",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    models.StringArray{"Monday"},
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "inactive",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	_, err := svc.SendChatMessage(driver.ID, subscription.ID, "Hello!")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chat is only available for active subscriptions")
}

func TestSubscriptionChatSvc_GetChatHistory_Success(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.User{}, &models.RideSubscription{}, &models.SubscriptionSubscriber{}, &models.SubscriptionChat{}, &models.SubscriptionChatReadStatus{})
	svc := NewSubscriptionChatSvc(db, nil, nil)

	driver := models.User{ID: 1, FullName: "Driver", Email: "driver@test.com"}
	subscriber := models.User{ID: 2, FullName: "Subscriber", Email: "sub@test.com"}
	db.Create(&driver)
	db.Create(&subscriber)

	subscription := models.RideSubscription{
		ID:               1,
		UserID:           driver.ID,
		Title:            "Test Ride",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    models.StringArray{"Monday"},
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "active",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	subscriptionSub := models.SubscriptionSubscriber{
		SubscriptionID: subscription.ID,
		SubscriberID:   subscriber.ID,
		Status:         "active",
		NotifyEnabled:  true,
	}
	db.Create(&subscriptionSub)

	msg1 := models.SubscriptionChat{SubscriptionID: subscription.ID, SenderID: driver.ID, Message: "Message 1"}
	msg2 := models.SubscriptionChat{SubscriptionID: subscription.ID, SenderID: subscriber.ID, Message: "Message 2"}
	db.Create(&msg1)
	db.Create(&msg2)

	resp, err := svc.GetChatHistory(driver.ID, subscription.ID, 50, 0)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 2, resp.Count)
	assert.Len(t, resp.Messages, 2)
}

func TestSubscriptionChatSvc_MarkChatAsRead_Success(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.User{}, &models.RideSubscription{}, &models.SubscriptionSubscriber{}, &models.SubscriptionChat{}, &models.SubscriptionChatReadStatus{})
	svc := NewSubscriptionChatSvc(db, nil, nil)

	driver := models.User{ID: 1, FullName: "Driver", Email: "driver@test.com"}
	db.Create(&driver)

	subscription := models.RideSubscription{
		ID:               1,
		UserID:           driver.ID,
		Title:            "Test Ride",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    models.StringArray{"Monday"},
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "active",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	msg := models.SubscriptionChat{SubscriptionID: subscription.ID, SenderID: driver.ID, Message: "Test"}
	db.Create(&msg)

	err := svc.MarkChatAsRead(driver.ID, subscription.ID)

	assert.NoError(t, err)

	var readStatus models.SubscriptionChatReadStatus
	db.Where("subscription_id = ? AND user_id = ?", subscription.ID, driver.ID).First(&readStatus)
	assert.Equal(t, msg.ID, readStatus.LastReadMessageID)
}

func TestSubscriptionChatSvc_GetSubscriptionChats_Success(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.User{}, &models.RideSubscription{}, &models.SubscriptionSubscriber{}, &models.SubscriptionChat{})
	svc := NewSubscriptionChatSvc(db, nil, nil)

	driver := models.User{ID: 1, FullName: "Driver", Email: "driver@test.com"}
	subscriber := models.User{ID: 2, FullName: "Subscriber", Email: "sub@test.com"}
	db.Create(&driver)
	db.Create(&subscriber)

	subscription := models.RideSubscription{
		ID:               1,
		UserID:           driver.ID,
		Title:            "Test Ride",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    models.StringArray{"Monday"},
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "active",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	subscriptionSub := models.SubscriptionSubscriber{
		SubscriptionID: subscription.ID,
		SubscriberID:   subscriber.ID,
		Status:         "active",
		NotifyEnabled:  true,
	}
	db.Create(&subscriptionSub)

	chatRooms, err := svc.GetSubscriptionChats(driver.ID)

	assert.NoError(t, err)
	assert.Len(t, chatRooms, 1)
	assert.Equal(t, subscription.ID, chatRooms[0].SubscriptionID)
}

func TestSubscriptionChatSvc_CanSendMessage_Success(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.User{}, &models.RideSubscription{}, &models.SubscriptionSubscriber{})
	svc := NewSubscriptionChatSvc(db, nil, nil)

	driver := models.User{ID: 1, FullName: "Driver", Email: "driver@test.com"}
	db.Create(&driver)

	subscription := models.RideSubscription{
		ID:               1,
		UserID:           driver.ID,
		Title:            "Test Ride",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    models.StringArray{"Monday"},
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "active",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	canSend, msg := svc.CanSendMessage(driver.ID, subscription.ID)

	assert.True(t, canSend)
	assert.Empty(t, msg)
}

func TestSubscriptionChatSvc_CanSendMessage_NotActive(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.User{}, &models.RideSubscription{})
	svc := NewSubscriptionChatSvc(db, nil, nil)

	driver := models.User{ID: 1, FullName: "Driver", Email: "driver@test.com"}
	db.Create(&driver)

	subscription := models.RideSubscription{
		ID:               1,
		UserID:           driver.ID,
		Title:            "Test Ride",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    models.StringArray{"Monday"},
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "inactive",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	canSend, msg := svc.CanSendMessage(driver.ID, subscription.ID)

	assert.False(t, canSend)
	assert.Equal(t, "Subscription is not active", msg)
}
