package handlers

import (
	"net/http"
	"strconv"
	"time"
)

// Budgets are independent of challenge rows: reissuing or consuming an OTP
// must not reset the verification limit. Database failures deny the request.
func (h *Handler) allowOTPRequest(w http.ResponseWriter, r *http.Request, scope, key string, limit int64, window time.Duration) bool {
	retryAfter, err := h.db.Querier().TakeRateLimit(r.Context(), scope, key, time.Now().Unix(), int64(window/time.Second), limit)
	if err != nil {
		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.check_otp_limit",
			"failed to check OTP rate limit",
			"scope",
			scope,
		)
		return false
	}
	if retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
		h.writeStatus(
			w,
			r,
			http.StatusTooManyRequests,
			"err.otp_rate_limited",
			"OTP rate limit exceeded",
			"scope",
			scope,
		)
		return false
	}
	return true
}
