# syntax=docker/dockerfile:1
FROM node:22-alpine AS docs
WORKDIR /src
COPY package*.json ./
RUN npm ci
COPY docs/api.yaml docs/api.yaml
RUN npm run redocly

FROM golang:1.26.11-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go generate ./database ./templates
COPY --from=docs /src/handlers/static/api/index.html handlers/static/api/index.html
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app . && mkdir /data

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chown=65532:65532 /data /data
COPY --from=build /out/app /app
USER 65532:65532
WORKDIR /data
EXPOSE 3080
ENTRYPOINT ["/app"]
CMD ["serve"]
