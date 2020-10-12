package sumologrus

import (
	"github.com/segmentio/backo-go"
	"github.com/sirupsen/logrus"
	"time"
)

type Config struct {
	EndPointURL string
	Tags        []string
	Host        string
	Level       logrus.Level
	Interval    time.Duration
	BatchSize   int
	Verbose     bool
	GZip        bool

	// The maximum number of goroutines that will be spawned by a client to send
	// requests to the backend API.
	// This field is not exported and only exposed internally to let unit tests
	// mock the current time.
	maxConcurrentRequests int

	// The retry policy used by the client to resend requests that have failed.
	// The function is called with how many times the operation has been retried
	// and is expected to return how long the client should wait before trying
	// again.
	// If not set the client will fallback to use a default retry policy.
	RetryAfter func(int) time.Duration
}

const DefaultInterval = 5 * time.Second
const DefaultBatchSize = 250

func (c *Config) validate() error {
	if c.Interval <= 0 {
		return ConfigError{
			Reason: "negative or 0 time intervals are not supported",
			Field:  "Interval",
			Value:  c.Interval,
		}
	}

	if c.BatchSize <= 0 {
		return ConfigError{
			Reason: "negative or 0 batch sizes are not supported",
			Field:  "BatchSize",
			Value:  c.BatchSize,
		}
	}

	return nil
}

func makeConfig(c Config) Config {
	if c.Interval == 0 {
		c.Interval = DefaultInterval
	}
	if c.BatchSize == 0 {
		c.BatchSize = DefaultBatchSize
	}
	if c.maxConcurrentRequests == 0 {
		c.maxConcurrentRequests = 1000
	}

	if c.RetryAfter == nil {
		c.RetryAfter = backo.DefaultBacko().Duration
	}

	return c
}
