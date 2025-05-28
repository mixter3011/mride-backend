package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTSvc struct {
	secret []byte
}

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func NewJWTSvc(secret string) *JWTSvc {
	return &JWTSvc{
		secret: []byte(secret),
	}
}

func (j *JWTSvc) GenToken(userID int, email string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWTSvc) ValidToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return j.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
