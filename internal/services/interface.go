package services

type OTPService interface {
	GenCode() string
	SaveOTP(phone, code string) error
	SendOTP(phone, code string) error
	VerifyOTP(phone, code string) error
}
