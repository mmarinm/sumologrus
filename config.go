package sumologrus

import (
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
}

const DefaultInterval = 5 * time.Second
const DefaultBatchSize = 100

func (c *Config) validate() error {
	if c.Interval < 0 {
		return ConfigError{
			Reason: "negative time intervals are not supported",
			Field:  "Interval",
			Value:  c.Interval,
		}
	}

	if c.BatchSize < 0 {
		return ConfigError{
			Reason: "negative batch sizes are not supported",
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

	return c
}
