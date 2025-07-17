package services

import (
	"fmt"
	"time"

	"mride-backend/internal/models"
	"mride-backend/internal/utils"

	"gorm.io/gorm"
)

type AuthSvc struct {
	db     *gorm.DB
	jwtSvc *JWTSvc
}

func NewAuthSvc(db *gorm.DB, jwtSvc *JWTSvc) *AuthSvc {
	return &AuthSvc{
		db:     db,
		jwtSvc: jwtSvc,
	}
}

func (a *AuthSvc) SignUp(req models.SignUpReq) (*models.AuthResp, error) {
	if req.Password != req.ConfirmPassword {
		return nil, fmt.Errorf("passwords don't match")
	}

	var count int64
	if err := a.db.Model(&models.User{}).Where("email = ?", req.Email).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("email already registered")
	}

	pwdHash, err := utils.HashPwd(req.Password)
	if err != nil {
		return nil, err
	}

	user := models.User{
		FullName:     req.FullName,
		Email:        req.Email,
		PasswordHash: pwdHash,
	}

	if err := a.db.Create(&user).Error; err != nil {
		return nil, err
	}

	accessToken, err := a.jwtSvc.GenToken(int(user.ID), user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := a.jwtSvc.GenRefreshToken(int(user.ID))
	if err != nil {
		return nil, err
	}

	user.RefreshToken = refreshToken
	user.RefreshTokenExpiry = time.Now().Add(7 * 24 * time.Hour)

	if err := a.db.Save(&user).Error; err != nil {
		return nil, err
	}

	return &models.AuthResp{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (a *AuthSvc) SignIn(req models.SignInReq) (*models.AuthResp, error) {
	var user models.User
	if err := a.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, err
	}

	if !utils.CheckPwd(req.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	accessToken, err := a.jwtSvc.GenToken(int(user.ID), user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := a.jwtSvc.GenRefreshToken(int(user.ID))
	if err != nil {
		return nil, err
	}

	user.RefreshToken = refreshToken
	user.RefreshTokenExpiry = time.Now().Add(7 * 24 * time.Hour)

	if err := a.db.Save(&user).Error; err != nil {
		return nil, err
	}

	return &models.AuthResp{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (a *AuthSvc) GetUserByID(id int) (*models.User, error) {
	var user models.User
	if err := a.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (a *AuthSvc) UpdateProfile(userID int, req models.UpdateProfileReq) error {
	if req.Email != "" {
		var count int64
		if err := a.db.Model(&models.User{}).Where("email = ? AND id != ?", req.Email, userID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("email already registered")
		}
	}

	updates := make(map[string]interface{})

	if req.FullName != "" {
		updates["full_name"] = req.FullName
	}

	if req.Email != "" {
		updates["email"] = req.Email
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	return a.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
}

func (a *AuthSvc) UpdatePassword(userID int, currentPassword, newPassword string) error {
	var user models.User
	if err := a.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	if !utils.CheckPwd(currentPassword, user.PasswordHash) {
		return fmt.Errorf("current password is incorrect")
	}

	newHash, err := utils.HashPwd(newPassword)
	if err != nil {
		return err
	}

	return a.db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", newHash).Error
}

func (a *AuthSvc) PhoneExists(phone string) (bool, error) {
	var count int64
	if err := a.db.Model(&models.User{}).Where("phone = ?", phone).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (a *AuthSvc) StorePendingPhoneUpdate(userID int, phone string) error {
	pendingUpdate := models.PendingPhoneUpdate{
		UserID:   uint(userID),
		NewPhone: phone,
	}

	return a.db.Create(&pendingUpdate).Error
}

func (a *AuthSvc) GetPendingPhoneUpdate(userID int) (string, error) {
	var pendingUpdate models.PendingPhoneUpdate
	if err := a.db.Where("user_id = ?", userID).First(&pendingUpdate).Error; err != nil {
		return "", err
	}
	return pendingUpdate.NewPhone, nil
}

func (a *AuthSvc) UpdatePhoneVerified(userID int, phone string) error {
	return a.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"phone":          phone,
		"phone_verified": true,
	}).Error
}

func (a *AuthSvc) ClearPendingPhoneUpdate(userID int) {
	a.db.Where("user_id = ?", userID).Delete(&models.PendingPhoneUpdate{})
}

func (a *AuthSvc) UpdateLocation(userID int, latitude, longitude float64) error {
	return a.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"latitude":  latitude,
		"longitude": longitude,
	}).Error
}

func (a *AuthSvc) UpdatePhone(userID int, phone string) error {
	return a.db.Model(&models.User{}).Where("id = ?", userID).Update("phone", phone).Error
}

func (a *AuthSvc) VerifyPhone(userID int) error {
	return a.db.Model(&models.User{}).Where("id = ?", userID).Update("phone_verified", true).Error
}

func (a *AuthSvc) ValidateRefreshToken(token string) (*Claims, error) {
	return a.jwtSvc.ValidToken(token)
}

func (a *AuthSvc) GenToken(userID int, email string) (string, error) {
	return a.jwtSvc.GenToken(userID, email)
}

func (a *AuthSvc) Logout(userID int) error {
	return a.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"refresh_token":        "",
		"refresh_token_expiry": time.Time{},
	}).Error
}

func (a *AuthSvc) GetDB() *gorm.DB {
	return a.db
}
