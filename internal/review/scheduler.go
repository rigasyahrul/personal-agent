package review

import (
	"math"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/domain"
)

type ReviewItemState struct {
	Stage        int
	IntervalDays float64
	EaseFactor   float64
	Reps         int
	Lapses       int
	DueAt        time.Time
}

func ApplyRating(item ReviewItemState, rating domain.Rating, now time.Time) ReviewItemState {
	switch rating {
	case domain.RatingAgain:
		item.Lapses++
		item.Reps = 0
		item.Stage = 0
		item.IntervalDays = 0
		item.EaseFactor = math.Max(1.3, item.EaseFactor-.2)
		item.DueAt = now.Add(10 * time.Minute)
	case domain.RatingHard:
		item.Reps++
		item.EaseFactor = math.Max(1.3, item.EaseFactor-.15)
		if item.Stage == 0 {
			item.IntervalDays = .5
		} else {
			item.IntervalDays *= 1.2
		}
		if item.Stage < 1 {
			item.Stage = 1
		}
		item.DueAt = addDays(now, item.IntervalDays)
	case domain.RatingGood:
		item.Reps++
		if item.Stage == 0 {
			item.IntervalDays = 1
			item.Stage = 1
		} else if item.Stage == 1 {
			item.IntervalDays = 3
			item.Stage = 2
		} else {
			item.IntervalDays *= item.EaseFactor
		}
		item.DueAt = addDays(now, item.IntervalDays)
	case domain.RatingEasy:
		item.Reps++
		item.EaseFactor += .15
		if item.Stage < 2 {
			item.IntervalDays = 4
			item.Stage = 2
		} else {
			item.IntervalDays = item.IntervalDays * item.EaseFactor * 1.3
		}
		item.DueAt = addDays(now, item.IntervalDays)
	default:
		panic("invalid rating")
	}
	return item
}

func addDays(t time.Time, days float64) time.Time {
	return t.Add(time.Duration(days * 24 * float64(time.Hour)))
}
