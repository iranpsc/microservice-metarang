package helpers

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// CustomValidator wraps go-playground validator with custom Persian rules
type CustomValidator struct {
	validate *validator.Validate
}

// NewCustomValidator creates a new custom validator with Persian rules
func NewCustomValidator() *CustomValidator {
	v := validator.New()

	// Register custom validators
	v.RegisterValidation("persian", validatePersian)
	v.RegisterValidation("persian_alpha", validatePersianAlpha)
	v.RegisterValidation("persian_num", validatePersianNum)
	v.RegisterValidation("persian_alpha_num", validatePersianAlphaNum)
	v.RegisterValidation("iranian_mobile", validateIranianMobile)
	v.RegisterValidation("iranian_postal_code", validateIranianPostalCode)
	v.RegisterValidation("iranian_national_code", validateIranianNationalCode)

	return &CustomValidator{validate: v}
}

// Validate validates a struct
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validate.Struct(i)
}

// validatePersian validates Persian characters
func validatePersian(fl validator.FieldLevel) bool {
	persianRegex := regexp.MustCompile(`^[\x{0600}-\x{06FF}\s]+$`)
	return persianRegex.MatchString(fl.Field().String())
}

// validatePersianAlpha validates Persian alphabetic characters
func validatePersianAlpha(fl validator.FieldLevel) bool {
	persianAlphaRegex := regexp.MustCompile(`^[\x{0600}-\x{06FF}]+$`)
	return persianAlphaRegex.MatchString(fl.Field().String())
}

// validatePersianNum validates Persian numbers
func validatePersianNum(fl validator.FieldLevel) bool {
	persianNumRegex := regexp.MustCompile(`^[۰-۹]+$`)
	return persianNumRegex.MatchString(fl.Field().String())
}

// validatePersianAlphaNum validates Persian alphanumeric
func validatePersianAlphaNum(fl validator.FieldLevel) bool {
	persianAlphaNumRegex := regexp.MustCompile(`^[\x{0600}-\x{06FF}۰-۹\s]+$`)
	return persianAlphaNumRegex.MatchString(fl.Field().String())
}

// validateIranianMobile validates Iranian mobile numbers (09xxxxxxxxx)
func validateIranianMobile(fl validator.FieldLevel) bool {
	mobile := NormalizePersianNumbers(fl.Field().String())
	mobileRegex := regexp.MustCompile(`^09[0-9]{9}$`)
	return mobileRegex.MatchString(mobile)
}

// validateIranianPostalCode validates Iranian postal codes (10 digits, no dash)
func validateIranianPostalCode(fl validator.FieldLevel) bool {
	postalCode := NormalizePersianNumbers(fl.Field().String())
	// Remove any dashes or spaces
	postalCode = strings.ReplaceAll(postalCode, "-", "")
	postalCode = strings.ReplaceAll(postalCode, " ", "")
	
	postalCodeRegex := regexp.MustCompile(`^[0-9]{10}$`)
	return postalCodeRegex.MatchString(postalCode)
}

// validateIranianNationalCode validates Iranian national codes (10 digits with check digit)
func validateIranianNationalCode(fl validator.FieldLevel) bool {
	nationalCode := NormalizePersianNumbers(fl.Field().String())
	
	if len(nationalCode) != 10 {
		return false
	}

	// Check if all digits
	for _, char := range nationalCode {
		if char < '0' || char > '9' {
			return false
		}
	}

	// Validate check digit
	check := int(nationalCode[9] - '0')
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(nationalCode[i]-'0') * (10 - i)
	}

	remainder := sum % 11
	
	return (remainder < 2 && check == remainder) || (remainder >= 2 && check == 11-remainder)
}

