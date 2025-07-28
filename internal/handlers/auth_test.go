package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mride-backend/internal/handlers"
	"mride-backend/internal/models"
	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRouterAndHandler(t *testing.T) (*gin.Engine, *services.AuthSvc) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&models.User{}, &models.PendingPhoneUpdate{})
	assert.NoError(t, err)

	jwtSvc := services.NewJWTSvc("test-secret", db)
	authSvc := services.NewAuthSvc(db, jwtSvc)

	authHandler := handlers.NewAuthHandler(authSvc, nil)

	r := gin.Default()
	r.POST("/auth/signup", authHandler.SignUp)
	r.POST("/auth/signin", authHandler.SignIn)

	protected := r.Group("/auth")
	protected.Use(func(c *gin.Context) {

		c.Set("user_id", 1)
		c.Next()
	})
	protected.POST("/logout", authHandler.Logout)

	return r, authSvc
}

func TestSignUpAndSignInEndpoints(t *testing.T) {
	router, _ := setupRouterAndHandler(t)

	signupBody := models.SignUpReq{
		FullName:        "Test User",
		Email:           "test@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	jsonBody, _ := json.Marshal(signupBody)
	req, _ := http.NewRequest("POST", "/auth/signup", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	signinBody := models.SignInReq{
		Email:    "test@example.com",
		Password: "password123",
		Phone:    "9999999999",
	}
	jsonBody, _ = json.Marshal(signinBody)
	req, _ = http.NewRequest("POST", "/auth/signin", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	signinBody.Password = "wrong"
	jsonBody, _ = json.Marshal(signinBody)
	req, _ = http.NewRequest("POST", "/auth/signin", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestLogoutEndpoint(t *testing.T) {
	router, authSvc := setupRouterAndHandler(t)

	if _, err := authSvc.SignUp(models.SignUpReq{
		FullName:        "Logout User",
		Email:           "logout@example.com",
		Password:        "logout123",
		ConfirmPassword: "logout123",
	}); err != nil {
		t.Errorf("Signup failed %v", err)
	}

	req, _ := http.NewRequest("POST", "/auth/logout", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestProfileEndpoints(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Errorf("AutoMigrate failed: %v", err)
	}

	user := models.User{
		FullName:     "John Profile",
		Email:        "profile@example.com",
		PasswordHash: "hashed",
	}
	db.Create(&user)

	jwtSvc := services.NewJWTSvc("secret", db)
	authSvc := services.NewAuthSvc(db, jwtSvc)
	authHandler := handlers.NewAuthHandler(authSvc, nil)

	r := gin.Default()

	r.GET("/auth/profile", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		authHandler.GetProfile(c)
	})

	r.PUT("/auth/profile", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		authHandler.UpdateProfile(c)
	})

	req, _ := http.NewRequest("GET", "/auth/profile", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	update := models.UpdateProfileReq{
		FullName: "Updated Name",
		Email:    "newprofile@example.com",
	}
	body, _ := json.Marshal(update)
	req, _ = http.NewRequest("PUT", "/auth/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestUpdatePasswordHandler(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Errorf("AutoMigrate failed: %v", err)
	}

	hashed, _ := utils.HashPwd("old123")
	user := models.User{
		FullName:     "Test Pwd",
		Email:        "pwd@example.com",
		PasswordHash: hashed,
	}
	db.Create(&user)

	jwtSvc := services.NewJWTSvc("secret", db)
	authSvc := services.NewAuthSvc(db, jwtSvc)
	authHandler := handlers.NewAuthHandler(authSvc, nil)

	r := gin.Default()
	r.PUT("/auth/password", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		authHandler.UpdatePassword(c)
	})

	body := models.UpdatePasswordReq{
		CurrentPassword: "old123",
		NewPassword:     "new12345",
		ConfirmPassword: "new12345",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/auth/password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestRefreshTokenHandler(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Errorf("AutoMigrate failed: %v", err)
	}

	jwtSvc := services.NewJWTSvc("secret", db)
	authSvc := services.NewAuthSvc(db, jwtSvc)
	authHandler := handlers.NewAuthHandler(authSvc, nil)

	user := models.User{
		FullName:     "Refresher",
		Email:        "refresh@example.com",
		PasswordHash: "hash",
	}
	db.Create(&user)

	refreshToken, _ := jwtSvc.GenRefreshToken(int(user.ID))
	user.RefreshToken = refreshToken
	user.RefreshTokenExpiry = time.Now().Add(5 * time.Minute)
	db.Save(&user)

	r := gin.Default()
	r.POST("/auth/refresh", authHandler.RefreshToken)

	body := map[string]string{"refresh_token": refreshToken}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, 200, resp.Code)

	body["refresh_token"] = "invalid.token"
	jsonBody, _ = json.Marshal(body)
	req, _ = http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, 401, resp.Code)

	expiredToken, _ := jwtSvc.GenRefreshToken(int(user.ID))
	user.RefreshToken = expiredToken
	user.RefreshTokenExpiry = time.Now().Add(-5 * time.Minute)
	db.Save(&user)

	body["refresh_token"] = expiredToken
	jsonBody, _ = json.Marshal(body)
	req, _ = http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, 401, resp.Code)
}

func TestPhoneUpdateFlow(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err := db.AutoMigrate(&models.User{}, &models.OTP{}, &models.PendingPhoneUpdate{}); err != nil {
		t.Errorf("AutoMigrate failed: %v", err)
	}

	user := models.User{
		FullName:     "PhoneUser",
		Email:        "phone@example.com",
		PasswordHash: "hash",
	}
	db.Create(&user)

	jwtSvc := services.NewJWTSvc("secret", db)
	authSvc := services.NewAuthSvc(db, jwtSvc)
	otpSvc := &MockOTPSvc{}
	authHandler := handlers.NewAuthHandler(authSvc, otpSvc)

	r := gin.Default()
	r.POST("/auth/request-phone-update", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		authHandler.RequestPhoneUpdate(c)
	})
	r.POST("/auth/confirm-phone-update", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		authHandler.ConfirmPhoneUpdate(c)
	})

	reqBody := models.UpdatePhoneReq{Phone: "9999999999"}
	jsonReq, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/request-phone-update", bytes.NewBuffer(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, 200, resp.Code)

	confirmReq := models.ConfirmPhoneUpdateReq{OTP: "123456"}
	jsonConfirm, _ := json.Marshal(confirmReq)
	req, _ = http.NewRequest("POST", "/auth/confirm-phone-update", bytes.NewBuffer(jsonConfirm))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, 200, resp.Code)
}

