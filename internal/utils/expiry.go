package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseExpiry converts expiry string (1H, 1D, 1M, 1Y) to time.Time
func ParseExpiry(expiry string) (time.Time, error) {
	if len(expiry) < 2 {
		return time.Time{}, fmt.Errorf("invalid expiry format: %s", expiry)
	}

	// Extract number and unit
	valueStr := expiry[:len(expiry)-1]
	unit := strings.ToUpper(string(expiry[len(expiry)-1]))

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid expiry value: %s", expiry)
	}

	now := time.Now()
	var expiresAt time.Time

	switch unit {
	case "H":
		expiresAt = now.Add(time.Duration(value) * time.Hour)
	case "D":
		expiresAt = now.Add(time.Duration(value) * 24 * time.Hour)
	case "M":
		expiresAt = now.AddDate(0, value, 0)
	case "Y":
		expiresAt = now.AddDate(value, 0, 0)
	default:
		return time.Time{}, fmt.Errorf("invalid expiry unit: %s (use H, D, M, or Y)", unit)
	}

	return expiresAt, nil
}

// ValidatePermissions checks if all permissions are valid
func ValidatePermissions(permissions []string) error {
	validPermissions := map[string]bool{
		"deposit":  true,
		"transfer": true,
		"read":     true,
	}

	if len(permissions) == 0 {
		return fmt.Errorf("at least one permission is required")
	}

	for _, perm := range permissions {
		if !validPermissions[perm] {
			return fmt.Errorf("invalid permission: %s (valid: deposit, transfer, read)", perm)
		}
	}

	return nil
}
