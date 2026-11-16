package handlers

import (
	"net/http"

	"app/templates"
)

// The generated hash authorizes only our inline bundle; other scripts stay same-origin.
const documentCSP = "default-src 'self'; script-src 'self' '" + templates.MainScriptCSPHash + "'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; object-src 'none'; base-uri 'none'; form-action 'self'; frame-src 'self'; frame-ancestors 'self'"

// Apply only to our rendered pages; Redocly has separate resource requirements.
func documentHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", documentCSP)
	// Preserve same-origin form Origin on HTTP LAN hosts without exposing queries.
	w.Header().Set("Referrer-Policy", "strict-origin")
}
