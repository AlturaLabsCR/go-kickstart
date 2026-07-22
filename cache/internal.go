package cache

type errStr string

func (e errStr) Error() string {
	return string(e)
}
