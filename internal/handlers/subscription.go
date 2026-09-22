package handlers

import (
	"net/http"
	"strconv"

	"mride-backend/internal/models"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type SubscriptionService interface {
	CreateSubscription(userID uint, req models.CreateSubscriptionReq) (*models.SubscriptionResp, error)
	GetSubscription(subscriptionID uint, userID *uint) (*models.SubscriptionResp, error)
	UpdateSubscription(userID, subscriptionID uint, req models.UpdateSubscriptionReq) error
	DeleteSubscription(userID, subscriptionID uint) error
	SubscribeToRide(userID, subscriptionID uint) error
	UnsubscribeFromRide(userID, subscriptionID uint) error
	GetUserSubscriptions(userID uint) ([]models.SubscriptionResp, error)
	GetUserSubscribedRides(userID uint) ([]models.SubscriptionResp, error)
	SearchSubscriptions(req models.SearchSubscriptionsReq, userID *uint) ([]models.SubscriptionResp, error)
	GetNearbySubscriptions(req models.NearbySubscriptionsReq, userID *uint) ([]models.SubscriptionResp, error)
	GetAllSubscriptions(userID *uint) ([]models.SubscriptionResp, error)
	GetSubscriptionStats() (*models.SubscriptionStatsResp, error)
	ToggleNotifications(userID, subscriptionID uint, enabled bool) error
	SendDailyNotifications() error
}

type SubscriptionHandler struct {
	subscriptionSvc SubscriptionService
}

func NewSubscriptionHandler(subscriptionSvc SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionSvc: subscriptionSvc,
	}
}

func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var req models.CreateSubscriptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	if req.FromLocation == req.ToLocation {
		utils.ErrJSON(c, http.StatusBadRequest, "From and to locations cannot be the same")
		return
	}

	resp, err := h.subscriptionSvc.CreateSubscription(uint(userID.(int)), req)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Subscription created successfully", resp)
}

func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	var userID *uint
	if userIDVal, exists := c.Get("user_id"); exists {
		uid := uint(userIDVal.(int))
		userID = &uid
	}

	resp, err := h.subscriptionSvc.GetSubscription(uint(subscriptionID), userID)
	if err != nil {
		utils.ErrJSON(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccJSON(c, "Subscription retrieved successfully", resp)
}

func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	var req models.UpdateSubscriptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err = h.subscriptionSvc.UpdateSubscription(uint(userID.(int)), uint(subscriptionID), req)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Subscription updated successfully", nil)
}

func (h *SubscriptionHandler) DeleteSubscription(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err = h.subscriptionSvc.DeleteSubscription(uint(userID.(int)), uint(subscriptionID))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Subscription deleted successfully", nil)
}

func (h *SubscriptionHandler) SubscribeToRide(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err = h.subscriptionSvc.SubscribeToRide(uint(userID.(int)), uint(subscriptionID))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Successfully subscribed to the ride", nil)
}

func (h *SubscriptionHandler) UnsubscribeFromRide(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err = h.subscriptionSvc.UnsubscribeFromRide(uint(userID.(int)), uint(subscriptionID))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Successfully unsubscribed from the ride", nil)
}

func (h *SubscriptionHandler) GetMySubscriptions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	resp, err := h.subscriptionSvc.GetUserSubscriptions(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch user subscriptions")
		return
	}

	if resp == nil {
		resp = []models.SubscriptionResp{}
	}

	utils.SuccJSON(c, "User subscriptions retrieved successfully", resp)
}

func (h *SubscriptionHandler) GetMySubscribedRides(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	resp, err := h.subscriptionSvc.GetUserSubscribedRides(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch subscribed rides")
		return
	}

	if resp == nil {
		resp = []models.SubscriptionResp{}
	}

	utils.SuccJSON(c, "Subscribed rides retrieved successfully", resp)
}

