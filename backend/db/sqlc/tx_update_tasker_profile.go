package db

import "context"

type UpdateTaskerProfileTxParams struct {
	UpdateTaskerProfileParams
	// SkillIDs nil means "not supplied, leave the skills alone". A non-nil
	// empty slice means "clear them" -- the distinction is what lets the
	// dashboard save a bio without wiping the tasker's categories.
	SkillIDs []int64
}

// UpdateTaskerProfileTx saves the profile fields and, when the caller supplied
// them, replaces the skill set. Replacement is delete-then-insert, so it has to
// be atomic: a failure between the two would leave the tasker with no skills
// and therefore invisible to the category filter and the matching service.
func (store *SQLStore) UpdateTaskerProfileTx(ctx context.Context, arg UpdateTaskerProfileTxParams) (TaskerProfile, error) {
	var profile TaskerProfile

	err := store.execTx(ctx, func(q *Queries) error {
		updated, err := q.UpdateTaskerProfile(ctx, arg.UpdateTaskerProfileParams)
		if err != nil {
			return err
		}
		profile = updated

		if arg.SkillIDs == nil {
			return nil
		}

		if err := q.DeleteTaskerSkills(ctx, profile.ID); err != nil {
			return err
		}

		seen := make(map[int64]bool, len(arg.SkillIDs))
		for _, categoryID := range arg.SkillIDs {
			if seen[categoryID] {
				continue
			}
			seen[categoryID] = true

			// CreateTaskerSkill is ON CONFLICT DO NOTHING and therefore :one
			// over a possibly-empty result; a duplicate the map already
			// filtered out is not an error worth failing the save for.
			if _, err := q.CreateTaskerSkill(ctx, CreateTaskerSkillParams{
				TaskerID:   profile.ID,
				CategoryID: categoryID,
			}); err != nil && !IsNotFound(err) {
				return err
			}
		}
		return nil
	})

	return profile, err
}
