package db

import (
	"context"
	"time"
)

type TransitionBookingTxParams struct {
	BookingID     int64
	TaskID        int64
	BookingStatus string
	TaskStatus    string
	// StartedAt/CompletedAt are set only by the transition that earns them;
	// nil leaves whatever is already stored (see the COALESCE in
	// UpdateBookingStatus).
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// TransitionBookingTx moves a booking and its task to their next states
// together. The task's status is derived from the booking's -- a started
// booking means an in-progress task -- so writing one without the other leaves
// the two disagreeing about the same job.
func (store *SQLStore) TransitionBookingTx(ctx context.Context, arg TransitionBookingTxParams) error {
	return store.execTx(ctx, func(q *Queries) error {
		if err := q.UpdateBookingStatus(ctx, UpdateBookingStatusParams{
			ID:          arg.BookingID,
			Status:      arg.BookingStatus,
			StartedAt:   arg.StartedAt,
			CompletedAt: arg.CompletedAt,
		}); err != nil {
			return err
		}

		return q.UpdateTaskStatus(ctx, UpdateTaskStatusParams{
			ID:     arg.TaskID,
			Status: arg.TaskStatus,
		})
	})
}
