// Package database defines the application's database interfaces.
package database

import "context"

type Database interface {
	Querier() Querier
	WithTx(ctx context.Context, fn func(q Querier) error) error
	IsErrNotFound(err error) bool
	Close() error
}

type Querier interface {
	// TakeRateLimit atomically charges a request to a fixed window. Zero means
	// allowed; a positive result is the retry delay in seconds. For failed
	// authentication attempts to persist, call outside the business transaction.
	TakeRateLimit(ctx context.Context, scope, key string, now, windowSeconds, maxRequests int64) (int64, error)

	// DeleteExpiredRecords removes at most 1000 expired rows per OTP/token table
	// and 100 expired rate-limit windows. Active records are left untouched.
	DeleteExpiredRecords(ctx context.Context, now int64) error

	// OncesertAccountByEmail creates an account by email if not yet exists.
	// It returns the email's subject.
	OncesertAccountByEmail(ctx context.Context, email string, createdAt int64) (sub int64, err error)

	// UpdateAccountEmail updates the email for the account subject.
	// It returns ErrEmailUnavailable if another account already uses the email.
	UpdateAccountEmail(ctx context.Context, sub int64, email string) error

	// DeleteAccount deletes the account for the account subject.
	DeleteAccount(ctx context.Context, sub int64) error

	// SelectAccountAuthID returns the current account authentication identity.
	// Authorization must read this directly from the database, never from cache.
	SelectAccountAuthID(ctx context.Context, sub int64) (string, error)

	// LockAccountAuthID locks the account row and returns its current
	// authentication identity. Call it inside WithTx; session issuance,
	// refresh and revocation use this lock order before touching refresh tokens.
	LockAccountAuthID(ctx context.Context, sub int64) (string, error)

	// SetAccountAuthID replaces the account authentication identity, which
	// invalidates every access token carrying the previous value.
	SetAccountAuthID(ctx context.Context, sub int64, authID string) error

	// SelectAccountBySub returns the account for the account subject.
	SelectAccountBySub(ctx context.Context, sub int64) (*Account, error)

	// AssignRoleToAccount assigns a role key to an account.
	AssignRoleToAccount(ctx context.Context, sub int64, roleKey string) error

	// SelectAccountRolesBySub returns role keys assigned to an account.
	SelectAccountRolesBySub(ctx context.Context, sub int64) ([]string, error)

	// InsertRefreshToken stores a hashed refresh token for an account session.
	InsertRefreshToken(ctx context.Context, tokenHash string, sub int64, authID string, expiresAt int64) error

	// SelectRefreshTokenByHash returns a refresh token record without locking it.
	SelectRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)

	// ConsumeRefreshTokenByHash deletes and returns a refresh token record by token hash.
	ConsumeRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)

	// DeleteRefreshTokenByHash deletes a refresh token by token hash.
	DeleteRefreshTokenByHash(ctx context.Context, tokenHash string) error

	// DeleteRefreshTokensBySub deletes every refresh token for an account.
	DeleteRefreshTokensBySub(ctx context.Context, sub int64) error

	// RoleHasPermission reports whether a role includes a permission.
	RoleHasPermission(ctx context.Context, roleKey string, permissionKey string) (bool, error)

	// UpsertAccountEmailChangeRequest creates or updates the pending email change request for an account.
	// otpHash is a keyed digest; plaintext codes are never stored.
	UpsertAccountEmailChangeRequest(ctx context.Context, sub int64, email string, otpHash string, expiresAt int64) error

	// ConsumeAccountEmailChangeRequest atomically consumes a matching, unexpired request.
	ConsumeAccountEmailChangeRequest(ctx context.Context, sub int64, otpHash string, now int64) (*AccountEmailChangeRequest, error)

	// DeleteAccountEmailChangeRequest deletes the pending email change request for an account.
	DeleteAccountEmailChangeRequest(ctx context.Context, sub int64) error

	// UpsertAccountLoginRequest creates or updates the login request for an email.
	// otpHash is a keyed digest; plaintext codes are never stored.
	UpsertAccountLoginRequest(ctx context.Context, email string, otpHash string, expiresAt int64) error

	// ConsumeAccountLoginRequest atomically consumes a matching, unexpired request.
	ConsumeAccountLoginRequest(ctx context.Context, email string, otpHash string, now int64) (*AccountLoginRequest, error)

	// DeleteAccountLoginRequest deletes the login request for an email.
	DeleteAccountLoginRequest(ctx context.Context, email string) error
}
