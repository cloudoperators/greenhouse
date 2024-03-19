// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Package retry contains helper methods that create retry loops using
// different retry strategies.
package retry

import (
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Strategy interface type contains methods for different retry strategies.
type Strategy interface {
	RetryUntilSuccessful(func() error)
	RetryWithBackOff(func() error)
	RetryOnce(func() error)
}

// ExponentialBackoff options.
type ExponentialBackoff struct {
	Factor        int
	MaxInterval   time.Duration
	MaxRetries    int
	MaxErrorMsg   string
	MaxErrorKey   string
	MaxErrorValue string
}

// RetryUntilSuccessful creates a retry loop with an exponential backoff.
func (eb ExponentialBackoff) RetryUntilSuccessful(action func() error) {
	duration := time.Second
	for {
		err := action()
		if err != nil {
			duration *= time.Duration(eb.Factor)
			if duration > eb.MaxInterval {
				duration = eb.MaxInterval
			}
			time.Sleep(duration)
			continue
		}
		break
	}
}

func (eb ExponentialBackoff) RetryWithBackOff(fn func() error) error {
	attempt := 1
	duration := time.Second
	for {
		err := fn()
		if err != nil {
			duration *= time.Duration(eb.Factor)
			attempt++
			if attempt > eb.MaxRetries {
				log.Log.Info("Max retries reached"+" "+eb.MaxErrorMsg, eb.MaxErrorKey, eb.MaxErrorValue)
				return fmt.Errorf("max retries reached: %w", err)
			}
			if duration > eb.MaxInterval {
				duration = eb.MaxInterval
			}
			time.Sleep(duration)
			continue
		}
		break
	}
	return nil
}

func (eb ExponentialBackoff) Retry(fn func() error) {
	attempt := 1
	for {
		err := fn()
		if err != nil {
			attempt++
			if attempt > eb.MaxRetries {
				log.Log.Info("Max retries reached"+" "+eb.MaxErrorMsg, eb.MaxErrorKey, eb.MaxErrorValue)
				break
			}
			continue
		}
		break
	}
}
