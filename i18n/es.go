package i18n

func init() {
	Locales["es"] = es
}

var es = map[string]string{
	"perm.change_email": "Cambiar correo de cuenta",
	"role.default":      "Usuario predeterminado",
	"root.greeting":     "Hola, mundo!",

	"err.account_not_found":          "Cuenta no encontrada.",
	"err.check_role_permission":      "No se pudo verificar el permiso del rol.",
	"err.confirm_email_change":       "No se pudo confirmar el cambio de correo.",
	"err.delete_account":             "No se pudo eliminar la cuenta.",
	"err.email_change_same":          "El correo nuevo coincide con el correo actual.",
	"err.finalize_login":             "No se pudo finalizar la verificacion de inicio de sesion.",
	"err.generate_email_change_otp":  "No se pudo generar el codigo de cambio de correo.",
	"err.generate_login_otp":         "No se pudo generar el codigo de inicio de sesion.",
	"err.invalid_email_change":       "Correo nuevo invalido.",
	"err.invalid_email_change_body":  "Cuerpo de solicitud de cambio de correo invalido.",
	"err.invalid_email_change_otp":   "Codigo de verificacion de cambio de correo invalido.",
	"err.invalid_email_confirm_body": "Cuerpo de confirmacion de cambio de correo invalido.",
	"err.invalid_login_body":         "Cuerpo de inicio de sesion invalido.",
	"err.invalid_login_email":        "Correo de inicio de sesion invalido.",
	"err.invalid_login_otp":          "Codigo de verificacion de inicio de sesion invalido.",
	"err.invalid_refresh_body":       "Cuerpo de renovacion de sesion invalido.",
	"err.invalid_subject":            "Sujeto autenticado invalido.",
	"err.invalid_verify_body":        "Cuerpo de verificacion invalido.",
	"err.invalid_verify_email":       "Correo de verificacion invalido.",
	"err.issue_tokens":               "No se pudieron emitir los tokens de sesion.",
	"err.missing_account_subject":    "Falta el sujeto de la cuenta autenticada.",
	"err.missing_bearer":             "Falta el token bearer.",
	"err.missing_email_change":       "Falta la solicitud de cambio de correo.",
	"err.missing_identity":           "Falta la identidad autenticada.",
	"err.missing_login":              "Falta la solicitud de inicio de sesion.",
	"err.missing_refresh":            "Falta el token de renovacion.",
	"err.permission_denied":          "Permiso denegado.",
	"err.refresh_session":            "No se pudo renovar la sesion.",
	"err.render_root":                "No se pudo renderizar la pagina principal.",
	"err.revoke_sessions":            "No se pudieron revocar las sesiones de la cuenta.",
	"err.save_email_change":          "No se pudo guardar la solicitud de cambio de correo.",
	"err.save_login":                 "No se pudo guardar la solicitud de inicio de sesion.",
	"err.select_account":             "No se pudo consultar la cuenta.",
	"err.select_email_change":        "No se pudo consultar la solicitud de cambio de correo.",
	"err.select_login":               "No se pudo consultar la solicitud de inicio de sesion.",
	"err.verify_bearer":              "No se pudo verificar el token bearer.",
}
