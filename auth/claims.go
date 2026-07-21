package auth

type Claims struct {
	Sub   string   `json:"sub"`
	Roles []string `json:"roles,omitempty"`
	Exp   int64    `json:"exp,omitempty"`
}

func (c *Claims) ExpiresAt() int64 {
	if c == nil {
		return 0
	}

	return c.Exp
}

func (c *Claims) SetExpiresAt(exp int64) {
	if c == nil {
		return
	}

	c.Exp = exp
}

func (c *Claims) Subject() string {
	if c == nil {
		return ""
	}

	return c.Sub
}
