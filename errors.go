package sumologrus

import (
	"fmt"
	"errors"
)

// ConfigError is returned by the `NewWithConfig` function when the one of the configuration
// fields was set to an impossible value (like a negative duration).
type ConfigError struct {
	Reason string
	Field  string

	// The value of the configuration field that caused the error.
	Value interface{}
}

func (e ConfigError) Error() string {
	return fmt.Sprintf("analytics.NewWithConfig: %s (analytics.Config.%s: %#v)", e.Reason, e.Field, e.Value)
}

