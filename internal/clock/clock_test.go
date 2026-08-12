package clock

import (
	"testing"
	"time"
)

func TestFakeClockAdvance(t *testing.T) {
	f := &FakeClock{T: time.Unix(0, 0)}
	f.Advance(time.Minute)
	if !f.Now().Equal(time.Unix(60, 0)) {
		t.Fatalf("Now() = %v, want %v", f.Now(), time.Unix(60, 0))
	}
}

func TestClocksImplementClock(t *testing.T) {
	var _ Clock = RealClock{}
	var _ Clock = &FakeClock{}

	if (RealClock{}).Now().Location() != time.UTC {
		t.Fatal("RealClock.Now() is not UTC")
	}
}
