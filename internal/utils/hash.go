package utils

import "golang.org/x/crypto/bcrypt"

func HashPwd(pwd string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pwd), 14)
	return string(bytes), err
}

func CheckPwd(pwd, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
	return err == nil
}
