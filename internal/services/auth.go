package services

import (
	"database/sql"
	"fmt"
	"strings"

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
	query := `SELECT id, full_name, email, phone, phone_verified, latitude, longitude, created_at, updated_at 
			 FROM users WHERE id = $1`

	err := a.db.QueryRow(query, id).Scan(
		&user.ID, &user.FullName, &user.Email, &user.Phone, &user.PhoneVerified,
		&user.Latitude, &user.Longitude, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (a *AuthSvc) UpdateProfile(userID int, req models.UpdateProfileReq) error {
	if req.Email != "" {
		var exists bool
		err := a.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND id != $2)", req.Email, userID).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("email already registered")
		}
	}

	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.FullName != "" {
		setParts = append(setParts, fmt.Sprintf("full_name = $%d", argIndex))
		args = append(args, req.FullName)
		argIndex++
	}

	if req.Email != "" {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, req.Email)
		argIndex++
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = NOW()"))
	args = append(args, userID)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d",
		strings.Join(setParts, ", "), argIndex)

	_, err := a.db.Exec(query, args...)
	return err
}

func (a *AuthSvc) UpdatePassword(userID int, currentPassword, newPassword string) error {
	var currentHash string
	err := a.db.QueryRow("SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		return err
	}

	if !utils.CheckPwd(currentPassword, currentHash) {
		return fmt.Errorf("current password is incorrect")
	}

	newHash, err := utils.HashPwd(newPassword)
	if err != nil {
		return err
	}

	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err = a.db.Exec(query, newHash, userID)
	return err
}

func (a *AuthSvc) PhoneExists(phone string) (bool, error) {
	var exists bool
	err := a.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)", phone).Scan(&exists)
	return exists, err
}

func (a *AuthSvc) StorePendingPhoneUpdate(userID int, phone string) error {
	query := `INSERT INTO pending_phone_updates (user_id, new_phone, created_at) 
			 VALUES ($1, $2, NOW())
			 ON CONFLICT (user_id) 
			 DO UPDATE SET new_phone = $2, created_at = NOW()`
	_, err := a.db.Exec(query, userID, phone)
	return err
}

func (a *AuthSvc) GetPendingPhoneUpdate(userID int) (string, error) {
	var phone string
	query := `SELECT new_phone FROM pending_phone_updates WHERE user_id = $1`
	err := a.db.QueryRow(query, userID).Scan(&phone)
	return phone, err
}

func (a *AuthSvc) UpdatePhoneVerified(userID int, phone string) error {
	query := `UPDATE users SET phone = $1, phone_verified = true, updated_at = NOW() WHERE id = $2`
	_, err := a.db.Exec(query, phone, userID)
	return err
}

func (a *AuthSvc) ClearPendingPhoneUpdate(userID int) {
	a.db.Exec("DELETE FROM pending_phone_updates WHERE user_id = $1", userID)
}

func (a *AuthSvc) UpdateLocation(userID int, latitude, longitude float64) error {
	query := `UPDATE users SET latitude = $1, longitude = $2, updated_at = NOW() WHERE id = $3`
	_, err := a.db.Exec(query, latitude, longitude, userID)
	return err
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
