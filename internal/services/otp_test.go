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
	return services.NewOTPSvc(db, "testSID", "testToken", "+10000000000",
		"test@example.com", "testAPIKey", "testdomain.com")
}

func TestGenCodeFormat(t *testing.T) {
	otpSvc := setupOTPService(t)
	code := otpSvc.GenCode()
	assert.Len(t, code, 6)
	// Verify it's all digits
	for _, char := range code {
		assert.True(t, char >= '0' && char <= '9', "Code should contain only digits")
	}
}

func TestSaveAndVerifyOTP(t *testing.T) {
	otpSvc := setupOTPService(t)

	phone := "+19998887777"
	code := otpSvc.GenCode()

	err := otpSvc.SaveOTP(phone, code, "phone")
	assert.NoError(t, err)

	err = otpSvc.VerifyOTP(phone, code, "phone")
	assert.NoError(t, err)

	err = otpSvc.VerifyOTP(phone, code, "phone")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OTP")

	wrongCode := "000000"
	if wrongCode == code {
		wrongCode = "111111"
	}
	err = otpSvc.VerifyOTP(phone, wrongCode, "phone")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OTP")
}

func TestSaveAndVerifyOTPWithTypes(t *testing.T) {
	otpSvc := setupOTPService(t)

	phone := "+19998887777"
	phoneCode := otpSvc.GenCode()
	err := otpSvc.SaveOTP(phone, phoneCode, "phone")
	assert.NoError(t, err)
	err = otpSvc.VerifyOTP(phone, phoneCode, "phone")
	assert.NoError(t, err)

	email := "test@example.com"
	emailCode := otpSvc.GenCode()
	err = otpSvc.SaveOTP(email, emailCode, "email")
	assert.NoError(t, err)
	err = otpSvc.VerifyOTP(email, emailCode, "email")
	assert.NoError(t, err)

	newPhone := "+19998887778"
	newPhoneCode := otpSvc.GenCode()
	err = otpSvc.SaveOTP(newPhone, newPhoneCode, "phone")
	assert.NoError(t, err)

	err = otpSvc.VerifyOTP(newPhone, newPhoneCode, "email")
	assert.Error(t, err)
}

func TestExpiredOTP(t *testing.T) {
	db := testutils.SetupTestDB(t, &models.OTP{})
	otpSvc := services.NewOTPSvc(db, "testSID", "testToken", "+10000000000",
		"test@example.com", "testAPIKey", "testdomain.com")

	expired := models.OTP{
		Contact:   "+19998880000",
		Phone:     "+19998880000",
		Code:      "123456",
		Type:      "phone",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
		Used:      false,
	}
	err := db.Create(&expired).Error
	assert.NoError(t, err)

	err = otpSvc.VerifyOTP(expired.Phone, expired.Code, "phone")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestInvalidOTPContact(t *testing.T) {
	otpSvc := setupOTPService(t)

	err := otpSvc.VerifyOTP("+19999999999", "123456", "phone")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OTP")
}

func TestSendEmailOTPDryRun(t *testing.T) {
	t.Skip("Skipping actual email SendOTP test (requires valid credentials)")

	db := testutils.SetupTestDB(t, &models.OTP{})
	otpSvc := services.NewOTPSvc(db, "testSID", "testToken", "+10000000000",
		"test@example.com", "testAPIKey", "testdomain.com")

	code := "654321"
	err := otpSvc.SendEmailOTP("user@example.com", code)
	assert.NoError(t, err)
}
