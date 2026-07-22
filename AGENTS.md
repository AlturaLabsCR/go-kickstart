# AGENTS.md

This repository is a Go web server template for REST APIs and small HTML surfaces. It uses Cobra/Viper for configuration, `net/http` for routing, `sqlc` for database access, embedded SQL migrations for schema changes, `templ` for HTML, Redocly for OpenAPI docs, and small internal interfaces for auth, database, permissions, localization, middleware, and cache behavior.

## Project Shape

- `cmd/`: CLI commands and runtime wiring. `cmd/serve.go` opens the database, wraps it with cache, creates the authenticator/localizer, and builds the HTTP handler.
- `handlers/`: HTTP route registration and endpoint implementations. Keep related endpoints together: account endpoints belong in `account.go`; a new `/myendpoint/...` group should usually get `myendpoint.go` with `registerMyEndpointRoutes`.
- `middleware/`: HTTP middleware. Middleware should depend on small function/interface types where practical, such as `LocalizeFunc`, instead of depending on full handler state.
- `database/`: database interfaces, models, migrations, sqlc config, and backend implementations.
- `database/postgres` and `database/sqlite`: backend-specific migrations, generated sqlc packages, and query adapters.
- `database/cached`: database read-through cache wrapper. Keep cache logic here, not in handlers.
- `cache/`: small byte-oriented key-value cache interface and memory implementation.
- `auth/`: JWT claims and authenticator construction.
- `perms/`: role and permission keys.
- `i18n/`: locale maps. User-facing strings and error messages should be represented by stable keys.
- `docs/api.yaml`: OpenAPI source. `handlers/static/api/index.html` is generated and ignored.
- `templates/`: `templ` HTML templates.

## Adding an Endpoint

1. Decide the route group and file.
   - Add account-specific routes to `handlers/account.go`.
   - Add a new file like `handlers/myendpoint.go` for a new `/myendpoint/...` group.
   - Add a `registerMyEndpointRoutes()` method and call it from `Handler.registerRoutes()` in `handlers/handler.go`.

2. Choose route protection.
   - Public endpoints can use `h.Add(method, h.routePath("/path"), h.HandlerFunc)`.
   - Authenticated endpoints should use `middleware.AuthenticateBearer(h.logger, h.localize, h.authenticator, next)`.
   - Permission-protected endpoints should wrap authentication plus `middleware.RequirePermission(...)`.
   - Add new middleware only when behavior is reusable across endpoints. Keep endpoint-specific validation inside the handler.

3. Keep request/response handling consistent.
   - Define request/response structs near the handler that uses them unless they are shared.
   - Use `decodeJSON` for JSON bodies.
   - Use `writeJSON` for JSON success responses.
   - Use `h.writeStatus` or `h.writeError` for error responses so logs and localized JSON error bodies stay consistent.
   - Format `h.writeStatus` and `h.writeError` calls one argument per line when they have localization keys or log args.

4. Localize messages and errors.
   - Error keys should be stable and concise, for example `err.select_account`, `err.invalid_login_body`, or `err.permission_denied`.
   - Add every new error key to both `i18n/en.go` and `i18n/es.go`.
   - Add user-visible role or permission labels as keys like `role.default` or `perm.change_email`.
   - Keep logs as clear English messages; keep response bodies localized through keys.

5. Update API docs.
   - Add the endpoint, request body, response schemas, auth requirements, and error responses to `docs/api.yaml`.
   - Run `npm run redocly` after docs changes.
   - Do not manually edit generated `handlers/static/api/index.html`.

## Database Changes

1. Add migrations for both backends.
   - Postgres migrations go in `database/postgres/migrations`.
   - SQLite migrations go in `database/sqlite/migrations`.
   - Use matching version prefixes for the same logical migration in both backends.
   - Filename format is `<version>_<name>.up.sql`, for example `20260722000_widgets.up.sql`.

