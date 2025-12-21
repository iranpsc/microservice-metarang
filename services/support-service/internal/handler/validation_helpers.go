package handler

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/shared/pkg/helpers"
)

// returnValidationError returns a gRPC InvalidArgument error with encoded validation fields
func returnValidationError(fields map[string]string) error {
	encodedError := helpers.EncodeValidationError(fields)
	return status.Errorf(codes.InvalidArgument, encodedError)
}

// validateRequired validates that a field is not empty/zero
func validateRequired(fieldName string, value interface{}, locale string) map[string]string {
	validationErrors := make(map[string]string)
	t := helpers.GetLocaleTranslations(locale)

	switch v := value.(type) {
	case string:
		if v == "" {
			validationErrors[fieldName] = fmt.Sprintf(t.Required, fieldName)
		}
	case uint64, uint32, int64, int32:
		if v == 0 {
			validationErrors[fieldName] = fmt.Sprintf(t.Required, fieldName)
		}
	}

	return validationErrors
}

// mergeValidationErrors merges multiple validation error maps
func mergeValidationErrors(errors ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, errs := range errors {
		for field, msg := range errs {
			result[field] = msg
		}
	}
	return result
}

