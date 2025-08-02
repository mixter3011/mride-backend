package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	emailSender string
	emailAPIKey string
	emailDomain string
}

func NewOTPSvc(db *gorm.DB, sid, token, phone, emailSender, emailAPIKey, emailDomain string) *OTPSvc {
	return &OTPSvc{
		db:          db,
		twilioSID:   sid,
		twilioToken: token,
		twilioPhone: phone,
		emailSender: emailSender,
		emailAPIKey: emailAPIKey,
		emailDomain: emailDomain,
	}
}

func (o *OTPSvc) GenCode() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func (o *OTPSvc) SaveOTP(contact, code, otpType string) error {
	otp := models.OTP{
		Contact:   contact,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Used:      false,
		Type:      otpType,
	}

	if otpType == "email" {
		otp.Email = contact
	} else {
		otp.Phone = contact
	}

	result := o.db.Create(&otp)
	return result.Error
}

func (o *OTPSvc) VerifyOTP(contact, code, otpType string) error {
	var otp models.OTP

	query := o.db.Where("contact = ? AND code = ? AND used = ? AND type = ?", contact, code, false, otpType)

	result := query.Order("created_at DESC").First(&otp)

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

func (o *OTPSvc) SendEmailOTP(email, code string) error {
	fmt.Printf("Sending OTP %s to email %s\n", code, email)

	apiURL := "https://api.sendgrid.com/v3/mail/send"

	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{
					{"email": email},
				},
			},
		},
		"from": map[string]string{
			"email": o.emailSender,
			"name":  "MRide App",
		},
		"subject": "Your MRide Verification Code",
		"content": []map[string]string{
			{
				"type":  "text/plain",
				"value": fmt.Sprintf("Your MRide verification code is: %s\n\nThis code will expire in 5 minutes.\n\nIf you didn't request this code, please ignore this email.", code),
			},
			{
				"type": "text/html",
				"value": fmt.Sprintf(`<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
					<h2 style="color: #333;">MRide Verification Code</h2>
					<p>Your verification code is:</p>
					<div style="background-color: #f4f4f4; padding: 15px; text-align: center; font-size: 24px; font-weight: bold; letter-spacing: 3px; margin: 20px 0;">%s</div>
					<p>This code will expire in 5 minutes.</p>
					<p style="color: #666; font-size: 12px;">If you didn't request this code, please ignore this email.</p>
				</div>`, code),
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+o.emailAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send email, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	return nil
}

var _ OTPService = (*OTPSvc)(nil)
