package utils

import (
	"regexp"
	"strings"
)

func ValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return re.MatchString(strings.ToLower(email))
}

func ValidPhone(phone string) bool {
	re := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	return re.MatchString(phone)
}

func ValidPwd(pwd string) bool {
	return len(pwd) >= 8
}
