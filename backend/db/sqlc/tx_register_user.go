package db

import "context"

type RegisterUserTxParams struct {
	CreateUserParams
	// AfterCreate runs inside the transaction, once the row exists but before
	// the commit -- a tasker whose profile insert fails must not be left as an
	// account that can log in but has nothing to work from.
	AfterCreate func(user User, q *Queries) error
}

type RegisterUserTxResult struct {
	User User
	// TaskerProfile is nil for a client, who has none.
	TaskerProfile *TaskerProfile
}

// RegisterUserTx creates the account and, for a tasker, the empty profile that
// every tasker-side query assumes exists.
func (store *SQLStore) RegisterUserTx(ctx context.Context, arg RegisterUserTxParams) (RegisterUserTxResult, error) {
	var result RegisterUserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		user, err := q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			return err
		}
		result.User = user

		if user.Role == RoleTasker {
			profile, err := q.CreateTaskerProfile(ctx, CreateTaskerProfileParams{
				UserID: user.ID,
				// Defaults a new tasker can work from before they have filled
				// anything in: reachable in the directory, with the same
				// service radius the schema defaults to.
				Bio:             "",
				HourlyRate:      0,
				YearsExperience: 0,
				ServiceRadiusKm: DefaultServiceRadiusKm,
				IsAvailable:     true,
			})
			if err != nil {
				return err
			}
			result.TaskerProfile = &profile
		}

		if arg.AfterCreate != nil {
			return arg.AfterCreate(user, q)
		}
		return nil
	})

	return result, err
}
