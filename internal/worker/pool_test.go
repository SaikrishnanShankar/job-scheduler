package worker

import (
	"testing"
	"time"
)

func TestExponentialBackoff(t *testing.T) {
	cases := []struct {
		attempt int
		minWant time.Duration
		maxWant time.Duration
	}{
		{1, 2 * time.Second, 3 * time.Second},
		{2, 4 * time.Second, 5 * time.Second},
		{3, 8 * time.Second, 9 * time.Second},
		{10, 60 * time.Second, 60 * time.Second}, // capped
	}

	for _, c := range cases {
		got := exponentialBackoff(c.attempt)
		if got < c.minWant || got > c.maxWant {
			t.Errorf("exponentialBackoff(%d) = %v, want between %v and %v", c.attempt, got, c.minWant, c.maxWant)
		}
	}
}

func TestExponentialBackoffNeverExceedsCap(t *testing.T) {
	for attempt := 1; attempt <= 20; attempt++ {
		got := exponentialBackoff(attempt)
		if got > 60*time.Second {
			t.Errorf("exponentialBackoff(%d) = %v, exceeds 60s cap", attempt, got)
		}
		if got <= 0 {
			t.Errorf("exponentialBackoff(%d) = %v, must be positive", attempt, got)
		}
	}
}
