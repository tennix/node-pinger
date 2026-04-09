package probe

import (
	"math/rand"
	"testing"
	"time"
)

func TestJitterDelayBounds(t *testing.T) {
	t.Parallel()

	rnd := rand.New(rand.NewSource(42))
	for range 100 {
		delay := JitterDelay(10*time.Second, 0.2, rnd)
		if delay < 0 {
			t.Fatalf("delay = %v, want >= 0", delay)
		}
		if delay > 2*time.Second {
			t.Fatalf("delay = %v, want <= 2s", delay)
		}
	}
}
