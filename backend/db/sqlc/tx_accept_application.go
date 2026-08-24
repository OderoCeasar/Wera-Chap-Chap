package db

import "context"

type AcceptApplicationTxParams struct {
	Task        Task
	Application TaskApplication
}

type AcceptApplicationTxResult struct {
	Booking Booking
}

// AcceptApplicationTx is the moment a task becomes a booking. All four writes
// have to land together: accepting one application, closing the rival ones,
// moving the task out of "open" so nobody else can be accepted, and creating
// the booking itself. A partial apply here would leave a task with two
// accepted taskers or a booking nobody can see.
func (store *SQLStore) AcceptApplicationTx(ctx context.Context, arg AcceptApplicationTxParams) (AcceptApplicationTxResult, error) {
	var result AcceptApplicationTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		if err := q.UpdateApplicationStatus(ctx, UpdateApplicationStatusParams{
			ID:     arg.Application.ID,
			Status: ApplicationAccepted,
		}); err != nil {
			return err
		}

		if err := q.RejectRivalApplications(ctx, RejectRivalApplicationsParams{
			TaskID: arg.Task.ID,
			ID:     arg.Application.ID,
		}); err != nil {
			return err
		}

		if err := q.UpdateTaskStatus(ctx, UpdateTaskStatusParams{
			ID:     arg.Task.ID,
			Status: TaskMatched,
		}); err != nil {
			return err
		}

		booking, err := q.CreateBooking(ctx, CreateBookingParams{
			TaskID:     arg.Task.ID,
			ClientID:   arg.Task.ClientID,
			TaskerID:   arg.Application.TaskerID,
			AgreedRate: arg.Application.ProposedRate,
		})
		if err != nil {
			return err
		}
		result.Booking = booking
		return nil
	})

	return result, err
}
