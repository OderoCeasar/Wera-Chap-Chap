package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides all functions to execute db queries and transactions.
//
// Everything above the data layer depends on this interface rather than on
// *pgxpool.Pool, so a handler can be exercised against a fake without a
// database and the set of writes that must be atomic stays visible in one
// place: the Tx methods below.
type Store interface {
	Querier
	RegisterUserTx(ctx context.Context, arg RegisterUserTxParams) (RegisterUserTxResult, error)
	AcceptApplicationTx(ctx context.Context, arg AcceptApplicationTxParams) (AcceptApplicationTxResult, error)
	TransitionBookingTx(ctx context.Context, arg TransitionBookingTxParams) error
	UpdateTaskerProfileTx(ctx context.Context, arg UpdateTaskerProfileTxParams) (TaskerProfile, error)
	SetAvailabilityTx(ctx context.Context, arg SetAvailabilityTxParams) ([]TaskerAvailability, error)
	CreateReviewTx(ctx context.Context, arg CreateReviewTxParams) (Review, error)
}

// SQLStore provides all functions to execute SQL queries and transactions.
type SQLStore struct {
	*Queries
	connPool *pgxpool.Pool
}

// NewStore creates a new Store.
func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		Queries:  New(connPool),
		connPool: connPool,
	}
}
