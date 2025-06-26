package core

import (
	"context"
	"time"
)

// Duration is a domain-specific wrapper around time.Duration
type Duration time.Duration

// Common duration constants
const (
	Nanosecond  Duration = Duration(time.Nanosecond)
	Microsecond          = Duration(time.Microsecond)
	Millisecond          = Duration(time.Millisecond)
	Second               = Duration(time.Second)
	Minute               = Duration(time.Minute)
	Hour                 = Duration(time.Hour)
)

// Std converts domain Duration to time.Duration
func (d Duration) Std() time.Duration {
	return time.Duration(d)
}

// TimeProvider abstracts time operations for the domain
type TimeProvider interface {
	Now() time.Time
	Since(t time.Time) Duration
	Until(t time.Time) Duration
	Sleep(d Duration)
	WithTimeout(ctx context.Context, timeout Duration) (context.Context, context.CancelFunc)
	ParseDuration(s string) (Duration, error)
}
