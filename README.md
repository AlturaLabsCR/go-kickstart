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

There are several places where placeholder strings are used,
update with your preferred details.

- [ ] Copyright notices: `NAME HERE`, `EMAIL ADDRESS`.
