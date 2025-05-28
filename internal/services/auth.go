package services

import (
	"database/sql"
	"fmt"

	"mride-backend/internal/models"
	"mride-backend/internal/utils"
)

type AuthSvc struct {
	db     *sql.DB
	jwtSvc *JWTSvc
}

func NewAuthSvc(db *sql.DB, jwtSvc *JWTSvc) *AuthSvc {
	return &AuthSvc{
		db:     db,
		jwtSvc: jwtSvc,
	}
}

func (a *AuthSvc) SignUp(req models.SignUpReq) (*models.AuthResp, error) {
	if req.Password != req.ConfirmPassword {
		return nil, fmt.Errorf("passwords don't match")
	}

	var exists bool
	err := a.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	pwdHash, err := utils.HashPwd(req.Password)
	if err != nil {
		return nil, err
	}

	var user models.User
	query := `INSERT INTO users (full_name, email, password_hash) 
			 VALUES ($1, $2, $3) 
			 RETURNING id, full_name, email, phone, phone_verified, created_at, updated_at`

	err = a.db.QueryRow(query, req.FullName, req.Email, pwdHash).Scan(
		&user.ID, &user.FullName, &user.Email, &user.Phone, &user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	token, err := a.jwtSvc.GenToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &models.AuthResp{
		Token: token,
		User:  user,
	}, nil
}

func (a *AuthSvc) SignIn(req models.SignInReq) (*models.AuthResp, error) {
	var user models.User
	query := `SELECT id, full_name, email, password_hash, phone, phone_verified, created_at, updated_at 
			 FROM users WHERE email = $1`

	err := a.db.QueryRow(query, req.Email).Scan(
		&user.ID, &user.FullName, &user.Email, &user.PasswordHash, &user.Phone, &user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, err
	}

	if !utils.CheckPwd(req.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	token, err := a.jwtSvc.GenToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &models.AuthResp{
		Token: token,
		User:  user,
	}, nil
}

func (a *AuthSvc) GetUserByID(id int) (*models.User, error) {
	var user models.User
	query := `SELECT id, full_name, email, phone, phone_verified, created_at, updated_at 
			 FROM users WHERE id = $1`

	err := a.db.QueryRow(query, id).Scan(
		&user.ID, &user.FullName, &user.Email, &user.Phone, &user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (a *AuthSvc) UpdatePhone(userID int, phone string) error {
	query := `UPDATE users SET phone = $1 WHERE id = $2`
	_, err := a.db.Exec(query, phone, userID)
	return err
}

func (a *AuthSvc) VerifyPhone(userID int) error {
	query := `UPDATE users SET phone_verified = true WHERE id = $1`
	_, err := a.db.Exec(query, userID)
	return err
}
