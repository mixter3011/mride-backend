package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mride-backend/internal/handlers"
	"mride-backend/internal/models"
	"mride-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MockOTPService struct {
	mock.Mock
}

func (m *MockOTPService) GenCode() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockOTPService) GenCryptoCode() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockOTPService) SaveOTP(phone, code string) error {
	args := m.Called(phone, code)
	return args.Error(0)
}

func (m *MockOTPService) SaveEmailOTP(email, code string) error {
	args := m.Called(email, code)
	return args.Error(0)
}

func (m *MockOTPService) SendOTP(phone, code string) error {
	args := m.Called(phone, code)
	return args.Error(0)
}

func (m *MockOTPService) SendEmailOTP(email, code string) error {
	args := m.Called(email, code)
	return args.Error(0)
}

func (m *MockOTPService) VerifyOTP(phone, code string) error {
	args := m.Called(phone, code)
	return args.Error(0)
}

func (m *MockOTPService) VerifyEmailOTP(email, code string) error {
	args := m.Called(email, code)
	return args.Error(0)
}

func setupAuthSvcForOTP(t *testing.T) *services.AuthSvc {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.OTP{})
	assert.NoError(t, err)

	jwtSvc := services.NewJWTSvc("secret", db)
	return services.NewAuthSvc(db, jwtSvc)
}

func TestSendOTPHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockOTP := new(MockOTPService)
	authSvc := setupAuthSvcForOTP(t)

	user := models.User{
		FullName: "OTP User",
		Email:    "otp@example.com",
		Phone:    nil,
	}
	db := authSvc.GetDB()
	err := db.Create(&user).Error
	assert.NoError(t, err)

	mockOTP.On("GenCode").Return("123456")
	mockOTP.On("SaveOTP", "+1234567890", "123456").Return(nil)
	mockOTP.On("SendOTP", "+1234567890", "123456").Return(nil)

	handler := handlers.NewOTPHandler(mockOTP, authSvc)

	router := gin.Default()
	router.POST("/send-otp", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		handler.SendOTP(c)
	})

	payload := map[string]string{"phone": "+1234567890"}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/send-otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockOTP.AssertExpectations(t)
}

func TestVerifyOTPHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockOTP := new(MockOTPService)
	authSvc := setupAuthSvcForOTP(t)

	phone := "+1234567890"
	user := models.User{
		FullName: "Verify User",
		Email:    "verify@example.com",
		Phone:    &phone,
	}
	db := authSvc.GetDB()
	err := db.Create(&user).Error
	assert.NoError(t, err)

	mockOTP.On("VerifyOTP", "+1234567890", "123456").Return(nil)

	handler := handlers.NewOTPHandler(mockOTP, authSvc)

	router := gin.Default()
	router.POST("/verify-otp", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		handler.VerifyOTP(c)
	})

	payload := map[string]string{
		"phone": "+1234567890",
		"code":  "123456",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/verify-otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockOTP.AssertExpectations(t)
}

func TestSendOTPHandlerPhoneAlreadyExists(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockOTP := new(MockOTPService)
	authSvc := setupAuthSvcForOTP(t)

	existingPhone := "+1234567890"
	existingUser := models.User{
		FullName: "Existing User",
		Email:    "existing@example.com",
		Phone:    &existingPhone,
	}
	newUser := models.User{
		FullName: "New User",
		Email:    "new@example.com",
		Phone:    nil,
	}

	db := authSvc.GetDB()
	err := db.Create(&existingUser).Error
	assert.NoError(t, err)
	err = db.Create(&newUser).Error
	assert.NoError(t, err)

	handler := handlers.NewOTPHandler(mockOTP, authSvc)

	router := gin.Default()
	router.POST("/send-otp", func(c *gin.Context) {
		c.Set("user_id", int(newUser.ID))
		handler.SendOTP(c)
	})

	payload := map[string]string{"phone": "+1234567890"}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/send-otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "already registered")
}

func TestSendOTPHandlerInvalidPhone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockOTP := new(MockOTPService)
	authSvc := setupAuthSvcForOTP(t)

	user := models.User{
		FullName: "Test User",
		Email:    "test@example.com",
	}
	db := authSvc.GetDB()
	err := db.Create(&user).Error
	assert.NoError(t, err)

	handler := handlers.NewOTPHandler(mockOTP, authSvc)

	router := gin.Default()
	router.POST("/send-otp", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		handler.SendOTP(c)
	})

	payload := map[string]string{"phone": "invalid-phone"}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/send-otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid phone format")
}

func TestSendEmailOTPHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockOTP := new(MockOTPService)
	authSvc := setupAuthSvcForOTP(t)

	user := models.User{
		FullName: "Email OTP User",
		Email:    "original@example.com",
	}
	db := authSvc.GetDB()
	err := db.Create(&user).Error
	assert.NoError(t, err)

	mockOTP.On("GenCode").Return("123456")
	mockOTP.On("SaveEmailOTP", "newemail@example.com", "123456").Return(nil)
	mockOTP.On("SendEmailOTP", "newemail@example.com", "123456").Return(nil)

	handler := handlers.NewOTPHandler(mockOTP, authSvc)

	router := gin.Default()
	router.POST("/send-email-otp", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		handler.SendEmailOTP(c)
	})

	payload := map[string]string{"email": "newemail@example.com"}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/send-email-otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockOTP.AssertExpectations(t)
}

func TestVerifyEmailOTPHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockOTP := new(MockOTPService)
	authSvc := setupAuthSvcForOTP(t)

	user := models.User{
		FullName: "Verify Email User",
		Email:    "verify@example.com",
	}
	db := authSvc.GetDB()
	err := db.Create(&user).Error
	assert.NoError(t, err)

	mockOTP.On("VerifyEmailOTP", "verify@example.com", "123456").Return(nil)

	handler := handlers.NewOTPHandler(mockOTP, authSvc)

	router := gin.Default()
	router.POST("/verify-email-otp", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		handler.VerifyEmailOTP(c)
	})

	payload := map[string]string{
		"email": "verify@example.com",
		"code":  "123456",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/verify-email-otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockOTP.AssertExpectations(t)
}

func TestSendEmailOTPHandlerInvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockOTP := new(MockOTPService)
	authSvc := setupAuthSvcForOTP(t)

	user := models.User{
		FullName: "Test User",
		Email:    "test@example.com",
	}
	db := authSvc.GetDB()
	err := db.Create(&user).Error
	assert.NoError(t, err)

	handler := handlers.NewOTPHandler(mockOTP, authSvc)

	router := gin.Default()
	router.POST("/send-email-otp", func(c *gin.Context) {
		c.Set("user_id", int(user.ID))
		handler.SendEmailOTP(c)
	})

	payload := map[string]string{"email": "invalid-email-format"}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/send-email-otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid email format")
}
