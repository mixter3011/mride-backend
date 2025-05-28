package models

import (
	"time"
)

type User struct {
	ID            int       `json:"id" db:"id"`
	FullName      string    `json:"full_name" db:"full_name"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  string    `json:"-" db:"password_hash"`
	Phone         *string   `json:"phone" db:"phone"`
	PhoneVerified bool      `json:"phone_verified" db:"phone_verified"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type SignUpReq struct {
	FullName        string `json:"full_name" binding:"required,min=2,max=100"`
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

type SignInReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
}

type PhoneVerifyReq struct {
	Phone string `json:"phone" binding:"required"`
}

type OTPVerifyReq struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required,len=6"`
}

type AuthResp struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type OTP struct {
	ID        int       `json:"id" db:"id"`
	Phone     string    `json:"phone" db:"phone"`
	Code      string    `json:"code" db:"code"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	Used      bool      `json:"used" db:"used"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
