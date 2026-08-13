package review_test

import (
	"math"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/review"
)

func TestApplyRatingExactTable(t *testing.T) {
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name                string
		in                  review.ReviewItemState
		rating              domain.Rating
		stage, reps, lapses int
		interval, ease      float64
		due                 time.Time
	}{
		{"again", review.ReviewItemState{Stage: 3, IntervalDays: 10, EaseFactor: 1.4, Reps: 8, Lapses: 2}, domain.RatingAgain, 0, 0, 3, 0, 1.3, now.Add(10 * time.Minute)},
		{"hard-new", review.ReviewItemState{EaseFactor: 2.5}, domain.RatingHard, 1, 1, 0, .5, 2.35, now.Add(12 * time.Hour)},
		{"hard-later", review.ReviewItemState{Stage: 2, IntervalDays: 10, EaseFactor: 1.4, Reps: 2}, domain.RatingHard, 2, 3, 0, 12, 1.3, now.Add(12 * 24 * time.Hour)},
		{"good-new", review.ReviewItemState{EaseFactor: 2.5}, domain.RatingGood, 1, 1, 0, 1, 2.5, now.Add(24 * time.Hour)},
		{"good-stage-one", review.ReviewItemState{Stage: 1, IntervalDays: 1, EaseFactor: 2.5, Reps: 1}, domain.RatingGood, 2, 2, 0, 3, 2.5, now.Add(72 * time.Hour)},
		{"good-later", review.ReviewItemState{Stage: 2, IntervalDays: 4, EaseFactor: 2.5, Reps: 2}, domain.RatingGood, 2, 3, 0, 10, 2.5, now.Add(10 * 24 * time.Hour)},
		{"easy-new", review.ReviewItemState{EaseFactor: 2.5}, domain.RatingEasy, 2, 1, 0, 4, 2.65, now.Add(4 * 24 * time.Hour)},
		{"easy-later", review.ReviewItemState{Stage: 2, IntervalDays: 4, EaseFactor: 2.5, Reps: 2}, domain.RatingEasy, 2, 3, 0, 13.78, 2.65, now.Add(time.Duration(13.78 * 24 * float64(time.Hour)))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := review.ApplyRating(tt.in, tt.rating, now)
			if got.Stage != tt.stage || got.Reps != tt.reps || got.Lapses != tt.lapses {
				t.Errorf("counters = (stage %d, reps %d, lapses %d), want (%d, %d, %d)", got.Stage, got.Reps, got.Lapses, tt.stage, tt.reps, tt.lapses)
			}
			if math.Abs(got.IntervalDays-tt.interval) > 1e-9 {
				t.Errorf("IntervalDays = %v, want %v", got.IntervalDays, tt.interval)
			}
			if math.Abs(got.EaseFactor-tt.ease) > 1e-9 {
				t.Errorf("EaseFactor = %v, want %v", got.EaseFactor, tt.ease)
			}
			if !got.DueAt.Equal(tt.due) {
				t.Errorf("DueAt = %v, want %v", got.DueAt, tt.due)
			}
		})
	}
}
