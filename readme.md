# Chirpy

Chirpy is a REST API written in Go. It provides user management, authentication with JWT access tokens and refresh tokens, chirp creation/retrieval/deletion, administrative endpoints, and Polka webhook integration.

## Technology

- Go
- PostgreSQL
- SQLC
- Goose migrations
- Argon2id password hashing
- JWT authentication

## Running Chirpy

Start PostgreSQL if necessary:

```bash
sudo service postgresql start
```

From the Chirpy project root:

```bash
go build -o out && ./out
```

The server runs at:

```text
http://localhost:8080
```

Run the test suite with:

```bash
go test ./...
```

## Environment Variables

Chirpy uses a `.env` file in the project root.

```text
DB_URL="postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable"
PLATFORM="dev"
JWT_SECRET="<secret>"
POLKA_KEY="<polka-api-key>"
```

The `.env` file must not be committed to Git.

---

# API

## Health Check

### `GET /api/healthz`

Checks whether the Chirpy server is running.

**Response**

```text
200 OK
```

```text
OK
```

---

# Users

## Create User

### `POST /api/users`

Creates a new Chirpy user.

**Request**

```json
{
  "email": "user@example.com",
  "password": "password"
}
```

The password is hashed with Argon2id before being stored. Plaintext passwords are never stored.

**Success**

```text
201 Created
```

```json
{
  "id": "5a47789c-a617-444a-8a80-b50359247804",
  "created_at": "2026-11-04T10:00:00Z",
  "updated_at": "2026-11-04T10:00:00Z",
  "email": "user@example.com",
  "is_chirpy_red": false
}
```

---

## Update User

### `PUT /api/users`

Updates the authenticated user's email and password.

**Authentication**

```text
Authorization: Bearer <access-token>
```

**Request**

```json
{
  "email": "new@example.com",
  "password": "new-password"
}
```

The user ID is determined from the JWT. A user cannot use this endpoint to update another user's account.

**Success**

```text
200 OK
```

```json
{
  "id": "5a47789c-a617-444a-8a80-b50359247804",
  "created_at": "2026-11-04T10:00:00Z",
  "updated_at": "2026-11-04T11:00:00Z",
  "email": "new@example.com",
  "is_chirpy_red": false
}
```

**Errors**

```text
401 Unauthorized
```

Returned when the access token is missing or invalid.

---

# Authentication

## Login

### `POST /api/login`

Authenticates a user using email and password.

**Request**

```json
{
  "email": "user@example.com",
  "password": "password"
}
```

**Success**

```text
200 OK
```

```json
{
  "id": "5a47789c-a617-444a-8a80-b50359247804",
  "created_at": "2026-11-04T10:00:00Z",
  "updated_at": "2026-11-04T10:00:00Z",
  "email": "user@example.com",
  "is_chirpy_red": false,
  "token": "<jwt-access-token>",
  "refresh_token": "<refresh-token>"
}
```

The access token is a JWT valid for **1 hour**.

The refresh token is a random 256-bit token valid for **60 days**.

**Authentication Failure**

```text
401 Unauthorized
```

```json
{
  "error": "Incorrect email or password"
}
```

---

## Refresh Access Token

### `POST /api/refresh`

Creates a new one-hour access token using a valid refresh token.

There is no request body.

**Authentication**

```text
Authorization: Bearer <refresh-token>
```

**Success**

```text
200 OK
```

```json
{
  "token": "<new-jwt-access-token>"
}
```

**Failure**

```text
401 Unauthorized
```

A refresh token cannot be used if it:

- Does not exist
- Has expired
- Has been revoked

---

## Revoke Refresh Token

### `POST /api/revoke`

Revokes a refresh token.

There is no request body.

**Authentication**

```text
Authorization: Bearer <refresh-token>
```

**Success**

```text
204 No Content
```

Once revoked, the refresh token can no longer be used to obtain new access tokens.

---

# Chirps

## Create Chirp

### `POST /api/chirps`

Creates a chirp for the authenticated user.

**Authentication**

```text
Authorization: Bearer <access-token>
```

**Request**

```json
{
  "body": "Hello, world!"
}
```

The user ID is taken from the JWT and cannot be supplied by the client.

Chirps are limited to **140 characters**.

The following standalone words are replaced with `****`, case-insensitively:

- `kerfuffle`
- `sharbert`
- `fornax`

Punctuation changes the word match, so `kerfuffle!` is not replaced.

