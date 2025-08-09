package services_test

import (
	"testing"
	"time"

	"mride-backend/internal/models"
	"mride-backend/internal/services"
	"mride-backend/internal/testutils"
	"mride-backend/internal/utils"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func createAuthSvc(t *testing.T) (*services.AuthSvc, *gorm.DB) {
	db := testutils.SetupTestDB(t, &models.User{}, &models.PendingPhoneUpdate{})
	jwtSvc := services.NewJWTSvc("test-secret", db)
	authSvc := services.NewAuthSvc(db, jwtSvc)
	return authSvc, db
}

func TestAuthSignUpSignInLogoutFlow(t *testing.T) {
	authSvc, db := createAuthSvc(t)

	signUp := models.SignUpReq{
		FullName:        "Test User",
		Email:           "test@example.com",
		Password:        "securePass123",
		ConfirmPassword: "securePass123",
	}

	resp, err := authSvc.SignUp(signUp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, signUp.Email, resp.User.Email)

	assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), resp.User.RefreshTokenExpiry, time.Hour)

	signIn := models.SignInReq{
		Email:    signUp.Email,
		Password: signUp.Password,
		Phone:    "1234567890",
	}
	signInResp, err := authSvc.SignIn(signIn)
	assert.NoError(t, err)
	assert.NotEmpty(t, signInResp.Token)
	assert.NotEmpty(t, signInResp.RefreshToken)

	var user models.User
	err = db.First(&user, signInResp.User.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, signInResp.RefreshToken, user.RefreshToken)

	err = authSvc.Logout(int(user.ID))
	assert.NoError(t, err)

	var loggedOut models.User
	err = db.First(&loggedOut, user.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "", loggedOut.RefreshToken)
	assert.True(t, loggedOut.RefreshTokenExpiry.IsZero())
}

func TestGetUserByID(t *testing.T) {
	authSvc, db := createAuthSvc(t)

	user := models.User{
		FullName:     "Jane Doe",
		Email:        "jane@example.com",
		PasswordHash: "hashed",
	}
	assert.NoError(t, db.Create(&user).Error)

	found, err := authSvc.GetUserByID(int(user.ID))
	assert.NoError(t, err)
	assert.Equal(t, user.Email, found.Email)
}
func TestUpdateProfile(t *testing.T) {
	authSvc, db := createAuthSvc(t)

	user := models.User{
		FullName:     "Old Name",
		Email:        "old@example.com",
		PasswordHash: "hash",
	}
	db.Create(&user)

	err := authSvc.UpdateProfile(int(user.ID), models.UpdateProfileReq{
		FullName: "New Name",
		Email:    "new@example.com",
	})
	assert.NoError(t, err)

	var updated models.User
	db.First(&updated, user.ID)
	assert.Equal(t, "New Name", updated.FullName)
	assert.Equal(t, "new@example.com", updated.Email)

	db.Create(&models.User{FullName: "Other", Email: "dup@example.com", PasswordHash: "x"})
	err = authSvc.UpdateProfile(int(user.ID), models.UpdateProfileReq{
		Email: "dup@example.com",
	})
	assert.EqualError(t, err, "email already registered")

	err = authSvc.UpdateProfile(int(user.ID), models.UpdateProfileReq{})
	assert.EqualError(t, err, "no fields to update")
}

func TestUpdatePassword(t *testing.T) {
	authSvc, db := createAuthSvc(t)

	hashedPwd, _ := utils.HashPwd("oldpass")
	user := models.User{
		FullName:     "User",
		Email:        "pass@example.com",
		PasswordHash: hashedPwd,
	}
	db.Create(&user)

	err := authSvc.UpdatePassword(int(user.ID), "oldpass", "newpass")
	assert.NoError(t, err)

	var updated models.User
	db.First(&updated, user.ID)
	assert.True(t, utils.CheckPwd("newpass", updated.PasswordHash))

	err = authSvc.UpdatePassword(int(user.ID), "wrongpass", "another")
	assert.EqualError(t, err, "current password is incorrect")
}

func TestPhoneExists(t *testing.T) {
	authSvc, db := createAuthSvc(t)
	phone := "9999999999"

	ok, err := authSvc.PhoneExists(phone)
	assert.NoError(t, err)
	assert.False(t, ok)

	db.Create(&models.User{FullName: "P", Email: "p@example.com", PasswordHash: "x", Phone: &phone})

	ok, err = authSvc.PhoneExists(phone)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestPendingPhoneUpdateFlow(t *testing.T) {
	authSvc, db := createAuthSvc(t)

	user := models.User{FullName: "U", Email: "u@example.com", PasswordHash: "x"}
	db.Create(&user)

	err := authSvc.StorePendingPhoneUpdate(int(user.ID), "12345")
	assert.NoError(t, err)

	phone, err := authSvc.GetPendingPhoneUpdate(int(user.ID))
	assert.NoError(t, err)
	assert.Equal(t, "12345", phone)

	authSvc.ClearPendingPhoneUpdate(int(user.ID))
	var count int64
	db.Model(&models.PendingPhoneUpdate{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestPhoneAndLocationUpdates(t *testing.T) {
	authSvc, db := createAuthSvc(t)

	user := models.User{FullName: "Loc", Email: "loc@example.com", PasswordHash: "x"}
	db.Create(&user)

	err := authSvc.UpdatePhoneVerified(int(user.ID), "8888888888")
	assert.NoError(t, err)

	var updated models.User
	db.First(&updated, user.ID)
	assert.Equal(t, "8888888888", *updated.Phone)
	assert.True(t, updated.PhoneVerified)

	err = authSvc.UpdatePhone(int(user.ID), "7777777777")
	assert.NoError(t, err)

	err = authSvc.VerifyPhone(int(user.ID))
	assert.NoError(t, err)
	db.First(&updated, user.ID)
	assert.Equal(t, "7777777777", *updated.Phone)
	assert.True(t, updated.PhoneVerified)

	err = authSvc.UpdateLocation(int(user.ID), 22.5, 88.6)
	assert.NoError(t, err)
	db.First(&updated, user.ID)
	assert.Equal(t, 22.5, *updated.Latitude)
	assert.Equal(t, 88.6, *updated.Longitude)
}

func TestValidateRefreshToken(t *testing.T) {
	authSvc, db := createAuthSvc(t)

	user := models.User{
		FullName:     "Test User",
		Email:        "test@example.com",
		PasswordHash: "hash",
	}
	assert.NoError(t, db.Create(&user).Error)

	refreshToken, err := authSvc.GenRefreshToken(int(user.ID))
	assert.NoError(t, err)

	user.RefreshToken = refreshToken
	user.RefreshTokenExpiry = time.Now().Add(7 * 24 * time.Hour)
	assert.NoError(t, db.Save(&user).Error)

	claims, err := authSvc.ValidateRefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, int(user.ID), claims.UserID)

	_, err = authSvc.ValidateRefreshToken("invalid.token.value")
	assert.Error(t, err)

	otherToken, _ := authSvc.GenRefreshToken(999)
	_, err = authSvc.ValidateRefreshToken(otherToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refresh token not found or invalid")
}
