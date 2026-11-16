# Browser authentication

Visit `<root>/login`, enter an email, then enter the six-digit code delivered by
the configured mailer. A native form submits to `POST <root>/login`; the server
responds with `303 See Other`, which makes the browser follow with a GET. HTML
forms do not have a success-redirect attribute. Forms are enhanced with a small
TypeScript adaptation of htmz's iframe transport and Semantic's loader pattern.
Login and confirmation still submit normally if the script is unavailable. Modal
controls use native `commandfor` / `command` attributes, following Semantic;
Cancel is `.outline`, confirm is a normal submit button, and the header has an
accessible close button. Confirm is associated with a hidden POST form, while
Cancel and the close button do not submit anything. Per-session links use
`data-dialog-open` to open their modal through the shared client.
An authenticated GET to `/login` redirects to `/home` after verifying the access
cookie against the live session record. When only the refresh cookie remains,
the page automatically submits `action=resume` to the existing `POST /login` and
returns to `/home` if renewal succeeds. There is no separate continuation button.
GET requests never rotate tokens or change cookies. Prefetch and prerender
requests do not start renewal; an activated prerender reloads to check the session.

`<root>/home` requires the access cookie and lists sessions using the existing
Bearer authentication middleware and `ListSessions` handler, including its
100-row cursor pagination. Refresh sessions sends a native iframe GET to `/home`;
the adapter supplies Bearer authorization internally. Each session has a Sign out
link that opens a native `<dialog>` modal. Cancel and Escape dismiss it without a
request. Confirm posts to `/login` and reuses the API session-revocation handler,
which checks ownership. Revoking the current session also clears its cookies.
GET links never revoke sessions. Device labels are untrusted text and are escaped.
The device column shows the browser and operating system icons recognized from the
stored User-Agent; each icon is omitted when its value is unknown, and the escaped
User-Agent is shown when neither is recognized. Created, Last used and Expires are
rendered in the request language: Spanish puts the day before the abbreviated month
("11 feb") and English after it ("Feb 11"). The year is only shown when it is not
the current year, and the 12-hour clock appears only in Last used. The server emits
a `<time>` element with the UTC instant and a UTC fallback label; the client rewrites
it in the visitor's timezone after the first render and after every htmz swap. With
scripting unavailable the UTC value stays visible.

The dashboard includes the account email, creation date and number of sessions
on the current page (not a total across cursor pages). Its layout uses Semantic
cards, native details/forms and responsive grids; it has no sidebar or additional
client code. Profile and session reads share one verified request identity.

Changing email submits native forms through the same CSRF-protected
`POST <root>/login` adapter: `email_change` reuses `RequestEmailChange` and
`email_confirm` reuses `ConfirmEmailChange`, with Bearer authentication and the
same live `perm.change_email` check as the API. Codes go to the new address and
the existing email remains unchanged until confirmation succeeds. Wrong or
expired codes retain the confirmation form; quotas and localized errors come
from the existing JSON handlers. A missing/expired login instead redirects to
the login page. Cookies are not changed by email confirmation.

Successful submissions use 303 redirects to `/home?email=verify` or
`/home?email=updated`. These flags only select presentation/feedback; they neither
authorize changes nor put the new email/code in the URL. The "Already have a
code?" link reopens confirmation after a reload or session renewal. All paths use
the application prefix. Browser dashboard wiring lives in `handlers/browser_home.go`
and templates in `templates/home`; the public API endpoints are unchanged.

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

Outside development, browser form posts pass through `http.CrossOriginProtection`,
which accepts a request only when `Sec-Fetch-Site` is `same-origin`/`none`, or, without that
header, when the `Origin` host equals the `Host` header. A rejected post returns
localized 403 and renders "Unable to update" in the page. The debug log records
the reason, `origin`, `host` and `sec_fetch_site`, which is the fastest way to
diagnose a VPN, tunnel or reverse proxy that changes `Host`. Declare the public
origin with `csrf.origins` / `--csrf-origins` /
`APP_CSRF_ORIGINS` (comma-separated, exact `scheme://host[:port]` values)
when the proxy cannot preserve `Host`; those requests bypass both checks. An
`Origin: null` (sandboxed or in-app browsers) has no host to declare, so it can
only pass when the browser also sends `Sec-Fetch-Site`.

