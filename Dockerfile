# syntax=docker/dockerfile:1
FROM golang:1.26.11-alpine AS build
RUN apk add --no-cache nodejs npm
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN npm ci && go generate ./database
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