2. Versioning rules.
   - Migration versions are parsed from the filename prefix as `int64`.
   - Migrations are sorted by version and applied when `version > schema_migrations.version`.
   - Do not reuse a version. The migration loader rejects duplicates.
   - Use a monotonic timestamp-style version. If adding multiple migrations on the same day, increment the suffix: `20260722000`, `20260722001`, etc.
   - The migration SQL does not need to update `schema_migrations`; the migration runner does that after a successful transaction.

3. Add sqlc queries.
   - Postgres query files go in `database/postgres/queries`.
   - SQLite query files go in `database/sqlite/queries`.
   - Keep query names and behavior aligned across both backends when they implement the same app operation.

4. Generate database code.
   - Run `go generate ./database` after migration or query changes.
   - This runs `go tool sqlc generate` using `database/sqlc.yaml`.
   - Generated code lands in ignored backend `db` folders; adapters in `database/postgres/queries` and `database/sqlite/queries` are source files and should be updated manually.

5. Update the database interface and wrappers.
   - Add app-level models to `database/models.go` when generated row types should not leak out.
   - Add methods to `database.Querier` or `database.Database` only for operations needed by application code.
   - Implement each new `Querier` method in both `database/postgres/queries` and `database/sqlite/queries`.
   - Update `database/cached.Querier` so it still satisfies `database.Querier`.

## Caching Guidance

Cache only database reads that are stable, frequently reused, small, and have clear invalidation rules.

Good candidates:

- Account profile lookups by subject.
- Account role lists.
- Role-permission checks.
- Other read-only reference data or low-churn configuration rows.

Avoid caching:

- OTP/login/email-change requests.
- Refresh-token consumption or revocation flows.
- One-time, security-sensitive, or mutation-coupled reads.
- Large result sets or unbounded user-specific lists.
- Reads inside transactions unless the wrapper is explicitly designed for transaction consistency.

Implementation notes:

- Keep cache keys in `database/cached/internal.go`.
- Cache JSON structs/slices inside `database/cached` when useful; keep the public `cache.Store` byte-oriented.
- Invalidate related cache keys immediately after successful writes.
- Ignore best-effort invalidation errors only after the database write succeeds.
- Use the request `ctx` for cache reads/writes. If invalidation must be best effort after a successful write, keep the existing pattern deliberate and localized.

## Permissions and Roles

- Role and permission identifiers are i18n keys, defined in `perms/perms.go`.
- Add locale labels for new keys in `i18n/en.go` and `i18n/es.go`.
- Store role and permission relationships in the database through migrations.
- JWT claims may carry roles, but permission membership should be checked from the database.
- To protect an endpoint, use `RequirePermission` with the specific permission key. Add a role-specific wrapper in route registration only if it meaningfully improves readability.

## Localization and Errors

- `handlers.Handler.localize` is the single handler-owned request localization helper.
- Middleware receives `middleware.LocalizeFunc`; do not pass the whole handler or full `*i18n.Localizer` into middleware.
- API errors should be JSON objects shaped like `{"error":"..."}`.
- Use `h.writeError` when there is an underlying error to log.
- Use `h.writeStatus` when there is no underlying error.
- Keep localization keys short and stable, and keep log messages descriptive.

Example:

```go
h.writeError(
	w,
	r,
	http.StatusInternalServerError,
	err,
	"err.select_widget",
	"failed to select widget",
	"widget_id",
	widgetID,
)
```

## HTML Templates

- Edit `.templ` files in `templates/`.
- Run `go generate ./templates` after template changes.
- Generated `*_templ.go` files are ignored and should not be hand-edited.

## Configuration

- Add new runtime settings in `cmd/serve.go`.
- Keep Viper defaults, flags, flag binding, `runServerFromConfig`, and `runServer` arguments aligned.
- Avoid adding configuration until the code path actually needs it.

## Verification

Run the focused generation command for what changed, then run the project checks:

```sh
go generate ./...
go test ./...
```

Only run the generation commands relevant to the files changed. Do not add new tests unless explicitly requested, but do run existing tests before reporting completion.

## Dependency Updates

- Use `go get -u ./...` and `go mod tidy` for Go modules.
- Use `npm update` for Node dependencies.
- Re-run verification after dependency changes.
- Keep this template current unless the user asks for a pinned or conservative update.
