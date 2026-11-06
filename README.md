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
- `go run . --dev`

Development mode allows empty or short `APP_AUTH_SECRET` and `APP_OTP_SECRET`
with warnings. When the OTP secret is empty, a temporary key is generated;
pending verification codes will not survive a restart.

Outside `--dev`, both secrets must contain at least 32 bytes and differ from
each other. Generate them independently, for example with `openssl rand -hex 32`,
and keep them stable across restarts and instances. Set them through environment
variables, `auth.secret` / `otp.secret`, or `--auth-secret` / `--otp-secret`.
Equal nonempty secrets are rejected in both modes.

Copy [`contrib/app.example.yaml`](contrib/app.example.yaml) to `app.yaml` in the
working directory to customize the defaults, or pass `--config /path/to/app.yaml`.
Set both secrets before starting outside development; the example leaves them
empty intentionally.
Flags override environment variables, which override the configuration file.
For environment variables, replace dots and hyphens in keys with underscores
and prefix with `APP_` (for example, `APP_AUTH_ACCESS_TOKEN_TTL=15m`).

`app` and `app serve` both start the server; `serve` is optional.

Use a clean, unescaped path for `--root`, such as `/app/v1`. Startup rejects
wildcards, dot segments, repeated interior slashes, whitespace, backslashes,
query strings and fragments in the prefix.

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

See [AGENTS.md](AGENTS.md) for project conventions and verification commands.
