package services

import (
	"math"
	"sort"
	"time"

	"gorm.io/gorm"

	"wera-chap-chap/backend/models"
)

type MatchResult struct {
	Tasker            models.TaskerProfile `json:"tasker"`
	Score             float64              `json:"score"`
	SkillMatch        float64              `json:"skill_match"`
	AvailabilityMatch float64              `json:"availability_match"`
	DistanceScore     float64              `json:"distance_score"`
	RatingScore       float64              `json:"rating_score"`
	PriceMatch        float64              `json:"price_match"`
	DistanceKM        float64              `json:"distance_km"`
}

func RankedTaskers(db *gorm.DB, task models.Task, limit int) ([]MatchResult, error) {
	var taskers []models.TaskerProfile
	if err := db.Preload("User").Preload("Skills.Category").Preload("Availability").Where("is_available = ?", true).Find(&taskers).Error; err != nil {
		return nil, err
	}
	results := make([]MatchResult, 0, len(taskers))
	for _, tasker := range taskers {
		skill := boolScore(hasSkill(tasker, task.CategoryID))
		availability := boolScore(isAvailable(tasker, task.ScheduledAt))
		distance := distanceScore(tasker.ServiceRadiusKM)
		rating := math.Min(tasker.AvgRating/5, 1)
		price := priceScore(tasker.HourlyRate, task.BudgetAmount)
		score := skill*0.35 + availability*0.25 + distance*0.20 + rating*0.15 + price*0.05
		if skill == 0 {
			score *= 0.45
		}
		results = append(results, MatchResult{
			Tasker: tasker, Score: round(score), SkillMatch: skill, AvailabilityMatch: availability,
			DistanceScore: distance, RatingScore: rating, PriceMatch: price, DistanceKM: tasker.ServiceRadiusKM,
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func hasSkill(tasker models.TaskerProfile, categoryID uint) bool {
	for _, skill := range tasker.Skills {
		if skill.CategoryID == categoryID {
			return true
		}
	}
	return false
}

func isAvailable(tasker models.TaskerProfile, scheduledAt time.Time) bool {
	if scheduledAt.IsZero() {
		return true
	}
	day := int(scheduledAt.Weekday())
	hhmm := scheduledAt.Format("15:04")
	for _, slot := range tasker.Availability {
		if slot.DayOfWeek == day && slot.StartTime <= hhmm && slot.EndTime >= hhmm {
			return true
		}
	}
	return false
}

func distanceScore(serviceRadius float64) float64 {
	if serviceRadius <= 0 {
		return 0.5
	}
	return math.Min(serviceRadius/25, 1)
}

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
