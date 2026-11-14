package model

import "fmt"

// ValidateGameLifecycle checks the lifecycle fields of a game before they
// reach the database's CHECK constraints, so clients get a clear error.
func ValidateGameLifecycle(status string, platform *string, targetPriceINR *int32) error {
	if !GameStatuses[status] {
		return fmt.Errorf("invalid game status %q", status)
	}
	if platform != nil && !GamePlatforms[*platform] {
		return fmt.Errorf("invalid game platform %q", *platform)
	}
	if targetPriceINR != nil && *targetPriceINR <= 0 {
		return fmt.Errorf("target price must be positive, got %d", *targetPriceINR)
	}
	return nil
}