func TestUpdatePhoneAndVerifyPhone(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Errorf("AutoMigrate failed: %v", err)
	}

	user := models.User{
		FullName:     "DirectUser",
		Email:        "direct@example.com",
		PasswordHash: "hash",
	}
	db.Create(&user)

	jwtSvc := services.NewJWTSvc("secret", db)
	authSvc := services.NewAuthSvc(db, jwtSvc)
	authHandler := handlers.NewAuthHandler(authSvc, nil)

	r := gin.Default()
	r.POST("/auth/update-phone", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		authHandler.UpdatePhone(c)
	})
	r.POST("/auth/verify-phone", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		authHandler.VerifyPhone(c)
	})

	updatePhone := models.UpdatePhoneReq{Phone: "1234567890"}
	jsonPhone, _ := json.Marshal(updatePhone)
	req, _ := http.NewRequest("POST", "/auth/update-phone", bytes.NewBuffer(jsonPhone))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, 200, resp.Code)

	req, _ = http.NewRequest("POST", "/auth/verify-phone", nil)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, 200, resp.Code)
}

func TestUpdateLocation(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Errorf("AutoMigrate failed: %v", err)
	}

	user := models.User{
		FullName:     "LocationUser",
		Email:        "loc@example.com",
		PasswordHash: "hash",
	}
	db.Create(&user)

	jwtSvc := services.NewJWTSvc("secret", db)
	authSvc := services.NewAuthSvc(db, jwtSvc)
	authHandler := handlers.NewAuthHandler(authSvc, nil)

	r := gin.Default()
	r.POST("/auth/update-location", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		authHandler.UpdateLocation(c)
	})

	loc := models.UpdateLocationReq{
		Latitude:  22.57,
		Longitude: 88.36,
	}
	jsonLoc, _ := json.Marshal(loc)
	req, _ := http.NewRequest("POST", "/auth/update-location", bytes.NewBuffer(jsonLoc))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, 200, resp.Code)
}

type MockOTPSvc struct{}

func (m *MockOTPSvc) GenCode() string {
	return "123456"
}
func (m *MockOTPSvc) SaveOTP(phone, code string) error {
	return nil
}
func (m *MockOTPSvc) SendOTP(phone, code string) error {
	return nil
}
func (m *MockOTPSvc) VerifyOTP(phone, code string) error {
	if code != "123456" {
		return fmt.Errorf("invalid code")
	}
	return nil
}
