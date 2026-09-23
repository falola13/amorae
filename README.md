# Amorae

A monorepo template for a Next.js front end on a Go API, built so the first
feature and the fiftieth follow the same rules.

- **`apps/api`**: Go 1.26 modular monolith. Stdlib router, pgx, goose migrations,
  structured logs, Prometheus metrics, graceful shutdown.
- **`apps/web`**: Next.js 16 App Router as a backend-for-frontend. React Query over
  an axios client (through a same-origin `/api/v1` route handler), Zustand for
  client state, React Hook Form + Zod for forms, Server Actions for auth, an
  httpOnly session cookie, Tailwind 4. The full Amorae product UI, built from the
  design canvas.
- **Working vertical slice**: register → log in → view and edit profile → log out,
  with the auth, validation, error-handling and transaction patterns every later
  feature reuses.
- **Installable PWA**: manifest, full icon set, iOS splash screens for every
  current iPhone and iPad, and a service worker with an offline page that
  never caches personal data. See the brand guide in [`docs/BRAND.md`](docs/BRAND.md).

What to build next, and the gaps to close first, is
[`docs/ROADMAP.md`](docs/ROADMAP.md). What Amorae must do, and how well, is
specified in [`docs/requirements/`](docs/requirements/README.md): product requirements,
functional and non-functional requirements, the design specification, and the
register of decisions and open questions.

Read [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) before adding code. It covers
the dependency rule, how SOLID maps onto the files, and the scaling path. The
HTTP contract is in [`docs/API.md`](docs/API.md), and the decisions behind it
are in [`docs/adr/`](docs/adr).

## What works today

The Go API has auth, users and couple pairing; the prayer and Together
modules are still to come. The web app talks only to the real API, so a
screen whose endpoint doesn't exist yet says "Not available yet" where its
content would go, and the rest of the screen keeps working. Build a module in
Go and its screens come to life with no front-end flag to flip.

## Quick start (everything in Docker)

```bash
docker compose up --build
```

Then open http://localhost:3000. The API is on http://localhost:8088 and
Postgres on `localhost:5434` (user, password and database are all `amorae`).

## Local development (apps native, database in Docker)

Prerequisites: Go 1.26+, Node 24+, Docker.

```bash
npm run db                                # Postgres on :5434
cp apps/api/.env.example apps/api/.env
cp apps/web/.env.example apps/web/.env.local
npm run migrate                           # apply migrations
npm run dev:api                           # http://localhost:8088
npm --prefix apps/web install
npm run dev:web                           # http://localhost:3000 (second terminal)
```

## Scripts

Root `package.json` scripts work the same on Windows, macOS and Linux:

| Script | Does |
| --- | --- |
| `npm run up` / `down` | Full stack in Docker |
| `npm run db` | Start only Postgres |
| `npm run migrate` / `migrate:down` / `migrate:status` | goose, over the embedded migrations |
| `npm run dev:api` / `dev:web` | Run an app natively |
| `npm run test:api` | Go tests. Repository tests run only when `AMORAE_TEST_DATABASE_URL` is set |
| `npm run check` | vet + test + lint + typecheck + build: what CI runs, minus Docker |

To run the database-backed tests locally (with `npm run db` up):

```bash
AMORAE_TEST_DATABASE_URL="postgres://amorae:amorae@localhost:5434/amorae?sslmode=disable" npm run test:api
```

```powershell
$env:AMORAE_TEST_DATABASE_URL = "postgres://amorae:amorae@localhost:5434/amorae?sslmode=disable"; npm run test:api
```

## Layout

```
apps/
  api/        Go API: cmd/, internal/{app,config,platform,modules}, migrations/
  web/        Next.js: src/{app,features,lib,components}
docs/
  ROADMAP.md        where the product is, the gaps, and the order of work
  requirements/     what and how well: PRD, SRS, NFRs, design spec, decisions
  ARCHITECTURE.md   how and why
  API.md            HTTP contract
  adr/              architecture decision records
docker-compose.yml  db → migrate (one-shot) → api → web
.github/workflows/  CI: gofmt, vet, race tests against real Postgres, lint, typecheck, build, images
```

## Configuration

Each app documents its variables in its own `.env.example`:

- [`apps/api/.env.example`](apps/api/.env.example): `DATABASE_URL`, `HTTP_ADDR`,
  `SESSION_TTL`, `BCRYPT_COST`, logging, and so on. The API refuses to start on
  invalid config.
- [`apps/web/.env.example`](apps/web/.env.example): `API_URL`, `COOKIE_SECURE`
  and `BFF_SECRET`, all read at runtime on the server, so one image can serve
  every environment.

## Before production

The template leaves some decisions to you on purpose. See
*Deliberately not included* in the architecture doc. At minimum:

- Set the same random `BFF_SECRET` (`openssl rand -hex 32`) on the API and the
  web app, so the auth rate limit counts per visitor. The compose file's value
  is for local development only.
- Run the web app behind a reverse proxy that sets `X-Forwarded-For` (every
  host does). See *Rate limiting* in the architecture doc for why.
- The rate limiter is in-memory. Before running more than one API replica,
  swap in a shared (Redis) limiter.
- Don't publish `METRICS_ADDR`. Scrape it from inside your network.
- Put real secrets in your platform's secret store, not in `.env` files.
- Terminate TLS in front of the web app and leave `COOKIE_SECURE` at its
  production default (`true`).
