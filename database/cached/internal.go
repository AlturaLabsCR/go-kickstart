package cached

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"app/cache"
)

func accountKey(sub int64) string {
	return "db:account:" + strconv.FormatInt(sub, 10)
}

func accountRolesKey(sub int64) string {
	return "db:account_roles:" + strconv.FormatInt(sub, 10)
}

func rolePermissionKey(role string, permission string) string {
	return "db:role_permission:" + role + ":" + permission
}

func cacheMiss(err error) bool {
	return errors.Is(err, cache.ErrNotFound)
}

func setJSON(ctx context.Context, store cache.Store, key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	_ = store.Set(ctx, key, data)
}

func getJSON[T any](ctx context.Context, store cache.Store, key string) (T, bool, error) {
	var zero T

	data, err := store.Get(ctx, key)
	if err != nil {
		if cacheMiss(err) {
			return zero, false, nil
		}

		return zero, false, err
	}

	var value T
	err = json.Unmarshal(data, &value)
	if err != nil {
		_ = store.Delete(ctx, key)
		return zero, false, nil
	}

	return value, true, nil
}

func readThrough[T any](ctx context.Context, q *Querier, key string, load func() (T, error)) (T, error) {
	if !q.cacheRead {
		return load()
	}

	value, ok, err := getJSON[T](ctx, q.store, key)
	if err != nil || ok {
		return value, err
	}

	value, err = load()
	if err != nil {
		return value, err
	}

	setJSON(ctx, q.store, key, value)
	return value, nil
}
