package database

type Account struct {
	Sub       int64
	Email     string
	CreatedAt int64
}

type AccountEmailChangeRequest struct {
	Sub       int64
	Email     string
	Otp       string
	ExpiresAt int64
}

type RefreshToken struct {
	TokenHash string
	Sub       int64
	ExpiresAt int64
	CreatedAt int64
}
