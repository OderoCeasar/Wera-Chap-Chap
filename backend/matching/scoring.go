package matching

import (
	"math"
	"time"
)

// The scoring factors. Each returns 0..1 and is a pure function of the
// candidate, which is what makes them testable without a database.

func hasSkill(candidate Candidate, categoryID int64) bool {
	for _, skill := range candidate.Skills {
		if skill.TaskerSkill.CategoryID == categoryID {
			return true
		}
	}
	return false
}

// isAvailable checks the tasker's weekly schedule against the moment the task
// is scheduled for. A task with no scheduled time counts as available: the
// client has not committed to a slot, so no tasker can be ruled out on one.
//
// The comparison is lexical on "HH:MM", which is why tasker_availability stores
// those columns as text -- it is a correct ordering for zero-padded 24-hour
// times and needs no parsing.
func isAvailable(candidate Candidate, scheduledAt *time.Time) bool {
	if scheduledAt == nil || scheduledAt.IsZero() {
		return true
	}
	day := int32(scheduledAt.Weekday())
	hhmm := scheduledAt.Format("15:04")
	for _, slot := range candidate.Availability {
		if slot.DayOfWeek == day && slot.StartTime <= hhmm && slot.EndTime >= hhmm {
			return true
		}
	}
	return false
}

// distanceScore stands in for a real distance calculation until tasker
// locations are stored: a wider service radius is treated as more likely to
// cover the job. A tasker who has not set one scores neutral rather than zero.
func distanceScore(serviceRadius float64) float64 {
	if serviceRadius <= 0 {
		return 0.5
	}
	return math.Min(serviceRadius/25, 1)
}

// priceScore rewards a rate close to the client's budget in either direction --
// far under suggests a mismatch in scope as much as far over does.
func priceScore(rate, budget float64) float64 {
	if budget <= 0 || rate <= 0 {
		return 0.5
	}
	return math.Max(0, 1-math.Abs(rate-budget)/budget)
}

func boolScore(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func round(value float64) float64 {
	return math.Round(value*100) / 100
}
