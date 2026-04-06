# tree-time-backend

REST backend for `tree-time-app`.

## Stack
- Go
- [chi](https://github.com/go-chi/chi)
- PostgreSQL
- pgx (repository layer)
- goose (migrations)
- JWT auth

## Run
1. Start Postgres on `localhost:15432`.
2. Set env vars if needed:
   - `DB_URL` (default: `postgres://postgres:postgres@localhost:15432/postgres?sslmode=disable`)
   - `HTTP_ADDR` (default: `:8080`)
   - `JWT_SECRET` (default: `tree-time-secret`)
3. Apply migrations:
   - `go run ./cmd/migrate`
4. Run server:
   - `go run ./cmd/server`

## API
- `POST /api/user/registration`
- `POST /api/user/login`
- `POST /api/logout`
- `GET /api/block-sessions` (Bearer token)
- `POST /api/block-sessions/start` (Bearer token)
- `POST /api/block-sessions/finish` (Bearer token)
