# go-kickstart

Go web server template for applications and REST APIs.

## Usage

```sh
npm ci # builds Redocly and the HTML templates/stylesheets
go generate ./database

# Start a development server
go run . --dev
# { "msg":"listening","addr":"[::]:3080" }
```

Browser login, protected home and session listing are described in
[the browser authentication guide](docs/browser.md).

## Configuration

Copy [`contrib/app.example.yaml`](contrib/app.example.yaml) to `app.yaml` in the
working directory to customize the defaults, or pass `--config /path/to/app.yaml`.
Flags override environment variables, which override the configuration file.
For environment variables, replace dots and hyphens in keys with underscores
and prefix with `APP_` (for example, `APP_AUTH_ACCESS_TOKEN_TTL=15m`).

Reverse proxy examples for Nginx, Caddy and Apache are in [the reverse proxy guide](docs/reverse-proxy.md).

The starter application's name is **app** (Go module, CLI, configuration and
build artifacts), displayed as **App** in HTML. Rename these together when
creating your application.

Session listing and individual or account-wide logout are available through the JSON API; see [session management](docs/sessions.md), including deployment notes.

See [multiple instances and Kubernetes](docs/multi-instance.md) for container builds,
separate migration jobs, probes, graceful shutdown and shared SQL configuration.

## Extending

See [named routes and CORS](docs/routes.md) for shared URL helpers, startup
validation and handler/endpoint origin policies.

- `go generate ./templates` downloads [Semantic CSS](https://github.com/tavocg/semantic)
  at the commit pinned in `templates/semantic.go`, inlines its CSS imports and writes
  `handlers/static/css/semantic.<sha256>.css` with the upstream licenses. It also
  generates `templates.SemanticStylesheetURL(links)`, used by the shared page
  head. Update `semanticCommit` and regenerate to upgrade; changed CSS gets a new
  URL automatically, and obsolete generated stylesheets are removed. Generation
  requires network access; the built server embeds and serves the CSS locally.
- `npm run browser` bundles the TypeScript frontend into
  `handlers/static/js/main.<sha256>.js` and generates
  `templates.MainScriptURL(links)`. Changed contents get a new URL automatically,
  and obsolete bundles are removed. `npm ci` also runs this build.
- Add a route group in `handlers/`, a registration method and its call in `registerRoutes`.
- Use `Add` for a handler function and `AddHandler` for middleware-wrapped handlers;
  both apply the configured root prefix.
- For endpoints with multiple response formats, see [content negotiation](docs/content-negotiation.md); JSON is the default.
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
