package parsian

// ParsianError handles Parsian error codes with Persian messages
// Matches Laravel's App\Parsian\Error class exactly
type ParsianError struct {
	Code int32
}

// NewParsianError creates a new Parsian error
func NewParsianError(code int32) *ParsianError {
	return &ParsianError{Code: code}
}

// Message returns the Persian error message for the error code
// Matches Laravel's App\Parsian\Error::message() exactly
func (e *ParsianError) Message() string {
	// Exact error codes from Laravel
	switch e.Code {
	case -138:
		return "تراکنش ناموفق می باشد"
	case -127:
		return "آدرس IP معتبر نمی باشد"
	case 58:
		return "انجام تراکنش مربوطه توسط پایانه ی انجام دهنده مجاز نمی باشد"
	case -1531:
		return "تایید تراکنش ناموفق امکان پذیر نمی باشد"
	default:
		return "خطای ناشناخته"
	}
}

// GetCode returns the error code
func (e *ParsianError) GetCode() int32 {
	return e.Code
}

// IsSuccess checks if the code indicates success (0)
func (e *ParsianError) IsSuccess() bool {
	return e.Code == 0
}