**Success**

```text
201 Created
```

```json
{
  "id": "94b7e44c-3604-42e3-bef7-ebfcc3efff8f",
  "created_at": "2026-11-04T10:00:00Z",
  "updated_at": "2026-11-04T10:00:00Z",
  "body": "Hello, world!",
  "user_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

**Errors**

```text
400 Bad Request
401 Unauthorized
```

---

## Get All Chirps

### `GET /api/chirps`

Returns chirps ordered by `created_at` in ascending order.

**Success**

```text
200 OK
```

```json
[
  {
    "id": "94b7e44c-3604-42e3-bef7-ebfcc3efff8f",
    "created_at": "2026-11-04T10:00:00Z",
    "updated_at": "2026-11-04T10:00:00Z",
    "body": "Hello, world!",
    "user_id": "123e4567-e89b-12d3-a456-426614174000"
  }
]
```

If there are no chirps:

```json
[]
```

### Filter by Author

The optional `author_id` query parameter filters chirps at the database level.

```text
GET /api/chirps?author_id=<user-uuid>
```

Example:

```text
GET /api/chirps?author_id=123e4567-e89b-12d3-a456-426614174000
```

Only chirps belonging to that user are returned.

Results remain sorted by `created_at` ascending.

---

## Get One Chirp

### `GET /api/chirps/{chirpID}`

Returns a single chirp by UUID.

Example:

```text
GET /api/chirps/94b7e44c-3604-42e3-bef7-ebfcc3efff8f
```

**Success**

```text
200 OK
```

```json
{
  "id": "94b7e44c-3604-42e3-bef7-ebfcc3efff8f",
  "created_at": "2026-11-04T10:00:00Z",
  "updated_at": "2026-11-04T10:00:00Z",
  "body": "Hello, world!",
  "user_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

**Not Found**

```text
404 Not Found
```

---

## Delete Chirp

### `DELETE /api/chirps/{chirpID}`

Deletes a chirp.

**Authentication**

```text
Authorization: Bearer <access-token>
```

Only the user who created the chirp may delete it.

**Success**

```text
204 No Content
```

**Errors**

```text
401 Unauthorized
```

Missing or invalid access token.

```text
403 Forbidden
```

The authenticated user does not own the chirp.

```text
404 Not Found
```

The chirp does not exist.

---

# Polka Webhooks

## Chirpy Red Upgrade

### `POST /api/polka/webhooks`

Receives webhook events from the Polka payment service.

**Authentication**

```text
Authorization: ApiKey <polka-api-key>
```

Requests with a missing or incorrect API key are rejected.

**User Upgrade Event**

```json
{
  "event": "user.upgraded",
  "data": {
    "user_id": "3311741c-680c-4546-99f3-fc9efac2036c"
  }
}
```

The matching user is updated to:

```text
is_chirpy_red = true
```

**Success**

```text
204 No Content
```

**User Not Found**

```text
404 Not Found
```

### Other Events

Events other than:

```text
user.upgraded
```

are ignored and return:

```text
204 No Content
```

The upgrade operation is idempotent. Sending the same `user.upgraded` event multiple times leaves the user in the same Chirpy Red state.

---

# Administrative Endpoints

## Metrics

### `GET /admin/metrics`

Returns an HTML page displaying the number of file-server requests handled since Chirpy started.

---

## Reset

### `POST /admin/reset`

Development-only endpoint.

The endpoint is available only when:

```text
PLATFORM="dev"
```

It:

- Deletes all users
- Cascades deletion to associated chirps and refresh tokens
- Resets the file-server hit counter

**Success**

```text
200 OK
```

If Chirpy is not running in the `dev` environment:

```text
403 Forbidden
```

---

# Website

### `/app/`

Serves Chirpy's static website and assets.

Example:

```text
GET /app/
GET /app/assets/logo.png
```

File-server requests are counted by the metrics middleware.

---

# Authentication Summary

Chirpy uses three authentication mechanisms.

| Purpose | Authorization Header |
|---|---|
| Authenticated Chirpy API request | `Bearer <JWT access token>` |
| Refresh/revoke session | `Bearer <refresh token>` |
| Polka webhook | `ApiKey <Polka API key>` |

Access tokens expire after **1 hour**.

Refresh tokens expire after **60 days** and may be revoked early.

Passwords are hashed with **Argon2id** and plaintext passwords are never stored.