package db

import "context"

type AvailabilitySlot struct {
	DayOfWeek int32
	StartTime string
	EndTime   string
}

type SetAvailabilityTxParams struct {
	TaskerID int64
	Slots    []AvailabilitySlot
}

// SetAvailabilityTx replaces a tasker's whole weekly schedule. Same reasoning
// as the skill set: delete-then-insert has to commit as one unit, or a tasker
// who half-saved a schedule reads as available at no time at all, which the
// matching service scores as a hard miss.
func (store *SQLStore) SetAvailabilityTx(ctx context.Context, arg SetAvailabilityTxParams) ([]TaskerAvailability, error) {
	saved := []TaskerAvailability{}

	err := store.execTx(ctx, func(q *Queries) error {
		if err := q.DeleteTaskerAvailability(ctx, arg.TaskerID); err != nil {
			return err
		}

		for _, slot := range arg.Slots {
			created, err := q.CreateTaskerAvailability(ctx, CreateTaskerAvailabilityParams{
				TaskerID:  arg.TaskerID,
				DayOfWeek: slot.DayOfWeek,
				StartTime: slot.StartTime,
				EndTime:   slot.EndTime,
			})
			if err != nil {
				return err
			}
			saved = append(saved, created)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return saved, nil
}
