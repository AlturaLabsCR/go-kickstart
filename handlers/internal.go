package handlers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	appauth "github.com/myrepo/myserver/auth"
	"github.com/myrepo/myserver/middleware"
)

func decodeJSON(body io.Reader, dst any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain a single JSON value")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

var validEmail = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9+-]*[a-z0-9]|\.[a-z0-9+-]*[a-z0-9])*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$`)

// normalizeEmail returns normalized email address if it meets strict rules.
// Returns an empty string for invalid input.
func normalizeEmail(original string) string {
	email := strings.ToLower(strings.TrimSpace(original))

	local, domain, ok := strings.Cut(email, "@")
	if !ok {
		return ""
	}

	if len(local) > 30 {
		return ""
	}

	if len(domain) > 24 {
		return ""
	}

	if !validEmail.MatchString(email) {
		return ""
	}

	return email
}

func authenticatedAccountClaimsAndSubject(r *http.Request) (*appauth.Claims, int64, int, bool) {
	identity, ok := middleware.AuthenticatedClaims(r.Context())
	if !ok {
		return nil, 0, http.StatusInternalServerError, false
	}

	if identity == nil || identity.Sub == "" {
		return nil, 0, http.StatusUnauthorized, false
	}

	sub, err := strconv.ParseInt(identity.Sub, 10, 64)
	if err != nil || sub <= 0 {
		return nil, 0, http.StatusUnauthorized, false
	}

	return identity, sub, http.StatusOK, true
}

func randomOTP() (int64, error) {
	max := big.NewInt(1_000_000)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}

	return value.Int64(), nil
}
