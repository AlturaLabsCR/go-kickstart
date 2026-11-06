# go-kickstart

Golang web server template repository,
intented for applications and REST APIs.
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

There are several places where placeholder strings are used,
update with your preferred details.

- [ ] Copyright notices: `NAME HERE`, `EMAIL ADDRESS`.
