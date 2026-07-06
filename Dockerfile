FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git nodejs npm

COPY . .

RUN npm --prefix frontend ci
RUN npm --prefix frontend run build
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/yuki-id-portal ./cmd/server

FROM alpine:3.23

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/yuki-id-portal /usr/local/bin/yuki-id-portal
COPY configs ./configs

EXPOSE 8080

ENV PORT=8080 \
	APP_BASE_URL=http://localhost:8080 \
	LOGTO_ISSUER=https://auth.liteyuki.org/oidc \
	LOGTO_API_BASE_URL=https://auth.liteyuki.org \
	LOGTO_MANAGEMENT_API_RESOURCE=https://default.logto.app/api \
	LOGTO_MANAGEMENT_API_SCOPE=all \
	SESSION_COOKIE_NAME=yp_session \
	APP_CATALOG_PATH=configs/app-catalog.yaml \
	ANNOUNCEMENTS_PATH=configs/announcements.yaml \
	SUPPORT_EMAIL=contact@liteyuki.org

ENTRYPOINT ["yuki-id-portal"]
