package services

import (
	"fmt"
	"math/rand"
	"mride-backend/internal/models"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
)

type OTPSvc struct {
	db          *gorm.DB
	twilioSID   string
	twilioToken string
	twilioPhone string
}

func NewOTPSvc(db *gorm.DB, sid, token, phone string) *OTPSvc {
	return &OTPSvc{
		db:          db,
		twilioSID:   sid,
		twilioToken: token,
		twilioPhone: phone,
	}
}

func (o *OTPSvc) GenCode() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func (o *OTPSvc) SaveOTP(phone, code string) error {
	otp := models.OTP{
		Phone:     phone,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Used:      false,
	}

	result := o.db.Create(&otp)
	return result.Error
}

func (o *OTPSvc) VerifyOTP(phone, code string) error {
	var otp models.OTP

	result := o.db.Where("phone = ? AND code = ? AND used = ?", phone, code, false).
		Order("created_at DESC").
		First(&otp)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf("invalid OTP")
		}
		return result.Error
	}

	if time.Now().After(otp.ExpiresAt) {
		return fmt.Errorf("OTP expired")
	}

	result = o.db.Model(&otp).Update("used", true)
	return result.Error
}

func (o *OTPSvc) SendOTP(phone, code string) error {
	fmt.Printf("Sending OTP %s to phone %s\n", code, phone)

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", o.twilioSID)

	msgData := url.Values{}
	msgData.Set("To", phone)
	msgData.Set("From", o.twilioPhone)
	msgData.Set("Body", fmt.Sprintf("Your carpool app verification code is: %s", code))

	client := &http.Client{}
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(msgData.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(o.twilioSID, o.twilioToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("failed to send SMS, status code: %d", resp.StatusCode)
	}

	return nil
}

var _ OTPService = (*OTPSvc)(nil)
