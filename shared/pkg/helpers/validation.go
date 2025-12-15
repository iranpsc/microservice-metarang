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
	v.RegisterValidation("ir_sheba", validateIranianSheba)
	v.RegisterValidation("ir_bank_card_number", validateIranianBankCardNumber)

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

// validateIranianSheba validates Iranian Sheba (IBAN) numbers
// Format: IR + 24 digits (IR + 2 check digits + 22 account digits)
func validateIranianSheba(fl validator.FieldLevel) bool {
	sheba := strings.ToUpper(strings.TrimSpace(fl.Field().String()))

	// Must start with IR and be 26 characters total
	if len(sheba) != 26 || !strings.HasPrefix(sheba, "IR") {
		return false
	}

	// Check if all characters after IR are digits
	for i := 2; i < 26; i++ {
		if sheba[i] < '0' || sheba[i] > '9' {
			return false
		}
	}

	// Validate IBAN check digits using mod 97 algorithm
	return validateIBANCheckDigits(sheba)
}

// validateIBANCheckDigits validates IBAN check digits using mod 97 algorithm
func validateIBANCheckDigits(iban string) bool {
	// Move first 4 characters to end
	rearranged := iban[4:] + iban[:4]

	// Convert letters to numbers (A=10, B=11, ..., Z=35)
	var numeric string
	for _, char := range rearranged {
		if char >= '0' && char <= '9' {
			numeric += string(char)
		} else if char >= 'A' && char <= 'Z' {
			numeric += string(rune('0' + (char - 'A' + 10)))
		} else {
			return false
		}
	}

	// Calculate mod 97
	remainder := 0
	for i := 0; i < len(numeric); i++ {
		remainder = (remainder*10 + int(numeric[i]-'0')) % 97
	}

	return remainder == 1
}

// validateIranianBankCardNumber validates Iranian bank card numbers
// Format: 16 digits, with Luhn algorithm check
func validateIranianBankCardNumber(fl validator.FieldLevel) bool {
	cardNum := NormalizePersianNumbers(fl.Field().String())
	// Remove spaces and dashes
	cardNum = strings.ReplaceAll(cardNum, " ", "")
	cardNum = strings.ReplaceAll(cardNum, "-", "")

	// Must be exactly 16 digits
	if len(cardNum) != 16 {
		return false
	}

	// Check if all digits
	for _, char := range cardNum {
		if char < '0' || char > '9' {
			return false
		}
	}

	// Validate using Luhn algorithm
	return validateLuhn(cardNum)
}

// validateLuhn validates a number using the Luhn algorithm
func validateLuhn(number string) bool {
	sum := 0
	alternate := false

	// Process from right to left
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')

		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		alternate = !alternate
	}

	return sum%10 == 0
}

// ValidateIranianSheba is a standalone function to validate Iranian Sheba
func ValidateIranianSheba(sheba string) bool {
	sheba = strings.ToUpper(strings.TrimSpace(sheba))

	if len(sheba) != 26 || !strings.HasPrefix(sheba, "IR") {
		return false
	}

	for i := 2; i < 26; i++ {
		if sheba[i] < '0' || sheba[i] > '9' {
			return false
		}
	}

	return validateIBANCheckDigits(sheba)
}

// ValidateIranianBankCardNumber is a standalone function to validate Iranian bank card numbers
func ValidateIranianBankCardNumber(cardNum string) bool {
	cardNum = NormalizePersianNumbers(cardNum)
	cardNum = strings.ReplaceAll(cardNum, " ", "")
	cardNum = strings.ReplaceAll(cardNum, "-", "")

	if len(cardNum) != 16 {
		return false
	}

	for _, char := range cardNum {
		if char < '0' || char > '9' {
			return false
		}
	}

	return validateLuhn(cardNum)
}
