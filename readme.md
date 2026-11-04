# Chirpy

Chirpy is a Twitter-like REST API written in Go. It supports user accounts, secure sessions, short posts called **chirps**, and a trusted payment webhook that upgrades users to Chirpy Red.

Built as part of the [Boot.dev](https://www.boot.dev/) backend curriculum, it is a compact, practical example of a production-style Go service: HTTP routing, authentication, PostgreSQL persistence, migrations, generated database code, tests, and basic operations.

## Why Chirpy?

Chirpy intentionally keeps the product small so the important backend choices stay visible. It demonstrates how to:

- hash passwords instead of storing them in plaintext;
- issue short-lived JWT access tokens alongside revocable refresh tokens;
- authorize users to modify only their own resources;
- version PostgreSQL schema changes with migrations;
- use SQLC for explicit SQL with typed Go query methods; and
- authenticate a third-party webhook with a dedicated API key.

The goal is not a complete social network—it is a readable reference implementation for the core building blocks of a secure Go API.

## Features

- User registration and authenticated account updates
- Argon2id password hashing
- One-hour JWT access tokens and 60-day revocable refresh tokens
- Create, retrieve, filter, sort, and delete chirps
- Ownership checks for chirp deletion
- 140-character chirp limit and simple word filtering
- Polka webhook authentication and idempotent Chirpy Red upgrades
- Health checks, static app serving, request metrics, and a development reset endpoint

## Architecture and technology

| Area | Choice |
| --- | --- |
| Language/runtime | Go |
| HTTP | Go `net/http` and `ServeMux` |
| Database | PostgreSQL |
| Migrations | [Goose](https://github.com/pressly/goose) |
| Query generation | [SQLC](https://sqlc.dev/) |
| Password hashing | Argon2id |
| Access authentication | JWT |

Handlers and routing are in `main.go`; reusable password, token, and API-key helpers are in `internal/auth`; SQLC-generated data access lives in `internal/database`; source SQL is in `sql/`.

## Prerequisites

- [Go](https://go.dev/dl/) 1.27.1 (or the version in `go.mod`)
- [PostgreSQL](https://www.postgresql.org/download/)
- [Goose](https://github.com/pressly/goose) for migrations
- [SQLC](https://sqlc.dev/) when editing SQL schema or query definitions

Goose and SQLC are needed for database setup and database-code changes; the generated Go code is committed, so ordinary builds and tests do not regenerate it.

## Installation and database setup

```bash
git clone https://github.com/JB4779/chirpy.git
cd chirpy
go mod download
createdb chirpy
```

Create `.env` in the repository root. It is ignored by Git—do not commit it.

```dotenv
DB_URL="postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable"
PLATFORM="dev"
JWT_SECRET="replace-with-a-long-random-secret"
POLKA_KEY="replace-with-your-polka-webhook-key"
```

`JWT_SECRET` signs access tokens and `POLKA_KEY` authenticates Polka webhook calls. Use unique high-entropy secrets and keep real values in a secret manager or local `.env`, never in the repository, documentation, or issues.

Apply the versioned PostgreSQL migrations:

```bash
goose -dir sql/schema postgres "$DB_URL" up
```

If your shell does not load `.env`, export `DB_URL` first or provide the database URL directly. After changing `sql/schema` or `sql/queries`, regenerate typed database code:

```bash
sqlc generate
```

## Build, run, and test

Start the server on `http://localhost:8080`:

```bash
go run .
```

Or build it first:

```bash
go build -o out .
./out
```

```bash
go test ./...
curl http://localhost:8080/api/healthz
# OK
```

## Quick start

Create an account, then log in and copy the returned `token` and `refresh_token`:

```bash
curl -X POST http://localhost:8080/api/users -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","password":"use-a-strong-password"}'

curl -X POST http://localhost:8080/api/login -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","password":"use-a-strong-password"}'
```

Create and then list chirps:

```bash
curl -X POST http://localhost:8080/api/chirps \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"body":"Hello from Chirpy!"}'

curl "http://localhost:8080/api/chirps?author_id=<user-uuid>&sort=desc"
```

## Authentication flow

| Purpose | Header | Details |
| --- | --- | --- |
| Authenticated user request | `Authorization: Bearer <access-token>` | JWT, expires after one hour |
| Refresh or revoke a session | `Authorization: Bearer <refresh-token>` | Stored random token, expires after 60 days unless revoked |
| Polka webhook | `Authorization: ApiKey <POLKA_KEY>` | Shared secret for trusted webhook calls |

`POST /api/users` stores an Argon2id hash. `POST /api/login` returns a JWT access token plus a persisted refresh token. Use `POST /api/refresh` to exchange a valid, unexpired, unrevoked refresh token for a new access token, and `POST /api/revoke` to end a session. Account updates and protected chirp operations derive the user ID from the JWT—not client input.

## API reference

Errors are returned as JSON:

```json
{ "error": "message" }
```

### Health, users, and sessions

| Method | Route | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/api/healthz` | None | `200 OK` with `OK` when the service is running. |
| `POST` | `/api/users` | None | Create a user with `email` and `password`; returns `201 Created`. |
| `PUT` | `/api/users` | Access token | Update the current user’s email and password. |
| `POST` | `/api/login` | None | Verify credentials; return user, `token`, and `refresh_token`. |
| `POST` | `/api/refresh` | Refresh token | Return a new access token. |
| `POST` | `/api/revoke` | Refresh token | Revoke the refresh token; returns `204 No Content`. |

Bad credentials, missing credentials, and invalid tokens receive `401 Unauthorized`.

### Chirps

| Method | Route | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/api/chirps` | Access token | Create a chirp for the authenticated user. |
| `GET` | `/api/chirps` | None | List chirps, with optional author filter and sort order. |
| `GET` | `/api/chirps/{chirpID}` | None | Get a chirp by UUID. |
| `DELETE` | `/api/chirps/{chirpID}` | Access token | Delete the caller’s own chirp. |

Create requests have a `body` no longer than 140 characters. The standalone words `kerfuffle`, `sharbert`, and `fornax` are replaced case-insensitively with `****`. Deleting another user’s chirp returns `403 Forbidden`; a missing chirp returns `404 Not Found`.

#### Filtering and sorting

`GET /api/chirps` accepts optional query parameters:

| Parameter | Values | Default | Effect |
| --- | --- | --- | --- |
| `author_id` | User UUID | — | Return chirps from one author. |
| `sort` | `asc`, `desc` | `asc` | Order results by `created_at`. |

```text
GET /api/chirps
GET /api/chirps?sort=asc
GET /api/chirps?sort=desc
GET /api/chirps?author_id=<user-uuid>&sort=desc
```

An empty result is `[]`; an invalid `author_id` returns `400 Bad Request`.

### Polka webhooks and Chirpy Red

`POST /api/polka/webhooks` requires:

```text
Authorization: ApiKey <POLKA_KEY>
```

Missing or incorrect API keys return `401 Unauthorized`. A valid upgrade event is:

```json
{ "event": "user.upgraded", "data": { "user_id": "<user-uuid>" } }
```

It sets `is_chirpy_red` to `true` and returns `204 No Content`. The operation is idempotent; unknown events are ignored with `204`, malformed JSON returns `400`, and a nonexistent user returns `404`.

### Admin and static routes

| Method | Route | Description |
| --- | --- | --- |
| `GET` | `/admin/metrics` | HTML page with file-server request count since startup. |
| `POST` | `/admin/reset` | Development-only: deletes users and related data, then resets metrics. |
| `GET` | `/app/` | Serves static app files and assets. |

The reset route only works when `PLATFORM="dev"`; otherwise it returns `403 Forbidden`.

## Project structure

```text
.
├── main.go                 # Server, routing, handlers, middleware
├── main_test.go            # HTTP/API tests
├── internal/auth/          # Password, JWT, refresh-token, API-key helpers
├── internal/database/      # SQLC-generated models and query methods
├── sql/schema/             # Goose migrations
├── sql/queries/            # SQLC query definitions
├── sqlc.yaml               # SQLC configuration
└── .env                    # Local secrets (ignored; create it yourself)
```

## Development notes

Before contributing, run:

```bash
go test ./...
```

For database changes: add a forward-only migration, apply it with Goose, update SQLC query definitions as needed, run `sqlc generate`, review the generated diff, and add or update tests. Keep tokens, passwords, API keys, and credential-bearing database URLs out of logs and commits.

Issues and focused pull requests are welcome.
