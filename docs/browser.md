# Browser authentication

Visit `<root>/login`, enter an email, then enter the six-digit code delivered by
the configured mailer. A native form submits to `POST <root>/login`; the server
responds with `303 See Other`, which makes the browser follow with a GET. HTML
forms do not have a success-redirect attribute. No JavaScript is required.

`<root>/home` requires the access cookie and lists sessions using the existing
Bearer authentication middleware and `ListSessions` handler, including its
100-row cursor pagination. Refresh sessions submits a native GET to `/home`;
the adapter supplies Bearer authorization internally. Device labels are untrusted text and are escaped.

The browser adapter calls the existing login, verification and refresh handlers
in process with JSON bodies, preserving the request context and User-Agent. It
reuses their validation, persistent quotas, OTP consumption and session issuance.
The public API still accepts JSON, returns tokens, and authenticates exclusively
with Bearer headers: it does not accept these browser cookies.

Cookies are host-only, HttpOnly, SameSite=Strict and scoped to the application
prefix. Outside development they are Secure and use the `__Secure-` name prefix;
serve the application through HTTPS. `--dev` permits local HTTP with separate
cookie names. Tokens never enter HTML, browser storage or redirect URLs. Keep
refresh-token TTL greater than access TTL for a useful renewal window.

After access expiry, `/home` redirects to `/login`; the Continue session form
rotates the refresh token through the existing refresh handler. Renewal uses
POST, not a side effect of GET. Sign out revokes the server-side session using
its refresh token and removes both cookies. Revocation invalidates issued access
tokens because verification checks the uncached session record.

All browser mutations use Go's CrossOriginProtection in addition to strict
SameSite cookies. No cookie authentication is enabled on API endpoints, and no
credentialed cross-origin CORS behavior is added. The proxy must preserve the
public Host for origin checks; forwarded headers are not used to grant trust.
Proxy throttling must also cover `POST <root>/login`, since it can issue and
verify OTPs. Existing application email budgets remain in effect.

The feature is isolated in `handlers/browser.go`, `internal/browser` and
`templates/browser`. To remove it, remove the `registerBrowserRoutes` call from
`Handler.registerRoutes` and delete those three sources. The web routes use the
handler's optional registration API, so the mandatory API route catalogue does
not need to change. Locale entries beginning with `web.` may then be removed.

`npm ci` installs the locked Node dependencies and builds both Redocly and the
site (Semantic CSS plus templ generation) via its postinstall script. It requires
Go and network access for the pinned stylesheet download. SQL generation remains
`go generate ./database`. There is no custom frontend JavaScript or TypeScript,
so no ESLint toolchain is needed for this feature.
