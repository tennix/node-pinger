package probe

import (
	"math/rand"
	"sync"
	"time"
)

type Scheduler struct {
	mu  sync.Mutex
	rnd *rand.Rand
}

func NewScheduler(seed int64) *Scheduler {
	return &Scheduler{rnd: rand.New(rand.NewSource(seed))}
}

func (s *Scheduler) Delay(interval time.Duration, factor float64) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return JitterDelay(interval, factor, s.rnd)
}

func JitterDelay(interval time.Duration, factor float64, rnd *rand.Rand) time.Duration {
	if interval <= 0 || factor <= 0 {
		return 0
	}
	maxJitter := float64(interval) * factor
	return time.Duration(rnd.Float64() * maxJitter)
}
