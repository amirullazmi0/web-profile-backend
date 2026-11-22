# Web Profile Backend

Go REST API with authentication built in.

## Stack

- **Router**: Gin (`gin-contrib/cors`, `gin-contrib/requestid`)
- **Database**: PostgreSQL via `database/sql` + `lib/pq` and the query builder in `pkg/database`
- **Migrations**: Prisma (schema and migrations only, no Prisma client in Go)
- **Cache / rate limit / tokens**: Redis (`go-redis/v9`)
- **Auth**: JWT HS256 (access + refresh, `typ` claim), bcrypt, httpOnly cookies + JSON body
- **Validation**: `go-playground/validator/v10`
- **Logging**: zap (console on a terminal, JSON in containers)
- **Docs**: Swagger via swag (`/swagger/index.html`)
- **Deploy**: Docker + CapRover, GitHub Actions CI/CD

## Getting started

```bash
cp .env.example .env          # already done by the scaffolder
docker compose up -d          # Postgres + Redis
npm install                   # Prisma CLI
npm run migrate:dev           # apply prisma/migrations
npm run db:seed               # create the superadmin from SEED_SUPERADMIN_*
make dev                      # air hot reload (go install github.com/air-verse/air@latest)
```

API: `http://localhost:8080/api` · Health: `/health` · Swagger: `/swagger/index.html`

## Project structure

```
main.go                     entrypoint (-seed, -healthcheck flags)
config/                     env loading (viper)
internal/
  handler/                  Gin handlers + routes.go (dependency wiring)
  usecase/                  business logic (interface + struct)
  repository/               SQL data access
  dto/                      request/response structs with validate tags
  model/                    DB row structs
  middleware/               auth (JWT), role, rate limit, logger, recovery
pkg/
  database/                 connection pool, query builder, seeder
  email/                    SMTP + embedded HTML templates
  logger/ redis/ response/ validator/
prisma/                     schema.prisma + migrations
templates/verify-email.html page rendered by GET /api/auth/verify-email
```

To add a feature, follow `.agent/workflows/add-feature.md`: model → dto → repository → usecase → handler → register in `routes.go`.

## Auth endpoints

Every response uses `{ success, message, data?, meta?, error? }`.

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/auth/register` | Creates an inactive user and emails a verification link |
| GET | `/api/auth/verify-email?token=` | HTML page; activates the account |
| POST | `/api/auth/resend-verification` | `{ email }` |
| POST | `/api/auth/login` | Returns `{ user, access_token, refresh_token, expires_in }` and sets cookies |
| POST | `/api/auth/refresh` | `{ refresh_token }` body or the `refresh_token` cookie; rotates the token |
| POST | `/api/auth/logout` | Always succeeds, clears cookies and the stored refresh token |
| POST | `/api/auth/forgot-password` | `{ email }` |
| POST | `/api/auth/reset-password` | `{ token, new_password }` |
| GET/PUT | `/api/users/profile` | Authenticated |
| PUT | `/api/users/change-password` | `{ old_password, new_password }` |
| GET | `/api/users` | SUPERADMIN, paginated (`page`, `limit`, `search`, `sort`, `order`) |
| POST | `/api/users/create-admin` | SUPERADMIN |
| PUT | `/api/users/update-admin`, `/update-password-admin/:id`, `/update-role-admin/:id`, `/set-active-admin/:id` | SUPERADMIN |
| DELETE | `/api/users/:id` | SUPERADMIN, soft delete |

Roles: `SUPERADMIN`, `ADMIN`, `USER`. Public auth routes are rate limited per IP through Redis.

## CI/CD

- `.github/workflows/ci.yml` runs on PRs and pushes to `main`, `staging`, `development`: go mod tidy check, vet, build, `test -race`, swagger freshness, Prisma validate/format, govulncheck.
- A push to `staging` deploys to the `staging` GitHub Environment, a push to `main` to `production`, through `.github/workflows/deploy.yml` (CapRover).
- Create both GitHub Environments with secrets `CAPROVER_SERVER`, `CAPROVER_TOKEN` (app token) and `CAPROVER_APP_NAME`.
- On CapRover, set the env vars from `.env.example` on the app and run `npm run migrate:deploy` against the production database before the first deploy.

Manual deploy from a laptop: `npm run deploy:vps`.
