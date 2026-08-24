package matching

import (
	"context"
	"math"
	"sort"

	db "wera-chap-chap/backend/db/sqlc"
)

// Result is one scored tasker. The individual factors travel alongside the
// total because the client shows them: a match is only persuasive if it can
// say why it ranked where it did.
type Result struct {
	Candidate         Candidate
	Score             float64
	SkillMatch        float64
	AvailabilityMatch float64
	DistanceScore     float64
	RatingScore       float64
	PriceMatch        float64
	DistanceKM        float64
}

// Service ranks taskers against a task.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Weights the factors carry in the total. They sum to 1, so a perfect match on
// everything scores exactly 1.0 and the client can render the total as a
// percentage without rescaling.
const (
	weightSkill        = 0.35
	weightAvailability = 0.25
	weightDistance     = 0.20
	weightRating       = 0.15
	weightPrice        = 0.05
)

// noSkillPenalty is applied to a tasker who does not list the task's category.
// They are not excluded -- a handyman may well manage an adjacent job, and in a
// thin marketplace some candidate beats none -- but they rank below anyone who
// does list it.
const noSkillPenalty = 0.45

// RankedTaskers scores every available tasker against the task, best first.
func (s *Service) RankedTaskers(ctx context.Context, task db.Task, limit int) ([]Result, error) {
	candidates, err := s.repo.LoadCandidates(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(candidates))
	for _, candidate := range candidates {
		skill := boolScore(hasSkill(candidate, task.CategoryID))
		availability := boolScore(isAvailable(candidate, task.ScheduledAt))
		distance := distanceScore(candidate.Profile.ServiceRadiusKm)
		rating := math.Min(candidate.Profile.AvgRating/5, 1)
		price := priceScore(candidate.Profile.HourlyRate, task.BudgetAmount)

		score := skill*weightSkill +
			availability*weightAvailability +
			distance*weightDistance +
			rating*weightRating +
			price*weightPrice
		if skill == 0 {
			score *= noSkillPenalty
		}

		results = append(results, Result{
			Candidate:         candidate,
			Score:             round(score),
			SkillMatch:        skill,
			AvailabilityMatch: availability,
			DistanceScore:     distance,
			RatingScore:       rating,
			PriceMatch:        price,
			DistanceKM:        candidate.Profile.ServiceRadiusKm,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}
