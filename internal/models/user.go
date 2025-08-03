package models

import (
	"time"
)

type User struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	FullName           string    `gorm:"not null" json:"full_name"`
	Email              string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash       string    `gorm:"not null" json:"-"`
	Phone              *string   `gorm:"uniqueIndex" json:"phone"`
	PhoneVerified      bool      `gorm:"default:false" json:"phone_verified"`
	Latitude           *float64  `json:"latitude"`
	Longitude          *float64  `json:"longitude"`
	RefreshToken       string    `gorm:"type:text" json:"-"`
	RefreshTokenExpiry time.Time `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
type UpdateProfileReq struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}
type UpdatePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}
type UpdatePhoneReq struct {
	Phone string `json:"phone" binding:"required"`
}
type ConfirmPhoneUpdateReq struct {
	OTP string `json:"otp" binding:"required"`
}
type UpdateLocationReq struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}
type PendingPhoneUpdate struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	NewPhone  string    `gorm:"not null" json:"new_phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type SignUpReq struct {
	FullName        string `json:"full_name" binding:"required,min=2,max=100"`
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
	Phone           string `json:"phone" binding:"required"`
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
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}
type OTP struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Phone     string    `gorm:"not null" json:"phone"`
	Code      string    `gorm:"not null" json:"code"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
