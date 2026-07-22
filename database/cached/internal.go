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
		_ = store.Delete(context.Background(), key)
		return zero, false, nil
	}

	return value, true, nil
}

func boolBytes(value bool) []byte {
	if value {
		return []byte{1}
	}

	return []byte{0}
}

func parseBoolBytes(data []byte) (bool, bool) {
	if len(data) != 1 {
		return false, false
	}

	switch data[0] {
	case 0:
		return false, true
	case 1:
		return true, true
	default:
		return false, false
	}
}
