package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	goauth "github.com/tavocg/go-auth"
	"github.com/tavocg/go-auth/jwt"
)

type Claims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp,omitempty"`
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

type Authenticator struct {
	tokener    *jwt.Tokener[*Claims]
	refreshTTL time.Duration
	store      sync.Map
}

type refreshRecord struct {
	Subject   string
	ExpiresAt int64
}

func NewAuthenticator(secret string, accessTTL, refreshTTL time.Duration) (*Authenticator, error) {
	access, err := jwt.NewHS256Tokener[*Claims](secret, accessTTL)
	if err != nil {
		return nil, err
	}

	return &Authenticator{
		tokener:    access,
		refreshTTL: refreshTTL,
	}, nil
}

func (a *Authenticator) Issue(_ context.Context, identity *Claims) (*goauth.Tokens, error) {
	if identity == nil || identity.Subject() == "" {
		return nil, goauth.ErrInvalidIdentity
	}

	claims := &Claims{Sub: identity.Subject()}
	accessToken, err := a.tokener.Sign(claims)
	if err != nil {
		return nil, err
	}

	refreshToken, err := randomSecret()
	if err != nil {
		return nil, err
	}

	var refreshExpiresAt int64
	if a.refreshTTL > 0 {
		refreshExpiresAt = time.Now().UTC().Add(a.refreshTTL).Unix()
	}

	a.store.Store(hashSecret(refreshToken), refreshRecord{
		Subject:   identity.Subject(),
		ExpiresAt: refreshExpiresAt,
	})

	return &goauth.Tokens{
		Access: goauth.Token{
			Value:     accessToken,
			ExpiresAt: claims.ExpiresAt(),
		},
		Refresh: goauth.Token{
			Value:     refreshToken,
			ExpiresAt: refreshExpiresAt,
		},
	}, nil
}

func (a *Authenticator) Refresh(ctx context.Context, refreshToken string) (*goauth.Tokens, error) {
	recordValue, ok := a.store.Load(hashSecret(refreshToken))
	if !ok {
		return nil, goauth.ErrRevokedToken
	}

	record, ok := recordValue.(refreshRecord)
	if !ok {
		return nil, goauth.ErrInvalidToken
	}
	if record.ExpiresAt != 0 && time.Now().UTC().Unix() >= record.ExpiresAt {
		a.store.Delete(hashSecret(refreshToken))
		return nil, goauth.ErrExpiredToken
	}

	a.store.Delete(hashSecret(refreshToken))
	return a.Issue(ctx, &Claims{Sub: record.Subject})
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

func (a *Authenticator) Revoke(_ context.Context, refreshToken string) error {
	key := hashSecret(refreshToken)
	if _, ok := a.store.Load(key); !ok {
		return goauth.ErrRevokedToken
	}

	a.store.Delete(key)
	return nil
}

func (a *Authenticator) RevokeAll(_ context.Context, identity *Claims) error {
	if identity == nil || identity.Subject() == "" {
		return goauth.ErrInvalidIdentity
	}

	a.store.Range(func(key, value any) bool {
		record, ok := value.(refreshRecord)
		if ok && record.Subject == identity.Subject() {
			a.store.Delete(key)
		}
		return true
	})

	return nil
}

func randomSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

var _ goauth.Authenticator[*Claims] = (*Authenticator)(nil)
