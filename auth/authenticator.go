package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"app/database"
	goauth "github.com/tavocg/go-auth"
	"github.com/tavocg/go-auth/jwt"
	secrets "github.com/tavocg/go-secrets"
)

type Authenticator struct {
	db         database.Database
	tokener    *jwt.Tokener[*Claims]
	refreshTTL time.Duration
}

func NewAuthenticator(db database.Database, secret string, accessTTL time.Duration, refreshTTL time.Duration) (*Authenticator, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}

	tokener, err := jwt.NewHS256Tokener[*Claims](secret, accessTTL)
	if err != nil {
		return nil, err
	}

	return &Authenticator{
		db:         db,
		tokener:    tokener,
		refreshTTL: refreshTTL,
	}, nil
}

func (a *Authenticator) Issue(ctx context.Context, identity *Claims) (at *goauth.Token, rt *goauth.Token, err error) {
	sub, err := subject(identity)
	if err != nil {
		return nil, nil, err
	}

	return a.issueForSub(ctx, a.db.Querier(), sub)
}

func (a *Authenticator) Refresh(ctx context.Context, refreshToken string) (at *goauth.Token, rt *goauth.Token, err error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, nil, goauth.ErrInvalidToken
	}

	tokenHash := hashRefreshToken(refreshToken)
	err = a.db.WithTx(ctx, func(q database.Querier) error {
		saved, err := q.ConsumeRefreshTokenByHash(ctx, tokenHash)
		if err != nil {
			if a.db.IsErrNotFound(err) {
				return goauth.ErrInvalidToken
			}
			return err
		}

		if saved.ExpiresAt != 0 && time.Now().UTC().Unix() >= saved.ExpiresAt {
			return goauth.ErrExpiredToken
		}

		at, rt, err = a.issueForSub(ctx, q, saved.Sub)
		return err
	})
	if err != nil {
		return nil, nil, err
	}

	return at, rt, nil
}

func (a *Authenticator) Verify(_ context.Context, accessToken string) (*Claims, error) {
	claims, err := a.tokener.Verify(accessToken)
	if err != nil {
		switch err {
		case jwt.ErrInvalidToken:
			return nil, goauth.ErrInvalidToken
		case jwt.ErrExpiredToken:
			return nil, goauth.ErrExpiredToken
		default:
			return nil, err
		}
	}
	if claims == nil || claims.Subject() == "" {
		return nil, goauth.ErrInvalidToken
	}

	return claims, nil
}

func (a *Authenticator) Revoke(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return goauth.ErrInvalidToken
	}

	return a.db.Querier().DeleteRefreshTokenByHash(ctx, hashRefreshToken(refreshToken))
}

func (a *Authenticator) RevokeAll(ctx context.Context, identity *Claims) error {
	sub, err := subject(identity)
	if err != nil {
		return err
	}

	return a.db.Querier().DeleteRefreshTokensBySub(ctx, sub)
}

func (a *Authenticator) CleanExpiredRefreshTokens(ctx context.Context) error {
	return a.db.Querier().DeleteExpiredRefreshTokens(ctx, time.Now().UTC().Unix())
}

func (a *Authenticator) issueForSub(ctx context.Context, q database.Querier, sub int64) (*goauth.Token, *goauth.Token, error) {
	roles, err := q.SelectAccountRolesBySub(ctx, sub)
	if err != nil {
		return nil, nil, err
	}

	claims := &Claims{
		Sub:   strconv.FormatInt(sub, 10),
		Roles: roles,
	}
	accessToken, err := a.tokener.Sign(claims)
	if err != nil {
		return nil, nil, err
	}

	refreshToken, err := secrets.RandStr()
	if err != nil {
		return nil, nil, err
	}

	var refreshExpiresAt int64
	if a.refreshTTL > 0 {
		refreshExpiresAt = time.Now().UTC().Add(a.refreshTTL).Unix()
	}

	if err := q.InsertRefreshToken(ctx, hashRefreshToken(refreshToken), sub, refreshExpiresAt); err != nil {
		return nil, nil, err
	}

	return &goauth.Token{
			Value:     accessToken,
			ExpiresAt: claims.ExpiresAt(),
		}, &goauth.Token{
			Value:     refreshToken,
			ExpiresAt: refreshExpiresAt,
		}, nil
}

func subject(identity *Claims) (int64, error) {
	if identity == nil || strings.TrimSpace(identity.Subject()) == "" {
		return 0, goauth.ErrInvalidIdentity
	}

	sub, err := strconv.ParseInt(identity.Subject(), 10, 64)
	if err != nil || sub <= 0 {
		return 0, goauth.ErrInvalidIdentity
	}

	return sub, nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