func (h *SubscriptionHandler) SearchSubscriptions(c *gin.Context) {
	req := models.SearchSubscriptionsReq{
		From:          c.Query("from"),
		To:            c.Query("to"),
		DepartureTime: c.Query("departure_time"),
		WeekDay:       c.Query("weekday"),
	}

	if req.From == "" && req.To == "" && req.DepartureTime == "" && req.WeekDay == "" {
		utils.ErrJSON(c, http.StatusBadRequest, "At least one search parameter required")
		return
	}

	var userID *uint
	if userIDVal, exists := c.Get("user_id"); exists {
		uid := uint(userIDVal.(int))
		userID = &uid
	}

	subscriptions, err := h.subscriptionSvc.SearchSubscriptions(req, userID)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to search subscriptions")
		return
	}

	if subscriptions == nil {
		subscriptions = []models.SubscriptionResp{}
	}

	utils.SuccJSON(c, "Subscriptions retrieved successfully", subscriptions)
}

func (h *SubscriptionHandler) GetNearbySubscriptions(c *gin.Context) {
	fromLat, _ := strconv.ParseFloat(c.Query("from_lat"), 64)
	fromLng, _ := strconv.ParseFloat(c.Query("from_lng"), 64)
	toLat, _ := strconv.ParseFloat(c.Query("to_lat"), 64)
	toLng, _ := strconv.ParseFloat(c.Query("to_lng"), 64)
	radius, _ := strconv.ParseFloat(c.Query("radius"), 64)

	if radius <= 0 {
		radius = 10
	}

	if fromLat < -90 || fromLat > 90 || fromLng < -180 || fromLng > 180 ||
		toLat < -90 || toLat > 90 || toLng < -180 || toLng > 180 {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid coordinates")
		return
	}

	maxDeviation, _ := strconv.ParseFloat(c.Query("max_route_deviation_km"), 64)

	req := models.NearbySubscriptionsReq{
		FromLatitude:        fromLat,
		FromLongitude:       fromLng,
		ToLatitude:          toLat,
		ToLongitude:         toLng,
		RadiusKM:            radius,
		MaxRouteDeviationKM: maxDeviation,
		DepartureTime:       c.Query("departure_time"),
		WeekDay:             c.Query("weekday"),
	}

	var userID *uint
	if userIDVal, exists := c.Get("user_id"); exists {
		uid := uint(userIDVal.(int))
		userID = &uid
	}

	subscriptions, err := h.subscriptionSvc.GetNearbySubscriptions(req, userID)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch nearby subscriptions: "+err.Error())
		return
	}

	if subscriptions == nil {
		subscriptions = []models.SubscriptionResp{}
	}

	utils.SuccJSON(c, "Nearby subscriptions retrieved successfully", subscriptions)
}

func (h *SubscriptionHandler) GetAllSubscriptions(c *gin.Context) {
	var userID *uint
	if userIDVal, exists := c.Get("user_id"); exists {
		uid := uint(userIDVal.(int))
		userID = &uid
	}

	subscriptions, err := h.subscriptionSvc.GetAllSubscriptions(userID)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch subscriptions")
		return
	}

	if subscriptions == nil {
		subscriptions = []models.SubscriptionResp{}
	}

	utils.SuccJSON(c, "All subscriptions retrieved successfully", subscriptions)
}

func (h *SubscriptionHandler) GetSubscriptionStats(c *gin.Context) {
	stats, err := h.subscriptionSvc.GetSubscriptionStats()
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch subscription stats")
		return
	}

	utils.SuccJSON(c, "Subscription stats retrieved successfully", stats)
}

func (h *SubscriptionHandler) ToggleNotifications(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid subscription ID")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err = h.subscriptionSvc.ToggleNotifications(uint(userID.(int)), uint(subscriptionID), req.Enabled)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	status := "disabled"
	if req.Enabled {
		status = "enabled"
	}

	utils.SuccJSON(c, "Notifications "+status+" successfully", nil)
}

func (h *SubscriptionHandler) SendDailyNotifications(c *gin.Context) {
	err := h.subscriptionSvc.SendDailyNotifications()
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to send daily notifications: "+err.Error())
		return
	}

	utils.SuccJSON(c, "Daily notifications sent successfully", nil)
}