Pages use `Referrer-Policy: strict-origin` to preserve the native form's `Origin`
while omitting paths and query strings from `Referer`. Using `no-referrer` would
send `Origin: null` on native POSTs, breaking CSRF checks on HTTP LAN hosts where
`Sec-Fetch-Site` is absent. In development, `http://desktop:3080` can therefore
submit same-origin forms without any CORS or trusted-CSRF-origin configuration.
The environment names, when needed, are `APP_CORS_ORIGINS` and
`APP_CSRF_ORIGINS` (plural); CORS controls cross-origin API access, not form CSRF.

After access expiry, `/home` redirects to `/login` to start automatic renewal.
The client uses Web Locks to serialize browser POSTs to the same login path
across tabs, including renewal, verification and sign out. Requests waiting for
the lock send the latest cookies when they start. The renewal handler rechecks
access first, so another tab's successful renewal avoids a second rotation. This
coordination also works when requests reach different server replicas.

Missing or rejected refresh credentials show the ordinary login form without a
retry loop. A failed renewal never clears cookies, so its response cannot erase
a newer successful session. The OTP verification step takes precedence over
automatic renewal, including when cookies belong to a previous database.
Dependency failures return an error status and keep
the cookies. Automatic renewal requires JavaScript and Web Locks (HTTPS or
localhost); otherwise the ordinary login form is shown. With JavaScript disabled,
a `noscript` link opens that form. A lost successful refresh response still
requires login: the existing one-time refresh contract has no replay recovery.
Sign out revokes the server-side session and removes both cookies. Revocation
invalidates issued access tokens because verification checks the uncached session
record. If access-token verification fails because of a database or other
dependency error, sign out returns an error and preserves both cookies for retry.
Invalid or expired access tokens instead fall back to refresh-based revocation.
Once that refresh identifies a session, revocation locks the account and deletes
the stable session, including a token rotated while waiting for the lock. A
consumed refresh can also identify its successor while its hash is retained,
so logout still revokes that session when the access cookie has expired. Only
hashes and session identities are retained, until the original refresh/access
credentials expire; legacy non-expiring refreshes use one configured refresh TTL
from consumption (or the access expiry if later). Unknown or no-longer-retained
hashes are an idempotent no-op. These references never authorize renewal, and a
duplicate refresh request returns 401 without automatically revoking its family.

`--dev` bypasses CrossOriginProtection for the browser's POST login adapter and
permits all CORS origins; production keeps the configured origin policies.
Browser cookies remain SameSite=Strict. No cookie authentication is enabled on
API endpoints, and no credentialed cross-origin CORS behavior is added. The proxy must preserve the
public Host for origin checks; forwarded headers are not used to grant trust.
Proxy throttling must also cover `POST <root>/login`, since it can issue and
verify OTPs. Existing application email budgets remain in effect.

The feature is isolated in `handlers/browser.go`, `internal/browser`,
`templates/login` and `templates/home`, with the shared wrapper in
`templates/base/browser.templ`. The frontend entry point `frontend/main.ts`
initializes `htmz.ts` (iframe requests, busy state and POST coordination),
`dates.ts` (visitor-local timestamps) and `session.ts` (automatic session renewal).
Shared form URL handling lives in `forms.ts`; it reads the action attribute
because a control named `action` shadows the form's JavaScript property.
They compile into one bundle, rendered inline once per document by
`templates.MainScript()` using templ's OnceHandle. The generated module remains
outside `#browser-view`, so swaps do not rerun the parent document's script.
The bundled `window === window.top` guard prevents initialization in the iframe.
To remove the feature, remove the `registerBrowserRoutes` call from
`Handler.registerRoutes`, the `routes.LoginPage` entry and the root page's sign-in
link, then delete the browser sources. If removing the client build too, remove
its build script/dependencies and `MainScriptPath` check in `Handler.validateRoutes`.
Locale entries beginning with `web.` may then be removed.

