# Yuki ID Portal

Yuki ID Portal is the account portal for `account.liteyuki.org`.
It uses Logto as the identity provider and keeps the first release database-free.

## Development

```bash
npm install
npm --prefix frontend install
npm run build
go run ./cmd/server
```

Required Logto settings:

- Redirect URI: `http://localhost:8080/auth/callback`
- Post logout redirect URI: `http://localhost:8080/`
- Scopes: `openid profile email roles urn:logto:scope:organizations urn:logto:scope:organization_roles`

Set `LOGTO_CLIENT_ID` and `LOGTO_CLIENT_SECRET` before using `/auth/login`.
For local UI preview without Logto credentials, run with `PORTAL_DEV_AUTH=true`.

## Production Shape

```text
frontend build -> internal/static/dist -> Go embed.FS -> single binary/container
```

The frontend only talks to `/api`. Logto Account API calls are proxied through the Go BFF.
Management API tokens must never be sent to the browser.
