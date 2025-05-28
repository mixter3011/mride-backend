package services

import (
	"database/sql"
	"fmt"
	"math/rand"
	"mride-backend/internal/models"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OTPSvc struct {
	db          *sql.DB
	twilioSID   string
	twilioToken string
	twilioPhone string
}

func NewOTPSvc(db *sql.DB, sid, token, phone string) *OTPSvc {
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
	query := `INSERT INTO otps (phone, code, expires_at) VALUES ($1, $2, $3)`
	expiresAt := time.Now().Add(5 * time.Minute)
	_, err := o.db.Exec(query, phone, code, expiresAt)
	return err
}

func (o *OTPSvc) VerifyOTP(phone, code string) error {
	var otp models.OTP
	query := `SELECT id, phone, code, expires_at, used FROM otps
		WHERE phone = $1 AND code = $2 AND used = false
		ORDER BY created_at DESC LIMIT 1`

	err := o.db.QueryRow(query, phone, code).Scan(
		&otp.ID, &otp.Phone, &otp.Code, &otp.ExpiresAt, &otp.Used,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("invalid OTP")
		}
		return err
	}

	if time.Now().After(otp.ExpiresAt) {
		return fmt.Errorf("OTP expired")
	}

	updateQuery := `UPDATE otps SET used = true WHERE id = $1`
	_, err = o.db.Exec(updateQuery, otp.ID)
	return err
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