`npm ci` installs the locked Node dependencies and builds both Redocly and the
site (Semantic CSS plus templ generation) via its postinstall script. It requires
Go and network access for the pinned stylesheet download. SQL generation remains
`go generate ./database`. The browser build runs ESLint, TypeScript checks and esbuild. Its generated
`handlers/static/js/main.<sha256>.js` is ignored. `templates/javascript.go`
fingerprints the final bundle and generates the inline component, its CSP hash
and the optional `templates.MainScriptURL(links)` helper in the ignored
`templates/javascript_generated.go`, preserving the root prefix for external use.
Changed contents get a new URL, identical contents keep it, and obsolete bundles
are removed after publishing the replacement. `npm run browser` rebuilds just the
client. The MIT notice for the original htmz pattern is in `licenses/htmz.txt`.
The generated inline bundle and static assets are embedded when Go builds the server. After rebuilding the
frontend, rebuild and restart the server too; reloading the page cannot replace
the bundle embedded in an already-running process.


Use `form[data-htmz]` for native GET/POST submissions with fields,
`a[data-htmz]` for GET navigation, or
`button[type=button][data-htmz][data-href]` for standalone requests. Standalone
buttons default to GET and may use `data-method="post"`; their name/value is sent
as a field. The URL fragment selects the replacement element, defaulting to
`#browser-view`. Links preserve modified clicks, downloads and `_blank` targets.
All controls share busy state, same-origin restrictions and error handling.
Concurrent submissions are suppressed; busy state clears on completion, failure
or timeout. The iframe is recreated after requests, as in Semantic's no-history
extension. Only the selected response element is swapped, without executing its
scripts. The top URL follows the response through `history.replaceState`.

An iframe cannot attach an Authorization header, and HttpOnly tokens cannot be
read by JavaScript. Browser requests therefore retain cookie authentication in
the web adapter, which supplies Bearer headers to the unchanged API handlers.
No tokens are stored in JavaScript or browser storage. CSRF protection remains
on every browser POST.

## Document security

The shared head embeds the pinned Semantic stylesheet in `<style>` once per
render context, using templ's OnceHandle. No stylesheet request is needed for
these pages. htmz replaces only `#browser-view`, preserving the main document's
style element. Responses still contain full HTML (including the hidden iframe's
own stylesheet); each new document carries the CSS rather than sharing an
external browser-cached file. Only the verified build-time asset is rendered as
raw HTML, and generation rejects a CSS HTML `</style` closing tag.

The application handler sends `X-Content-Type-Options: nosniff`,
`X-Frame-Options: SAMEORIGIN` and CSP `frame-ancestors 'self'`, including on API,
static and routing responses. Same-origin framing keeps htmz working while
blocking embedding by another origin. These headers also apply without nginx.

Own rendered pages (root HTML, login/home and negotiated HTML 404) use the stricter
policy in [handlers/security.go](../handlers/security.go): same-origin resources,
scripts, forms and frames; no plugins or base URL overrides; data images for the
favicon. The inline module is authorized by its generated SHA-256 CSP hash,
calculated over the exact script content, including its trailing newline. Other
inline scripts and eval remain blocked. Generation rejects an HTML `</script`
closing tag before publishing the bundle. Existing inline styles are allowed.
`Referrer-Policy: strict-origin` preserves same-origin
form checks on HTTP LAN hosts while excluding paths and queries from referrers.

Redocly receives the shared anti-framing/nosniff headers, but not the own-page
resource policy: its generated page uses inline scripts and a CDN bundle. Do not
weaken login/home CSP to accommodate documentation resources. Additional HTML
surfaces should explicitly adopt the document policy after checking their needs.
Development uses the same document protections and logs an explicit startup
warning about permissive CORS/CSRF and plaintext OTP debug logs.

## Input and log privacy

OTP verification requires exactly six ASCII digits. Malformed codes still consume
the persistent verification budget and return the existing localized 401 (or 429
when exhausted), without computing the HMAC or querying the challenge. JSON body
size limits and strict decoding remain in place.

Application logs omit email fields from validation/database error context and do
not echo JSON decoder errors, which can contain arbitrary client-supplied field
names. Issuance logs mask both email and OTP outside development, including when
debug logging is explicitly enabled. Development retains the validated email and
issued code for local workflows.

The verification URL still contains the email to preserve the existing
POST/redirect/GET flow. `no-store` and the referrer policy do not remove browser
history. See [proxy log privacy](reverse-proxy.md#client-ip-and-rate-limits) for
access-log sanitization and the separate nginx/platform error-log limitations.
Dependency error messages may also contain provider-supplied details; restrict
operational log access and retention rather than treating logs as public data.
