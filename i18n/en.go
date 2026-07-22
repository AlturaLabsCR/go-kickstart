package i18n

func init() {
	Locales["en"] = en
}

var en = map[string]string{
	"perm.change_email": "Change account email",
	"role.default":      "Default user",
	"root.greeting":     "Hello, world!",

	"err.account_not_found":          "Account not found.",
	"err.check_role_permission":      "Failed to check role permission.",
	"err.confirm_email_change":       "Failed to confirm account email change.",
	"err.delete_account":             "Failed to delete account.",
	"err.email_change_same":          "Account email change target matches current email.",
	"err.finalize_login":             "Failed to finalize login verification.",
	"err.generate_email_change_otp":  "Failed to generate account email change code.",
	"err.generate_login_otp":         "Failed to generate login code.",
	"err.invalid_email_change":       "Invalid account email change target.",
	"err.invalid_email_change_body":  "Invalid account email change request body.",
	"err.invalid_email_change_otp":   "Invalid account email change verification code.",
	"err.invalid_email_confirm_body": "Invalid account email confirm request body.",
	"err.invalid_login_body":         "Invalid login request body.",
	"err.invalid_login_email":        "Invalid login email.",
	"err.invalid_login_otp":          "Invalid login verification code.",
	"err.invalid_refresh_body":       "Invalid refresh request body.",
	"err.invalid_subject":            "Invalid authenticated subject.",
	"err.invalid_verify_body":        "Invalid verify request body.",
	"err.invalid_verify_email":       "Invalid verify email.",
	"err.issue_tokens":               "Failed to issue session tokens.",
	"err.missing_account_subject":    "Missing authenticated account subject.",
	"err.missing_bearer":             "Missing bearer token.",
	"err.missing_email_change":       "Missing account email change request.",
	"err.missing_identity":           "Missing authenticated identity.",
	"err.missing_login":              "Missing login request.",
	"err.missing_refresh":            "Missing refresh token.",
	"err.permission_denied":          "Permission denied.",
	"err.refresh_session":            "Failed to refresh session.",
	"err.render_root":                "Failed to render root page.",
	"err.revoke_sessions":            "Failed to revoke account sessions.",
	"err.save_email_change":          "Failed to save account email change request.",
	"err.save_login":                 "Failed to save login request.",
	"err.select_account":             "Failed to select account.",
	"err.select_email_change":        "Failed to select account email change request.",
	"err.select_login":               "Failed to select login request.",
	"err.verify_bearer":              "Failed to verify bearer token.",
}
