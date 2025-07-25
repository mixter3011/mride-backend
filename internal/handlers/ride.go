package handlers

import (
	"net/http"
	"strconv"

	"mride-backend/internal/models"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type RideService interface {
	CreateRide(userID uint, req models.CreateRideReq) (*models.RideResp, error)
	DeleteRide(userID, rideID uint) error
	GetAllUserRides(userID uint) ([]models.RideResp, error)
	GetRideByID(rideID uint) (*models.RideResp, error)
	SearchRides(from, to string) ([]models.RideResp, error)
	GetNearbyRides(userID uint, req models.NearbyRidesReq) ([]models.RideResp, error)
	JoinRide(userID, rideID uint) error
	LeaveRide(userID, rideID uint) error
	GetAllRides() ([]models.RideResp, error)
	StartRide(userID, rideID uint, req models.StartRideReq) error
	CompleteRide(userID, rideID uint) error
	GetRideProgress(rideID uint) (*models.RideProgressResp, error)
	GetActiveRidesForUser(userID uint) ([]models.RideProgressResp, error)
}
type RideHandler struct {
	rideSvc RideService
}

func NewRideHandler(rideSvc RideService) *RideHandler {
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

	if req.FromLocation == req.ToLocation {
		utils.ErrJSON(c, http.StatusBadRequest, "From and to locations cannot be the same")
		return
	}

	resp, err := h.rideSvc.CreateRide(uint(userID.(int)), req)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Ride created successfully", resp)
}

func (h *RideHandler) DeleteRide(c *gin.Context) {
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

	err = h.rideSvc.DeleteRide(uint(userID.(int)), uint(rideID))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Ride deleted successfully", nil)
}

func (h *RideHandler) GetMyRides(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	resp, err := h.rideSvc.GetAllUserRides(uint(userID.(int)))
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

	resp, err := h.rideSvc.GetRideByID(uint(rideID))
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

	userID := uint(0)
	if userIDVal, exists := c.Get("user_id"); exists {
		userID = uint(userIDVal.(int))
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

	err = h.rideSvc.JoinRide(uint(userID.(int)), uint(rideID))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Successfully joined the ride", nil)
}

func (h *RideHandler) LeaveRide(c *gin.Context) {
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

	err = h.rideSvc.LeaveRide(uint(userID.(int)), uint(rideID))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Successfully left the ride", nil)
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

func (h *RideHandler) StartRide(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, err := strconv.Atoi(rideIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid ride ID")
		return
	}

	var req models.StartRideReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err = h.rideSvc.StartRide(uint(userID.(int)), uint(rideID), req)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Ride started successfully", nil)
}

func (h *RideHandler) CompleteRide(c *gin.Context) {
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

	err = h.rideSvc.CompleteRide(uint(userID.(int)), uint(rideID))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Ride completed successfully", nil)
}

func (h *RideHandler) GetRideProgress(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, err := strconv.Atoi(rideIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid ride ID")
		return
	}

	progress, err := h.rideSvc.GetRideProgress(uint(rideID))
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccJSON(c, "Ride progress retrieved successfully", progress)
}

func (h *RideHandler) GetActiveRides(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	rides, err := h.rideSvc.GetActiveRidesForUser(uint(userID.(int)))
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to fetch active rides")
		return
	}

	if rides == nil {
		rides = []models.RideProgressResp{}
	}

	utils.SuccJSON(c, "Active rides retrieved successfully", rides)
}
