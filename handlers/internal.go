package handlers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strings"
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

func randomOTP() (int64, error) {
	max := big.NewInt(1_000_000)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}

	return value.Int64(), nil
}
