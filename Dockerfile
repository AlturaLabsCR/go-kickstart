# syntax=docker/dockerfile:1
FROM golang:1.26.11-alpine AS build
RUN apk add --no-cache nodejs npm
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN npm ci && go generate ./database
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app .

# Standalone binary for CI, migration jobs and Kubernetes.
FROM alpine:3 AS app
RUN apk add --no-cache ca-certificates
COPY --from=build /out/app /app
# Keep config discovery away from the extensionless /app executable.
WORKDIR /srv
USER 65532:65532
EXPOSE 3080
ENTRYPOINT ["/app"]

FROM nginx:stable-alpine
RUN apk add --no-cache ca-certificates su-exec tini \
    && addgroup -S -g 65532 app \
    && adduser -S -D -H -u 65532 -G app app

COPY --from=build /out/app /app
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY --chmod=755 deploy/entrypoint.sh /entrypoint.sh

WORKDIR /srv
ENV PORT=80
EXPOSE 80
STOPSIGNAL SIGTERM
ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
