package time

import (
	"context"
	"time"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
)

// RealTimeProvider implements the TimeProvider interface with real time operations
type RealTimeProvider struct{}

// NewRealTimeProvider creates a new real time provider
func NewRealTimeProvider() core.TimeProvider {
	return &RealTimeProvider{}
}

// Now returns the current time
func (p *RealTimeProvider) Now() time.Time {
	return time.Now()
}

// Since returns the time elapsed since t
func (p *RealTimeProvider) Since(t time.Time) core.Duration {
	return core.Duration(time.Since(t))
}

// Until returns the duration until t
func (p *RealTimeProvider) Until(t time.Time) core.Duration {
	return core.Duration(time.Until(t))
}

// Sleep pauses the current goroutine for the specified duration
func (p *RealTimeProvider) Sleep(d core.Duration) {
	time.Sleep(d.Std())
}

// WithTimeout returns a context that will be canceled after the specified timeout
func (p *RealTimeProvider) WithTimeout(ctx context.Context, timeout core.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout.Std())
}

// ParseDuration parses a duration string
func (p *RealTimeProvider) ParseDuration(s string) (core.Duration, error) {
	d, err := time.ParseDuration(s)
	return core.Duration(d), err
}
