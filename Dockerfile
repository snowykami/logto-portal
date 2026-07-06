# syntax=docker/dockerfile:1.7

FROM node:25-alpine AS frontend-deps
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci

FROM frontend-deps AS frontend-build
WORKDIR /src
COPY frontend ./frontend
COPY --from=frontend-deps /src/frontend/node_modules ./frontend/node_modules
RUN npm --prefix frontend run build

FROM golang:1.25-alpine AS backend-build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY configs ./configs
COPY --from=frontend-build /src/internal/static/dist ./internal/static/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/yuki-id-portal ./cmd/server

FROM alpine:3.23
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S portal \
	&& adduser -S portal -G portal
COPY --from=backend-build /out/yuki-id-portal /usr/local/bin/yuki-id-portal
COPY configs ./configs
USER portal
EXPOSE 8080
ENV PORT=8080 \
	APP_BASE_URL=http://localhost:8080 \
	LOGTO_ISSUER=https://auth.liteyuki.org/oidc \
	LOGTO_API_BASE_URL=https://auth.liteyuki.org \
	SESSION_COOKIE_NAME=yp_session \
	APP_CATALOG_PATH=configs/app-catalog.yaml \
	ANNOUNCEMENTS_PATH=configs/announcements.yaml \
	SUPPORT_EMAIL=contact@liteyuki.org
ENTRYPOINT ["yuki-id-portal"]
