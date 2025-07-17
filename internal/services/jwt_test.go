package services_test

import (
	"testing"
	"time"

	"mride-backend/internal/services"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupJWTService(t *testing.T) *services.JWTSvc {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	return services.NewJWTSvc("testsecretkey", db)
}

func TestGenAndValidateToken(t *testing.T) {
	jwtSvc := setupJWTService(t)

	token, err := jwtSvc.GenToken(123, "test@example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwtSvc.ValidToken(token)
	assert.NoError(t, err)
	assert.Equal(t, 123, claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
}

func TestInvalidToken(t *testing.T) {
	jwtSvc := setupJWTService(t)

	_, err := jwtSvc.ValidToken("invalid.token.value")
	assert.Error(t, err)
}

func TestExpiredToken(t *testing.T) {
	jwtSvc := setupJWTService(t)

	expiredClaims := &services.Claims{
		UserID: 123,
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenStr, err := token.SignedString([]byte("testsecretkey"))
	assert.NoError(t, err)

	_, err = jwtSvc.ValidToken(tokenStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestGenRefreshToken(t *testing.T) {
	jwtSvc := setupJWTService(t)

	token, err := jwtSvc.GenRefreshToken(456)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwtSvc.ValidToken(token)
	assert.NoError(t, err)
	assert.Equal(t, 456, claims.UserID)

	diff := time.Until(claims.ExpiresAt.Time)
	assert.GreaterOrEqual(t, int(diff.Hours()), 167)
	assert.LessOrEqual(t, int(diff.Hours()), 169)
}
