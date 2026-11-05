package queries

import (
	"context"
	"errors"

	"app/database/mysql/db"
)

func (q *MysqlQuerier) DeleteExpiredRecords(ctx context.Context, now int64) error {
	// Independent batches release locks between tables and still run if one fails.
	return errors.Join(
		q.queries.DeleteExpiredLoginRequests(ctx, db.DeleteExpiredLoginRequestsParams{Now: now}),
		q.queries.DeleteExpiredEmailChangeRequests(ctx, db.DeleteExpiredEmailChangeRequestsParams{Now: now}),
		q.queries.DeleteExpiredRefreshTokens(ctx, db.DeleteExpiredRefreshTokensParams{Now: now}),
		q.queries.DeleteExpiredRateLimits(ctx, db.DeleteExpiredRateLimitsParams{Now: now}),
	)
}
