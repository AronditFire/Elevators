package runner

import (
	"context"
	"time"
)

type Advancer interface{ Advance(float64) }

func Run(ctx context.Context, engine Advancer) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			// Берём фактически прошедшее время, чтобы задержки планировщика не делали симуляцию медленнее заданного темпа.
			dt := now.Sub(last).Seconds()
			last = now
			engine.Advance(dt)
		}
	}
}
