# Reverse proxy examples

Choose one proxy and replace `example.com` with your domain. These examples
assume the proxy and application run on the same host. Start the application
with `--host 127.0.0.1 --port 3080` and configure its secrets as usual.

| Proxy | Example | Setup / validation |
| --- | --- | --- |
| Nginx | [`contrib/nginx.example.conf`](../contrib/nginx.example.conf) | Include inside `http`; install your certificate/key at the shown paths; run `nginx -t`. |
| Caddy | [`contrib/Caddyfile.example`](../contrib/Caddyfile.example) | Copy to `/etc/caddy/Caddyfile`; run `caddy validate --config /etc/caddy/Caddyfile`. Caddy manages HTTPS for a domain pointing to this server with ports 80/443 reachable. |
| Apache 2.4 | [`contrib/apache.example.conf`](../contrib/apache.example.conf) | Enable `ssl`, `proxy`, `proxy_http`, `headers`, `filter`, `deflate`, `alias` and listening on 80/443. Install the certificate/key; include the virtual hosts; run `apachectl configtest`. On Debian/Ubuntu, use `a2enmod` to enable modules. |

All three preserve the request path. Nginx negotiates gzip for HTML, JSON and
other compressible types on every path, including responses to authenticated
requests and requests through another proxy. Bodies with a known size below
256 bytes and already-compressed binary formats are left uncompressed. See the
[Nginx gzip module](https://nginx.org/en/docs/http/ngx_http_gzip_module.html).
Caddy and Apache limit compression to text assets under `/s/`.
If using `--root /app`, prepend `/app` to Caddy/Apache's static matcher and
Nginx's rate-limited paths; keep the proxy upstream mapping unchanged so the
application receives the prefix. Nginx's compression already covers the prefix.

Nginx includes per-IP limits: issuance 10/minute, verification 30/minute and
refresh 60/minute, with small bursts and `429` rejections. These are tunable
average rates (leaky buckets), not fixed-window quotas. Login and email changes
share their issuance/verification zones. The application's identity limits also
apply.

The stock Caddy and Apache examples do **not** enforce per-IP request limits.
Caddy's [rate-limit handlers](https://caddyserver.com/docs/modules/http.handlers.rate_limit)
require an extra module. Apache's
[`mod_ratelimit`](https://httpd.apache.org/docs/2.4/mod/mod_ratelimit.html)
limits response bandwidth, not request counts. Add request throttling through
an appropriate extension or an upstream gateway when using those examples.

These configurations assume direct client connections to the proxy and replace
forwarded client information rather than accepting arbitrary forwarding headers.
If placing a CDN or another proxy in front, configure trusted proxies and the
client-IP rate-limit key for that deployment before relying on per-IP limits.
For containers on different hosts/networks, replace the loopback upstream with
the application's private address and prevent public access to its port.

## Dokploy with Docker Compose

Create a **Docker Compose** service in Dokploy (not an Application using Nixpacks
or Railpack). Select the repository and set Compose Path to `./docker-compose.yml`.
Use Compose mode, not Stack: the app is built with the existing Dockerfile.
Use Docker Compose 2.23.1 or newer. nginx's configuration is included inline in
`docker-compose.yml` using `configs.content`; no checkout file is bind-mounted
from the Dokploy host. nginx variables use `$$` to survive Compose interpolation.

In Environment, set `MYSQL_PASSWORD`, `MYSQL_ROOT_PASSWORD`, `APP_AUTH_SECRET` and
`APP_OTP_SECRET` to separate secrets generated with `openssl rand -hex 32`.
Configure `APP_MAIL_FROM` and `APP_MAIL_SMTP_*` to deliver login codes.
`APP_ROOT` is optional; nginx preserves the full request path and Host.

Enable **Isolated Deployments**. In Domains, select service **nginx**, container
port **80**, and enable HTTPS. Dokploy's gateway terminates HTTPS before nginx;
no application certificates or published host ports are required. Use Preview
Compose to confirm all three services retain their shared `backend` network.
See Dokploy's [Compose setup](https://docs.dokploy.com/docs/core/docker-compose)
and [domain settings](https://docs.dokploy.com/docs/core/docker-compose/domains).

MySQL data persists in the `mysql-data` named volume. The app waits for MySQL's
health check and applies its existing startup migrations. Configure per-client-IP
OTP throttling at Dokploy's gateway; the app's identity quotas remain enabled.
nginx compresses HTML, JSON and other compressible types on every route.

The root Dockerfile, its ignore file, the release workflow and Kubernetes examples
are unchanged. Each container runs its own service; no custom process supervisor
or deployment-specific image is needed.
