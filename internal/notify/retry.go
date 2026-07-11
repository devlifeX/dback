package notify

import (
	"context"
	"fmt"
	"time"
)

func SendWithRetry(ctx context.Context, send func(context.Context) error, delays []time.Duration) error {
	if len(delays) == 0 {
		delays = DefaultRetryDelays()
	}
	var lastErr error
	for attempt := 0; attempt <= len(delays); attempt++ {
		if err := send(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt == len(delays) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delays[attempt]):
		}
	}
	return fmt.Errorf("notify delivery failed after retries: %w", lastErr)
}
