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
- Scopes: `openid profile email role urn:logto:scope:organizations urn:logto:scope:organization_roles`

Set `LOGTO_CLIENT_ID` and `LOGTO_CLIENT_SECRET` before using `/auth/login`.
Set `LOGTO_MANAGEMENT_CLIENT_ID`, `LOGTO_MANAGEMENT_CLIENT_SECRET`, and
`LOGTO_MANAGEMENT_API_RESOURCE` before allowing profile updates.
For local UI preview without Logto credentials, run with `PORTAL_DEV_AUTH=true`.

Set `LOGTO_ACCOUNT_BASE_URL` to the Logto Account Center base URL. For Liteyuki:

```text
https://auth.liteyuki.org/account
```

The password page used by the portal is therefore:

```text
https://auth.liteyuki.org/account/password
```

The Management API client should be a Logto Machine-to-Machine app with a role
that grants the required Management API permission, for example `all` for the
first deployment. The token is only requested by the Go BFF and is never returned
to the browser.

When Management API is configured, `/api/app-catalog` fetches applications from
Logto via `GET /api/applications`. `configs/app-catalog.yaml` is still used as a
safe local overlay for portal URLs, icons, ordering, and role / organization
access rules. Static YAML entries never create standalone portal apps; if
Management API is unavailable, the portal returns an empty application list.

Developer self-service requests are stored in `PORTAL_REQUESTS_PATH`. Users can
request SPA or Traditional applications and global roles. Admin users with
`liteyuki-account-admin` review the requests; approval calls Logto Management
API to create applications or assign user roles.
Traditional application secrets are not persisted or returned to ordinary users
by the portal; handle secret delivery or rotation through a separate audited
admin flow.

## Production Shape

```text
frontend build -> internal/static/dist -> Go embed.FS -> single binary/container
```

The frontend only talks to `/api`. Logto Account API calls are proxied through the Go BFF.
Management API tokens must never be sent to the browser.
