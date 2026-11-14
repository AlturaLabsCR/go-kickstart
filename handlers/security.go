package handlers

import "net/http"

// Inline styles preserve the existing device layout; scripts stay same-origin.
const documentCSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; object-src 'none'; base-uri 'none'; form-action 'self'; frame-src 'self'; frame-ancestors 'self'"

// Apply only to our rendered pages; Redocly has separate resource requirements.
func documentHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", documentCSP)
	// Preserve same-origin form Origin on HTTP LAN hosts without exposing queries.
	w.Header().Set("Referrer-Policy", "strict-origin")
}
