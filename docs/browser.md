# Browser authentication

The landing and ordinary `/login` pages use native HTML navigation and forms,
without a client library. The landing email form posts directly to the same login
adapter. `POST <root>/login` adapts those forms to the existing
JSON login/verification handlers and returns `303 See Other`. HTML forms have no
success-redirect attribute. The public API remains JSON/Bearer-only; it does not
accept browser cookies. No new endpoints are required by the dashboard.

The browser adapter preserves request context, User-Agent, validation, persistent
quotas, live permission checks and session issuance. Cookies are host-only,
HttpOnly, SameSite=Strict and scoped to the application prefix. Production cookies
are Secure with `__Secure-` names; development uses separate names for HTTP.
Tokens never enter HTML, JavaScript, browser storage or redirect URLs. Keep the
refresh TTL longer than the access TTL for a useful renewal window.

## Dashboard and Datastar

`GET <root>/home` authenticates with the access cookie, then shares the verified
identity between profile and session reads. It shows the email, account creation
date, sessions on the current cursor page, email-change controls and security
information. Session pages contain up to 100 rows; the count is not an account
wide total. Styling uses Semantic and minimal responsive grid layout.

Only this page dynamically imports Datastar as an external ES module from
`<root>/s/js/datastar.<version>.js`. Module loading is deferred; landing/login do
not load Datastar. `templates/javascript.go` pins its release tag and SHA-256,
verifies cached/downloaded bytes, limits downloads to 1 MiB with a timeout, and
includes its license. Update the tag, bundle digest and reviewed license digest
together. The binary embeds and serves the file; runtime uses no CDN. The official
npm package is older than the pinned release, so the build downloads the official
self-hosted bundle directly. See [Datastar installation](https://data-star.dev/guide/getting_started).

Datastar owns reactive signals, attributes, local dialog opening and HTML morphing.
All dashboard forms use the same `@submit()` action in `frontend/main.ts`:

- It preserves native validation and submits only named form fields through
  `frontend/request.ts`, using `Accept: application/json`.
- The existing CSRF-protected `POST /login` adapter returns localized JSON errors
  with their real HTTP status, or small success signals. The client displays
  messages as text, including non-2xx responses; it never inserts error HTML.
- `email_change` and `email_confirm` reuse the API handlers and their live
  `perm.change_email` checks. Codes go to the new address; the account email stays
  unchanged until confirmation succeeds. Wrong codes retain the confirmation form.
  Successful confirmation updates the displayed email and a status message.
- Revoking another session returns a refresh signal. Revoking the current session
  or signing out clears cookies and returns a same-origin login destination.
- Busy state disables duplicate submissions and is cleared on failure/timeout.
  Mutations are never automatically retried after an ambiguous network failure.

Refresh and pagination share `@refresh()`, wrapping Datastar GET with no signals,
a same-origin check, busy/error handling and a 30-second timeout.
`GET /home?fragment=sessions` renders only the sessions card and count, reusing the
same API session-list handler. It avoids the profile lookup, document wrapper,
CSS and scripts. Datastar morphs those two elements, preserving the email form,
profile, document header/footer and focus where possible. Server-rendering rows
avoids duplicating device icons, translations, escaping and table templates in
JavaScript. JSON serves mutations/messages; HTML serves this list fragment.

Every session has an escaped device label and a Sign out link opening a native
`dialog`. Its hgroup contains a header with title/close button and a description
below. Cancel/Escape close locally. Confirmation uses a native POST form enhanced
by the shared action; GET links never revoke sessions. `commandfor` buttons handle
native modal controls. Without JavaScript, login and email forms still submit
normally; session links require Datastar to open their confirmation dialog.

The native form fallback keeps POST/redirect/GET email confirmation via
`/home?email=verify` and `/home?email=updated`. These flags are presentation only;
they never authorize a change or put the new address/code in the URL. The
"Already have a code?" link allows reopening confirmation after a reload.

Timestamps use `<time datetime>` with a localized UTC fallback. `frontend/dates.ts`
converts them to the visitor's timezone initially and after session fragments.
User-Agent labels are descriptive, untrusted input; they never authorize a device.

## Session restoration and revocation

A valid access cookie redirects `GET /login` to `/home`. When only the refresh
cookie remains, login loads the small inline controller to automatically POST
`action=resume`; the human login forms remain native. GET never rotates tokens or
changes cookies. Prefetch does not initiate renewal; activated prerenders reload.

`frontend/request.ts` holds an origin-wide Web Lock for adapter fetch POSTs until
the response body is read and cookies are applied. Waiting requests send the
latest cookies. Renewal rechecks access under that coordination so another tab's
successful renewal avoids a second rotation. Native login form navigations are
not intercepted or serialized by this helper. Locks coordinate clients; the
server still enforces all authentication and ownership checks.

Automatic restoration requires JavaScript and Web Locks (HTTPS or localhost).
Without locks, rejected refresh credentials, or a failed request, the ordinary
login form is shown without a retry loop. A `noscript` link provides that form
when scripting is disabled. Failed renewal never clears cookies: its delayed
response must not erase a newer successful session. OTP verification takes
precedence over restoration, including cookies from a previous database.
A lost successful refresh response still requires login; refresh is one-time.

Sign out revokes the stable server-side session before clearing both cookies.
Access verification checks its uncached session record. Dependency failures
preserve cookies for retry. Invalid/expired access falls back to refresh-based
revocation, including retained consumed hashes identifying a rotated successor.
Those references never authorize renewal; duplicate refresh does not automatically
revoke a family. Retention ends at the original refresh/access expiry; legacy
non-expiring refreshes use one configured refresh TTL from consumption, or the
access expiry if later. Unknown/expired references are an idempotent no-op.

## CSRF, CSP and privacy

Every browser POST, including JSON-adapter requests, passes through
`http.CrossOriginProtection`. Production accepts same-origin/none Fetch Metadata,
or an Origin matching Host when Metadata is absent. Trusted exact origins can be
configured through `csrf.origins` / `--csrf-origins` / `APP_CSRF_ORIGINS`.
`Origin: null` cannot be listed as a trusted host. The proxy must preserve public
Host; forwarded headers do not grant trust. CORS settings control API access,
not form CSRF. `--dev` permits all CORS origins and bypasses browser POST CSRF;
production keeps the configured policies. Cookies remain SameSite=Strict.

The proxy must also throttle `POST <root>/login`, which issues/verifies OTPs.
Application identity quotas remain in effect. Codes must be six ASCII digits;
malformed codes still consume verification quota. Decoder size/field checks and
API contracts are unchanged.

The shared head embeds verified Semantic CSS once per render context. The small
local controller is inline once on home or a restoring-login page, authorized by
its exact generated SHA-256 CSP hash. Ordinary login and landing need no script.
Datastar stays external. Home generates a fresh random CSP nonce for each full
response and supplies it through `html[data-nonce]`; Datastar consumes it for its
[CSP mode](https://data-star.dev/reference/security). Neither `unsafe-eval` nor
arbitrary inline scripts are allowed. Expressions are static; user values use
serialized signals or escaped text, never interpolated executable expressions.
Fragment responses contain no scripts or new nonce. Redocly retains its separate
resource requirements. Shared anti-framing and nosniff headers remain in place.

`Referrer-Policy: strict-origin` preserves native form Origin on HTTP LAN hosts
without exposing paths/queries in referrers. The verification URL still contains
email for the native redirect flow; no-store does not remove browser history.
Application logs mask email/OTP outside development; restrict operational log
access. See [proxy log privacy](reverse-proxy.md#client-ip-and-rate-limits) for
nginx/platform logging limitations.

## Build and removal

`npm ci` builds docs and frontend through postinstall. `npm run browser` runs ESLint,
TypeScript and esbuild, downloads/verifies Datastar and generates the ignored
fingerprinted `main.<sha256>.js`, inline component and CSP hash. `MainScriptURL`
remains available for external consumers. `go generate ./templates` downloads
pinned Semantic and generates templ files; SQL generation is separate. Rebuild
and restart the Go server after generating assets: a running binary cannot load
a replacement for its embedded files.

The feature lives in `handlers/browser*.go`, `internal/browser`, `templates/login`,
`templates/home` and `frontend`. To remove it,
remove `registerBrowserRoutes`, the LoginPage catalogue entry and landing sign-in
links, then remove the browser sources/build and related static-asset validation.
The API authentication contracts do not depend on this feature.
