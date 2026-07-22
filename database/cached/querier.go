package cached

import (
	"context"

	"app/cache"
	"app/database"
)

type Querier struct {
	next      database.Querier
	store     cache.Store
	cacheRead bool
}

var _ database.Querier = (*Querier)(nil)

func (q *Querier) OncesertAccountByEmail(ctx context.Context, email string, createdAt int64) (int64, error) {
	return q.next.OncesertAccountByEmail(ctx, email, createdAt)
}

func (q *Querier) UpdateAccountEmail(ctx context.Context, sub int64, email string) error {
	if err := q.next.UpdateAccountEmail(ctx, sub, email); err != nil {
		return err
	}

	_ = q.store.Delete(context.Background(), accountKey(sub))
	return nil
}

func (q *Querier) DeleteAccount(ctx context.Context, sub int64) error {
	if err := q.next.DeleteAccount(ctx, sub); err != nil {
		return err
	}

	_ = q.store.Delete(context.Background(), accountKey(sub))
	_ = q.store.Delete(context.Background(), accountRolesKey(sub))
	return nil
}

func (q *Querier) SelectAccountBySub(ctx context.Context, sub int64) (*database.Account, error) {
	if !q.cacheRead {
		return q.next.SelectAccountBySub(ctx, sub)
	}

	key := accountKey(sub)
	account, ok, err := getJSON[database.Account](ctx, q.store, key)
	if err != nil {
		return nil, err
	}
	if ok {
		return &account, nil
	}

	selected, err := q.next.SelectAccountBySub(ctx, sub)
	if err != nil {
		return nil, err
	}

	setJSON(ctx, q.store, key, selected)
	return selected, nil
}

func (q *Querier) AssignRoleToAccount(ctx context.Context, sub int64, roleKey string) error {
	if err := q.next.AssignRoleToAccount(ctx, sub, roleKey); err != nil {
		return err
	}

	_ = q.store.Delete(context.Background(), accountRolesKey(sub))
	return nil
}

func (q *Querier) SelectAccountRolesBySub(ctx context.Context, sub int64) ([]string, error) {
	if !q.cacheRead {
		return q.next.SelectAccountRolesBySub(ctx, sub)
	}

	key := accountRolesKey(sub)
	roles, ok, err := getJSON[[]string](ctx, q.store, key)
	if err != nil {
		return nil, err
	}
	if ok {
		return roles, nil
	}

	roles, err = q.next.SelectAccountRolesBySub(ctx, sub)
	if err != nil {
		return nil, err
	}

	setJSON(ctx, q.store, key, roles)
	return roles, nil
}

func (q *Querier) InsertRefreshToken(ctx context.Context, tokenHash string, sub int64, expiresAt int64) error {
	return q.next.InsertRefreshToken(ctx, tokenHash, sub, expiresAt)
}

func (q *Querier) ConsumeRefreshTokenByHash(ctx context.Context, tokenHash string) (*database.RefreshToken, error) {
	return q.next.ConsumeRefreshTokenByHash(ctx, tokenHash)
}

func (q *Querier) DeleteRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	return q.next.DeleteRefreshTokenByHash(ctx, tokenHash)
}

func (q *Querier) DeleteRefreshTokensBySub(ctx context.Context, sub int64) error {
	return q.next.DeleteRefreshTokensBySub(ctx, sub)
}

func (q *Querier) DeleteExpiredRefreshTokens(ctx context.Context, now int64) error {
	return q.next.DeleteExpiredRefreshTokens(ctx, now)
}

func (q *Querier) RoleHasPermission(ctx context.Context, roleKey string, permissionKey string) (bool, error) {
	if !q.cacheRead {
		return q.next.RoleHasPermission(ctx, roleKey, permissionKey)
	}

	key := rolePermissionKey(roleKey, permissionKey)
	data, err := q.store.Get(ctx, key)
	if err == nil {
		if allowed, ok := parseBoolBytes(data); ok {
			return allowed, nil
		}

		_ = q.store.Delete(context.Background(), key)
	} else if !cacheMiss(err) {
		return false, err
	}

	allowed, err := q.next.RoleHasPermission(ctx, roleKey, permissionKey)
	if err != nil {
		return false, err
	}

	_ = q.store.Set(ctx, key, boolBytes(allowed))
	return allowed, nil
}

func (q *Querier) UpsertAccountEmailChangeRequest(ctx context.Context, sub int64, email string, otp string, expiresAt int64) error {
	return q.next.UpsertAccountEmailChangeRequest(ctx, sub, email, otp, expiresAt)
}

func (q *Querier) SelectAccountEmailChangeRequestBySub(ctx context.Context, sub int64) (*database.AccountEmailChangeRequest, error) {
	return q.next.SelectAccountEmailChangeRequestBySub(ctx, sub)
}

func (q *Querier) DeleteAccountEmailChangeRequest(ctx context.Context, sub int64) error {
	return q.next.DeleteAccountEmailChangeRequest(ctx, sub)
}

func (q *Querier) UpsertAccountLoginRequest(ctx context.Context, email string, otp string, expiresAt int64) error {
	return q.next.UpsertAccountLoginRequest(ctx, email, otp, expiresAt)
}

func (q *Querier) SelectAccountLoginRequest(ctx context.Context, email string) (*database.AccountLoginRequest, error) {
	return q.next.SelectAccountLoginRequest(ctx, email)
}

func (q *Querier) DeleteAccountLoginRequest(ctx context.Context, email string) error {
	return q.next.DeleteAccountLoginRequest(ctx, email)
}
