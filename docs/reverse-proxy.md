# Reverse proxies

The default [Dockerfile](../Dockerfile) builds Go, the frontend, templates and SQL,
then runs Go behind [nginx](../deploy/nginx.conf) on port **80**. Go
listens only on `127.0.0.1:3080` as an unprivileged user. TLS terminates upstream;
there are no certificates to install in the container and no SQLite storage is
created. Use an external MySQL or PostgreSQL database through `APP_DB`.
The working directory is `/srv`, owned by root: starting from `/` would let
configuration discovery mistake the extensionless `/app` executable for YAML.

## Railway behind Cloudflare

1. Deploy the repository using its Dockerfile. Leave Build Command and Start
   Command unset so Railway uses the image's entrypoint.
2. Set service variable `PORT=80` and the domain's target port to **80**.
   Railway also uses `PORT` for [deployment health checks](https://docs.railway.com/deployments/healthchecks).
3. Set `APP_DB` to the private database connection string, for example
   `${{MySQL.MYSQL_URL}}` or `${{Postgres.DATABASE_URL}}` (adjust service names).
   Set independent `APP_AUTH_SECRET` and `APP_OTP_SECRET`, each generated with
   `openssl rand -hex 32`. Configure `APP_MAIL_FROM` and `APP_MAIL_SMTP_*` for OTP
   email delivery. Run in production mode, without `APP_DEV=true`.
4. Set Healthcheck Path to `/readyz`. It checks Go and SQL through nginx.
   Set `RAILWAY_DEPLOYMENT_DRAINING_SECONDS=40` so the default five-second drain
   and twenty-five-second shutdown can finish before Railway kills the container.
   See [deployment teardown](https://docs.railway.com/deployments/deployment-teardown).
5. Add your custom domain to Railway, point Cloudflare at the assigned target,
   enable its proxy and use **Full (strict)** TLS. Railway handles the public
   certificate; nginx receives HTTP internally.

`APP_ROOT` is optional. nginx preserves the request path and the original Host,
including its port, so browser forms work without extra CORS/CSRF origins for
same-origin requests. The rate-limit matchers also cover root-prefixed paths.

## Client IP and rate limits

Railway's edge supplies [X-Real-IP](https://docs.railway.com/networking/public-networking/specs-and-limits).
With Cloudflare in front, that peer can be a Cloudflare server. nginx checks this
address against Cloudflare's published [IPv4](https://www.cloudflare.com/ips-v4)
and [IPv6](https://www.cloudflare.com/ips-v6) ranges before using
[CF-Connecting-IP](https://developers.cloudflare.com/fundamentals/reference/http-headers/#cf-connecting-ip).
Otherwise it uses Railway's client address. Missing Cloudflare headers fall back
to that address; direct Railway requests cannot select a new budget by inventing
`CF-Connecting-IP` or `X-Forwarded-For` values. Keep the ranges current and leave
Cloudflare's removal of visitor IP headers disabled.

This configuration trusts Railway's edge to supply `X-Real-IP`: expose the service
through Railway's HTTP gateway, not a TCP proxy or a directly reachable nginx
port. On another host, configure its actual trusted proxies before using these
headers. Local Docker access can set them for testing and is not a trust boundary.

Limits apply to **public IPs, not devices**, and are local to each nginx replica.
Devices sharing Wi-Fi/NAT share a budget; changing networks can change it. IP
limits complement the application's persistent email/account quotas.

| POST paths (with any application root) | Rate | Burst |
| --- | --- | --- |
| `/auth/login`, `/account/email/change`, `/login` | 10/min | 2 |
| `/auth/verify`, `/account/email/change/confirm` | 30/min | 5 |
| `/auth/refresh` | 60/min | 10 |

These are leaky-bucket average rates with immediate bursts and `429` rejections.
All browser form actions share the `/login` budget, including verification,
resumption and logout. GET/HEAD and CORS preflights do not consume these budgets.

nginx negotiates gzip for HTML, JSON, CSS, JS, SVG and other compressible types on
every path, including authenticated and proxied requests. It skips known bodies
below 256 bytes and already-compressed formats. See the
[nginx gzip module](https://nginx.org/en/docs/http/ngx_http_gzip_module.html).

## Entrypoint and standalone application

Go and nginx are two processes. The small [shell entrypoint](../deploy/entrypoint.sh) starts both, stops the
container if either exits, and forwards termination to Go before gracefully
stopping nginx. `tini` handles PID 1 signal forwarding and orphaned processes.
The image explicitly uses `SIGTERM`, overriding nginx's inherited `SIGQUIT`.
For an explicit CLI command, `docker run ... IMAGE migrate` runs only Go.

CI and Kubernetes use the standalone, non-root `app` target instead:

```sh
docker build --target app -t app:multi-instance .
```

That target has `/app` as its entrypoint and needs no startup script or nginx.
The binary release build is unchanged. See [multiple instances](multi-instance.md).

## Generic proxy examples

[contrib/nginx.example.conf](../contrib/nginx.example.conf) is a standalone nginx
example for direct client connections, with the same compression and rate limits.
Include it inside `http {}`, replace `example.com` and configure TLS before public
production use. It uses the socket's client IP and overwrites forwarding headers;
behind another proxy, configure nginx's trusted real-IP sources first. The Docker
image uses `deploy/nginx.conf`, which handles Railway and Cloudflare instead.

For Caddy or Apache, run Go with `--host 127.0.0.1 --port 3080` on the same host.
Replace `example.com` in the examples. They preserve paths but do not implement
per-IP request throttling; configure that at your gateway. They compress text
assets under `/s/`; prepend `APP_ROOT` to that matcher when applicable.

| Proxy | Example | Setup / validation |
| --- | --- | --- |
| Caddy | [contrib/Caddyfile.example](../contrib/Caddyfile.example) | Copy to `/etc/caddy/Caddyfile`; run `caddy validate --config /etc/caddy/Caddyfile`. Caddy manages public HTTPS. |
| Apache 2.4 | [contrib/apache.example.conf](../contrib/apache.example.conf) | Enable `ssl`, `proxy`, `proxy_http`, `headers`, `filter`, `deflate`, `alias`; install TLS certificates; run `apachectl configtest`. |
