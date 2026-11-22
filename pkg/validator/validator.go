package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

// InitValidator initializes the validator with custom rules
func InitValidator() {
	validate = validator.New()

	// Register custom validators
	validate.RegisterValidation("phone_id", validatePhoneID)
	validate.RegisterValidation("postal_code_id", validatePostalCodeID)
	validate.RegisterValidation("future_date", validateFutureDate)
}

// validatePhoneID validates Indonesian phone number format
func validatePhoneID(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	// Indonesian phone format: 08xx, 628xx, +628xx (9-13 digits after prefix)
	matched, _ := regexp.MatchString(`^(\+62|62|0)[0-9]{9,13}$`, phone)
	return matched
}

// validatePostalCodeID validates Indonesian postal code (5 digits)
func validatePostalCodeID(fl validator.FieldLevel) bool {
	postalCode := fl.Field().String()
	matched, _ := regexp.MatchString(`^[0-9]{5}$`, postalCode)
	return matched
}

// validateFutureDate validates that date is in the future
func validateFutureDate(fl validator.FieldLevel) bool {
	dateStr := fl.Field().String()
	if dateStr == "" {
		return true // allow empty for optional fields
	}

	// Parse ISO 8601 format
	date, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return false
	}

	return date.After(time.Now())
}

// Validate validates a struct
func Validate(data interface{}) error {
	if validate == nil {
		InitValidator()
	}
	return validate.Struct(data)
}

// FormatValidationErrors formats validation errors into a readable map
func FormatValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := strings.ToLower(e.Field())
			errors[field] = formatErrorMessage(e)
		}
	}

	return errors
}

func formatErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", e.Field(), e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", e.Field(), e.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", e.Field(), e.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", e.Field(), e.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", e.Field(), e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", e.Field(), e.Param())
	case "url":
		return "Invalid URL format"
	case "uuid4":
		return "Invalid UUID format"
	case "datetime":
		return "Invalid datetime format (use ISO 8601)"
	case "phone_id":
		return "Invalid Indonesian phone number format"
	case "postal_code_id":
		return "Invalid postal code (must be 5 digits)"
	case "future_date":
		return "Date must be in the future"
	default:
		return fmt.Sprintf("%s is invalid", e.Field())
	}
}
