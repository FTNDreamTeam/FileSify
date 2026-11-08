package server

import (
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	nameRegex  = regexp.MustCompile(`^[\p{L}][\p{L}\s'’.\-]*$`)
	emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)
)

func validateName(field, v string) error {
	trimmed := strings.TrimSpace(v)
	runes := []rune(trimmed)
	if len(runes) < 2 || len(runes) > 50 || !nameRegex.MatchString(trimmed) {
		return status.Errorf(codes.InvalidArgument,
			"%s must be 2-50 characters and contain only letters, spaces, apostrophes, periods, or hyphens", field)
	}
	return nil
}

func validateEmail(v string) error {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" || len(trimmed) > 254 || !emailRegex.MatchString(trimmed) {
		return status.Error(codes.InvalidArgument, "please enter a valid email address")
	}
	return nil
}

func validatePassword(v string) error {
	if len([]rune(v)) < 12 {
		return status.Error(codes.InvalidArgument, "password must be at least 12 characters")
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range v {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return status.Error(codes.InvalidArgument,
			"password must contain at least one uppercase letter, one lowercase letter, one digit, and one special character")
	}
	return nil
}
