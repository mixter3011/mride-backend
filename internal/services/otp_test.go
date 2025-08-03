package services_test

import (
	"testing"
	"time"

	"mride-backend/internal/models"
	"mride-backend/internal/services"
	"mride-backend/internal/testutils"

	"github.com/stretchr/testify/assert"
)

func setupOTPService(t *testing.T) *services.OTPSvc {
	db := testutils.SetupTestDB(t, &models.OTP{})
	return services.NewOTPSvc(db, "testSID", "testToken", "+10000000000")
}

func TestGenCodeFormat(t *testing.T) {
	otpSvc := setupOTPService(t)
	code := otpSvc.GenCode()
	assert.Len(t, code, 6)
	for _, char := range code {
		assert.True(t, char >= '0' && char <= '9', "Code should contain only digits")
	}
}

func TestSaveAndVerifyOTP(t *testing.T) {
	otpSvc := setupOTPService(t)

	phone := "+19998887777"
	code := otpSvc.GenCode()

	err := otpSvc.SaveOTP(phone, code)
	assert.NoError(t, err)

	err = otpSvc.VerifyOTP(phone, code)
	assert.NoError(t, err)

	err = otpSvc.VerifyOTP(phone, code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OTP")
}

func TestExpiredOTP(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.OTP{})
	otpSvc := services.NewOTPSvc(db, "testSID", "testToken", "+10000000000")

	expired := models.OTP{
		Phone:     "+19998880000",
		Code:      "123456",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
		Used:      false,
	}
	err := db.Create(&expired).Error
	assert.NoError(t, err)

	err = otpSvc.VerifyOTP(expired.Phone, expired.Code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestSendOTPDryRun(t *testing.T) {
	t.Skip("Skipping actual Twilio SendOTP test (requires valid credentials)")

	db := testutils.SetupTestDB(t, &models.OTP{})
	otpSvc := services.NewOTPSvc(db, "testSID", "testToken", "+10000000000")

	code := "654321"
	err := otpSvc.SendOTP("+19998880000", code)
	assert.NoError(t, err)
}
