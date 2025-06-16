package handlers

import (
	"net/http"
	"strconv"

	"mride-backend/internal/models"
	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type RideHandler struct {
	rideSvc *services.RideSvc
}

func NewRideHandler(rideSvc *services.RideSvc) *RideHandler {
	return &RideHandler{
		rideSvc: rideSvc,
	}
}

func (h *RideHandler) CreateRide(c *gin.Context) {
	var req models.CreateRideReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	if req.CarNumber == "" {
		utils.ErrJSON(c, http.StatusBadRequest, "Car number is required")
		return
	}

	if req.CarModel == "" {
		utils.ErrJSON(c, http.StatusBadRequest, "Car model is required")
		return
	}

	if req.PassengerCount < 1 || req.PassengerCount > 8 {
		utils.ErrJSON(c, http.StatusBadRequest, "Passenger count must be between 1 and 8")
		return
	}

	if req.FromLocation == req.ToLocation {
		utils.ErrJSON(c, http.StatusBadRequest, "From and to locations cannot be the same")
		return
	}

	resp, err := h.rideSvc.CreateRide(userID.(int), req)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Ride created successfully", resp)
}

func (h *RideHandler) GetMyRides(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	resp, err := h.rideSvc.GetAllUserRides(userID.(int))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch user rides")
		return
	}

	utils.SuccJSON(c, "User rides retrieved successfully", resp)
}

func (h *RideHandler) GetRide(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, err := strconv.Atoi(rideIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid ride ID")
		return
	}

	resp, err := h.rideSvc.GetRideByID(rideID)
	if err != nil {
		utils.ErrJSON(c, http.StatusNotFound, "Ride not found")
		return
	}

	utils.SuccJSON(c, "Ride retrieved successfully", resp)
}

func (h *RideHandler) SearchRides(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")

	if from == "" && to == "" {
		utils.ErrJSON(c, http.StatusBadRequest, "At least one search parameter required")
		return
	}

	rides, err := h.rideSvc.SearchRides(from, to)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to search rides")
		return
	}

	utils.SuccJSON(c, "Rides retrieved successfully", rides)
}

func (h *RideHandler) GetNearbyRides(c *gin.Context) {
	fromLat, _ := strconv.ParseFloat(c.Query("from_lat"), 64)
	fromLng, _ := strconv.ParseFloat(c.Query("from_lng"), 64)
	toLat, _ := strconv.ParseFloat(c.Query("to_lat"), 64)
	toLng, _ := strconv.ParseFloat(c.Query("to_lng"), 64)
	radius, _ := strconv.ParseFloat(c.Query("radius"), 64)

	if radius <= 0 {
		radius = 10
	}

	userID := 0
	if userIDVal, exists := c.Get("user_id"); exists {
		userID = userIDVal.(int)
	}

	if fromLat < -90 || fromLat > 90 || fromLng < -180 || fromLng > 180 ||
		toLat < -90 || toLat > 90 || toLng < -180 || toLng > 180 {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid coordinates")
		return
	}

	req := models.NearbyRidesReq{
		FromLatitude:  fromLat,
		FromLongitude: fromLng,
		ToLatitude:    toLat,
		ToLongitude:   toLng,
		RadiusKM:      radius,
	}

	rides, err := h.rideSvc.GetNearbyRides(userID, req)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch nearby rides: "+err.Error())
		return
	}

	utils.SuccJSON(c, "Nearby rides retrieved successfully", rides)
}

func (h *RideHandler) JoinRide(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, err := strconv.Atoi(rideIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid ride ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err = h.rideSvc.JoinRide(userID.(int), rideID)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Successfully joined the ride", nil)
}

func (h *RideHandler) GetAllRides(c *gin.Context) {
	rides, err := h.rideSvc.GetAllRides()
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch rides")
		return
	}

	if rides == nil {
		rides = []models.RideResp{}
	}

	utils.SuccJSON(c, "All rides retrieved successfully", rides)
}
