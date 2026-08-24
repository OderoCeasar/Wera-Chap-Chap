package db

import "context"

type CreateReviewTxParams struct {
	CreateReviewParams
}

// CreateReviewTx stores the review and refreshes the reviewee's denormalised
// rating in the same transaction. Recalculating outside it -- as the GORM
// version did -- leaves avg_rating stale whenever the second statement fails,
// and stale is invisible: nothing later recomputes it.
//
// The rating update is keyed on user_id and quietly matches nothing when the
// reviewee is a client, who has no tasker profile.
func (store *SQLStore) CreateReviewTx(ctx context.Context, arg CreateReviewTxParams) (Review, error) {
	var review Review

	err := store.execTx(ctx, func(q *Queries) error {
		created, err := q.CreateReview(ctx, arg.CreateReviewParams)
		if err != nil {
			return err
		}
		review = created

		stats, err := q.GetRevieweeRatingStats(ctx, arg.RevieweeID)
		if err != nil {
			return err
		}

		return q.UpdateTaskerRatingByUserID(ctx, UpdateTaskerRatingByUserIDParams{
			UserID:       arg.RevieweeID,
			AvgRating:    stats.Average,
			TotalReviews: stats.Total,
		})
	})

	return review, err
}
