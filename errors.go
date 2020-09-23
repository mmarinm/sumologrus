package sumologrus

import (
	"errors"
	"fmt"
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
	return fmt.Sprintf("NewWithConfig: %s Config.%s: %v", e.Reason, e.Field, e.Value)
}

var (
	// This error is returned by methods of the `Client` interface when they are
	// called after the client was already closed.
	ErrClosed = errors.New("the client was already closed")

	// This error is used to notify the application that too many requests are
	// already being sent and no more messages can be accepted.
	ErrTooManyRequests = errors.New("too many requests are already in-flight")
)
