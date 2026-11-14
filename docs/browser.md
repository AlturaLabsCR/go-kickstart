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

Browser form posts pass through `http.CrossOriginProtection`, which accepts a
request only when `Sec-Fetch-Site` is `same-origin`/`none`, or, without that
header, when the `Origin` host equals the `Host` header. A rejected post returns
localized 403 and renders "Unable to update" in the page. The debug log records
the reason, `origin`, `host` and `sec_fetch_site`, which is the fastest way to
diagnose a VPN, tunnel or reverse proxy that changes `Host`. Declare the public
origin with `csrf.origins` / `--csrf-origins` /
`APP_CSRF_ORIGINS` (comma-separated, exact `scheme://host[:port]` values)
when the proxy cannot preserve `Host`; those requests bypass both checks. An
`Origin: null` (sandboxed or in-app browsers) has no host to declare, so it can
only pass when the browser also sends `Sec-Fetch-Site`.

After access expiry, `/home` redirects to `/login` to start automatic renewal.
The client uses Web Locks to serialize browser POSTs to the same login path
across tabs, including renewal, verification and sign out. Requests waiting for
the lock send the latest cookies when they start. The renewal handler rechecks
access first, so another tab's successful renewal avoids a second rotation. This
coordination also works when requests reach different server replicas.

Missing or rejected refresh credentials show the ordinary login form without a
retry loop. A failed renewal never clears cookies, so its response cannot erase
a newer successful session. Dependency failures return an error status and keep
the cookies. Automatic renewal requires JavaScript and Web Locks (HTTPS or
localhost); otherwise the ordinary login form is shown. With JavaScript disabled,
a `noscript` link opens that form. A lost successful refresh response still
requires login: the existing one-time refresh contract has no replay recovery.
Sign out revokes the server-side session and removes both cookies. Revocation
invalidates issued access tokens because verification checks the uncached session
record.

All browser mutations use Go's CrossOriginProtection in addition to strict
SameSite cookies. No cookie authentication is enabled on API endpoints, and no
credentialed cross-origin CORS behavior is added. The proxy must preserve the
public Host for origin checks; forwarded headers are not used to grant trust.
Proxy throttling must also cover `POST <root>/login`, since it can issue and
verify OTPs. Existing application email budgets remain in effect.

The feature is isolated in `handlers/browser.go`, `internal/browser` and
`templates/browser`, with its small client in `frontend/browser.ts`. To remove it, remove the `registerBrowserRoutes` call from
`Handler.registerRoutes` and delete those sources and remove the browser build script/dependencies. The web routes use the
handler's optional registration API, so the mandatory API route catalogue does
not need to change. Locale entries beginning with `web.` may then be removed.

`npm ci` installs the locked Node dependencies and builds both Redocly and the
site (Semantic CSS plus templ generation) via its postinstall script. It requires
Go and network access for the pinned stylesheet download. SQL generation remains
`go generate ./database`. The browser build runs ESLint, TypeScript checks and esbuild. Its generated
`handlers/static/js/browser.js` is ignored. `npm run browser` rebuilds just the
client. The MIT notice for the original htmz pattern is in `licenses/htmz.txt`.


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
