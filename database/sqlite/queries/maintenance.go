package queries

import (
	"context"
	"errors"
)

func (q *SQLiteQuerier) DeleteExpiredRecords(ctx context.Context, now int64) error {
	// Independent batches release locks between tables and still run if one fails.
	return errors.Join(
		q.queries.DeleteExpiredLoginRequests(ctx, now),
		q.queries.DeleteExpiredEmailChangeRequests(ctx, now),
		q.queries.DeleteExpiredRefreshTokens(ctx, now),
		q.queries.DeleteExpiredRateLimits(ctx, now),
	)
}
