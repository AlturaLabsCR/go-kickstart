package handlers

import (
	"net/http"
	"net/netip"
	"strconv"
	"time"
)

const otpLifetime = 10 * time.Minute

func (h *Handler) limitOTPIssuance(next http.Handler) http.Handler {
	return h.limitOTP("issue", 10, 100, next)
}

func (h *Handler) limitOTPVerification(next http.Handler) http.Handler {
	return h.limitOTP("verify", 30, 300, next)
}

// Charge aggregate budgets before parsing bodies, authentication or identity
// budgets. Global admission bounds new origin/identity keys even during rotation.
func (h *Handler) limitOTP(action string, originLimit, globalLimit int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.allowOTPRequest(w, r, "otp."+action+".global", "all", globalLimit, time.Minute) ||
			!h.allowOTPRequest(w, r, "otp."+action+".origin", otpOrigin(r.RemoteAddr), originLimit, time.Minute) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func otpOrigin(remoteAddr string) string {
	peer, err := netip.ParseAddrPort(remoteAddr)
	if err != nil {
		return "unknown" // A shared budget, never an unbounded client-supplied key.
	}
	addr := peer.Addr().Unmap().WithZone("")
	if addr.Is6() {
		return netip.PrefixFrom(addr, 64).Masked().String()
	}
	return addr.String()
}

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

// Separate budgets prevent both rapid resends and sustained email flooding.
func (h *Handler) allowOTPIssuance(w http.ResponseWriter, r *http.Request, scope, key string) bool {
	return h.allowOTPRequest(w, r, scope+".issue.minute", key, 1, time.Minute) &&
		h.allowOTPRequest(w, r, scope+".issue.window", key, 5, 10*time.Minute)
}
