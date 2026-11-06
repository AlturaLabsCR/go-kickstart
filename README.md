# go-kickstart

Go web server template for applications and REST APIs.
Very opinionated, your mileage may vary.

## Decisions

- [`spf13/cobra`](https://github.com/spf13/cobra) - CLI interaction library
- [`spf13/viper`](https://github.com/spf13/viper) - Configuration (files, variables, overrides, merges)
- [`a-h/templ`](https://github.com/a-h/templ) - Type-safe HTML templates
- [`sqlc-dev/sqlc`](https://github.com/sqlc-dev/sqlc) - Type-safe SQL code generation (no ORM)
- GPL-licensed

## Start using this template

- `npm ci`
- `go generate ./...`
- `APP_OTP_SECRET=$(openssl rand -hex 32) go run .`

Verification codes are stored as keyed HMAC digests, so the server requires an
independent `APP_OTP_SECRET` (or `otp.secret` / `--otp-secret`). It must differ
from `APP_AUTH_SECRET`. Use a stable value across restarts and instances.

Copy [`contrib/app.example.yaml`](contrib/app.example.yaml) to `app.yaml` in the
working directory to customize the defaults, or pass `--config /path/to/app.yaml`.
Set `otp.secret` before starting; the example leaves it empty intentionally.
Flags override environment variables, which override the configuration file.
For environment variables, replace dots and hyphens in keys with underscores
and prefix with `APP_` (for example, `APP_AUTH_ACCESS_TOKEN_TTL=15m`).

`app` and `app serve` both start the server; `serve` is optional.

Reverse proxy examples for Nginx, Caddy and Apache are in [the reverse proxy guide](docs/reverse-proxy.md).

The repository is named **go-kickstart**; the starter application's name is
**app** (Go module, CLI, configuration and build artifacts), displayed as **App**
in HTML. Rename these together when creating your application. The repository's
[LICENSE](LICENSE) is the source of license terms; source files omit placeholder
copyright headers. Keep project-specific attribution accurate when adapting it.

## Extending the application

- Add a route group in `handlers/`, a registration method and its call in `registerRoutes`.
- Use `Add` for a handler function and `AddHandler` for middleware-wrapped handlers;
  both apply the configured root prefix.
- Keep request/response types near their handler. Use `decodeBody` for localized
  validation errors and `internal/httpx` for JSON responses.
- Put reusable behavior in `middleware/`; authorization checks current permissions
  with one uncached database query.
- Add matching migrations and queries for SQLite, PostgreSQL and MySQL, then run
  `go generate ./database`. Keep driver-specific details in their adapters.
- Keep read-through caching in `database/cached`; never cache authentication,
  permissions, OTPs or refresh tokens.
- Add messages to both locales, update `docs/api.yaml`, and run `npm run redocly`.

See [AGENTS.md](AGENTS.md) for conventions and [docs/TODO.md](docs/TODO.md) for
review decisions and remaining automated coverage gaps.
