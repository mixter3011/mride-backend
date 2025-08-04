package services

import (
	"fmt"
	"math/rand"
	"mride-backend/internal/models"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/resend/resend-go/v2"
	"gorm.io/gorm"
)

type OTPSvc struct {
	db          *gorm.DB
	twilioSID   string
	twilioToken string
	twilioPhone string

	resendAPIKey string
}

func NewOTPSvc(db *gorm.DB, sid, token, phone string) *OTPSvc {
	return &OTPSvc{
		db:          db,
		twilioSID:   sid,
		twilioToken: token,
		twilioPhone: phone,

		resendAPIKey: os.Getenv("RESEND_API_KEY"),
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
	fmt.Printf("=== EMAIL OTP DEBUG START ===\n")
	fmt.Printf("Sending Email OTP %s to email %s\n", code, email)

	if o.resendAPIKey == "" {
		fmt.Printf("ERROR: RESEND_API_KEY is empty!\n")
		return fmt.Errorf("RESEND_API_KEY is not set")
	}
	fmt.Printf("API Key loaded: %s... (length: %d)\n", o.resendAPIKey[:min(10, len(o.resendAPIKey))], len(o.resendAPIKey))

	fmt.Printf("Creating Resend client...\n")
	client := resend.NewClient(o.resendAPIKey)

	params := &resend.SendEmailRequest{
		From:    "MRIDE <auth@mride.senachi.me>",
		To:      []string{email},
		Subject: "OTP for MRIDE",
		Text:    fmt.Sprintf("%s is your OTP to verify authentication for MRIDE. This OTP will expire in 5 minutes.", code),
	}

	fmt.Printf("Attempting to send email with params: From=%s, To=%s, Subject=%s\n", params.From, params.To[0], params.Subject)

	sent, err := client.Emails.Send(params)
	if err != nil {
		fmt.Printf("RESEND API ERROR: %v\n", err)
		fmt.Printf("Error type: %T\n", err)
		return fmt.Errorf("failed to send email via Resend: %v", err)
	}

	fmt.Printf("SUCCESS: Email sent! ID: %s\n", sent.Id)
	fmt.Printf("=== EMAIL OTP DEBUG END ===\n")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ OTPService = (*OTPSvc)(nil)
