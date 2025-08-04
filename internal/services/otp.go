package services

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"mride-backend/internal/models"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type OTPSvc struct {
	db          *gorm.DB
	twilioSID   string
	twilioToken string
	twilioPhone string

	smtpHost     string
	smtpPort     int
	smtpEmail    string
	smtpPassword string
}

func NewOTPSvc(db *gorm.DB, sid, token, phone string) *OTPSvc {
	return &OTPSvc{
		db:          db,
		twilioSID:   sid,
		twilioToken: token,
		twilioPhone: phone,

		smtpHost:     os.Getenv("SMTP_HOST"),
		smtpPort:     587,
		smtpEmail:    os.Getenv("SMTP_EMAIL"),
		smtpPassword: os.Getenv("SMTP_PASSWORD"),
	}
}

func (o *OTPSvc) GenCode() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func (o *OTPSvc) GenCryptoCode() (string, error) {
	codes := make([]byte, 6)
	if _, err := rand.Read(codes); err != nil {
		return "", err
	}
	for i := 0; i < 6; i++ {
		codes[i] = uint8(48 + (codes[i] % 10))
	}
	return string(codes), nil
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

func (o *OTPSvc) SaveEmailOTP(email, code string) error {
	otp := models.OTP{
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Used:      false,
	}

	result := o.db.Create(&otp)
	return result.Error
}

func (o *OTPSvc) VerifyOTP(phone, code string) error {
	var otp models.OTP

	result := o.db.Where("phone = ? AND code = ? AND used = ?", phone, code, false).Order("created_at DESC").First(&otp)

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

func (o *OTPSvc) VerifyEmailOTP(email, code string) error {
	var otp models.OTP

	result := o.db.Where("email = ? AND code = ? AND used = ?", email, code, false).Order("created_at DESC").First(&otp)

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

func (a *AuthSvc) EmailExists(email string) (bool, error) {
	var count int64
	err := a.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (a *AuthSvc) VerifyEmail(userID int, email string) error {
	return a.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"email_verified": true,
		"email":          email,
	}).Error
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

func (o *OTPSvc) SendEmailOTP(email, code string) error {
	fmt.Printf("Sending Email OTP %s to email %s\n", code, email)

	m := gomail.NewMessage()

	m.SetHeader("From", o.smtpEmail)

	m.SetHeader("To", email)

	m.SetHeader("Subject", "OTP for MRIDE")

	m.SetBody("text/plain", fmt.Sprintf("%s is your OTP to verify authentication for MRIDE. This OTP will expire in 5 minutes.", code))

	d := gomail.NewDialer(o.smtpHost, o.smtpPort, o.smtpEmail, o.smtpPassword)

	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

var _ OTPService = (*OTPSvc)(nil)
