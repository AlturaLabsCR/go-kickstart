// Package database defines the application's database interfaces.
package database

import "context"

type AccountLoginRequest struct {
	Email     string
	Otp       string
	ExpiresAt int64
}

type Database interface {
	Querier() Querier
	WithTx(ctx context.Context, fn func(q Querier) error) (err error)
	Exec(ctx context.Context, sql string) (err error)
	IsErrNotFound(err error) bool
	Close(ctx context.Context) (err error)
}

type Querier interface {
	// OncesertAccountByEmail creates an account by email if not yet exists.
	// It returns the email's subject.
	OncesertAccountByEmail(ctx context.Context, email string, createdAt int64) (sub int64, err error)

	// UpdateAccountEmail updates the email for the account subject.
	UpdateAccountEmail(ctx context.Context, sub int64, email string) error

	// DeleteAccount deletes the account for the account subject.
	DeleteAccount(ctx context.Context, sub int64) error

	// SelectAccountBySub returns the account for the account subject.
	SelectAccountBySub(ctx context.Context, sub int64) (*Account, error)

	// AssignRoleToAccount assigns a role key to an account.
	AssignRoleToAccount(ctx context.Context, sub int64, roleKey string) error

	// SelectAccountRolesBySub returns role keys assigned to an account.
	SelectAccountRolesBySub(ctx context.Context, sub int64) ([]string, error)

	// InsertRefreshToken stores a hashed refresh token for an account.
	InsertRefreshToken(ctx context.Context, tokenHash string, sub int64, expiresAt int64) error

	// ConsumeRefreshTokenByHash deletes and returns a refresh token record by token hash.
	ConsumeRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)

	// DeleteRefreshTokenByHash deletes a refresh token by token hash.
	DeleteRefreshTokenByHash(ctx context.Context, tokenHash string) error

	// DeleteRefreshTokensBySub deletes every refresh token for an account.
	DeleteRefreshTokensBySub(ctx context.Context, sub int64) error

	// DeleteExpiredRefreshTokens deletes every expired refresh token.
	DeleteExpiredRefreshTokens(ctx context.Context, now int64) error

	// RoleHasPermission reports whether a role includes a permission.
	RoleHasPermission(ctx context.Context, roleKey string, permissionKey string) (bool, error)

	// UpsertAccountEmailChangeRequest creates or updates the pending email change request for an account.
	UpsertAccountEmailChangeRequest(ctx context.Context, sub int64, email string, otp string, expiresAt int64) error

	// SelectAccountEmailChangeRequestBySub returns the pending email change request for an account.
	SelectAccountEmailChangeRequestBySub(ctx context.Context, sub int64) (*AccountEmailChangeRequest, error)

	// DeleteAccountEmailChangeRequest deletes the pending email change request for an account.
	DeleteAccountEmailChangeRequest(ctx context.Context, sub int64) error

	// UpsertAccountLoginRequest creates or updates the login request for an email.
	UpsertAccountLoginRequest(ctx context.Context, email string, otp string, expiresAt int64) error

	// SelectAccountLoginRequest returns the login request for an email.
	SelectAccountLoginRequest(ctx context.Context, email string) (*AccountLoginRequest, error)

	// DeleteAccountLoginRequest deletes the login request for an email.
	DeleteAccountLoginRequest(ctx context.Context, email string) error
}
