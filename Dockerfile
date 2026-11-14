# syntax=docker/dockerfile:1
FROM golang:1.26.11-alpine AS build
RUN apk add --no-cache nodejs npm
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN npm ci && go generate ./database
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app .

FROM nginx:stable-alpine
RUN apk add --no-cache su-exec \
    && addgroup -S -g 65532 app \
    && adduser -S -D -H -u 65532 -G app app

COPY --from=build /out/app /app
COPY contrib/nginx.example.conf /etc/nginx/conf.d/default.conf
COPY scripts/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh
EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]
