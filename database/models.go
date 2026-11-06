package database

const (
	PermissionChangeEmail = "perm.change_email"
	RoleDefault           = "role.default"
)

type Account struct {
	Sub       int64
	Email     string
	CreatedAt int64
}

type AccountLoginRequest struct {
	Email     string
	OTPHash   string
	ExpiresAt int64
}

type AccountEmailChangeRequest struct {
	Sub       int64
	Email     string
	OTPHash   string
	ExpiresAt int64
}

type RefreshToken struct {
	TokenHash string
	Sub       int64
	AuthID    string
	ExpiresAt int64
	CreatedAt int64
}
