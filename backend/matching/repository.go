package matching

import (
	"context"

	db "wera-chap-chap/backend/db/sqlc"
)

// Repository loads the tasker data the scorer needs.
//
// It exists to keep the three-query fan-out in one place: the old GORM version
// leaned on Preload, which issued a query per association per tasker. Here the
// profiles come back with their users joined, and skills and availability are
// fetched once for the whole candidate set and indexed by tasker id.
type Repository struct {
	store db.Store
}

func NewRepository(store db.Store) *Repository {
	return &Repository{store: store}
}

// Candidate is one available tasker with everything the scorer reads.
type Candidate struct {
	Profile      db.TaskerProfile
	User         db.User
	Skills       []db.ListTaskerSkillsByTaskerIDsRow
	Availability []db.TaskerAvailability
}

// LoadCandidates returns every available tasker, hydrated.
func (r *Repository) LoadCandidates(ctx context.Context) ([]Candidate, error) {
	availableOnly := true
	rows, err := r.store.ListTaskerProfiles(ctx, db.ListTaskerProfilesParams{
		AvailableOnly: &availableOnly,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []Candidate{}, nil
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TaskerProfile.ID)
	}

	skills, err := r.store.ListTaskerSkillsByTaskerIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	skillsByTasker := make(map[int64][]db.ListTaskerSkillsByTaskerIDsRow, len(rows))
	for _, skill := range skills {
		taskerID := skill.TaskerSkill.TaskerID
		skillsByTasker[taskerID] = append(skillsByTasker[taskerID], skill)
	}

	slots, err := r.store.ListTaskerAvailabilityByTaskerIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	slotsByTasker := make(map[int64][]db.TaskerAvailability, len(rows))
	for _, slot := range slots {
		slotsByTasker[slot.TaskerID] = append(slotsByTasker[slot.TaskerID], slot)
	}

	candidates := make([]Candidate, 0, len(rows))
	for _, row := range rows {
		id := row.TaskerProfile.ID
		candidates = append(candidates, Candidate{
			Profile:      row.TaskerProfile,
			User:         row.User,
			Skills:       skillsByTasker[id],
			Availability: slotsByTasker[id],
		})
	}
	return candidates, nil
}
