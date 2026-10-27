# Kura V4

A Go/Fiber monolith with an embedded React + TypeScript frontend. The project
follows the layered conventions explored in
[VictoriqueMoe/umineko_city_of_books](https://github.com/VictoriqueMoe/umineko_city_of_books),
with TanStack Router/Query, CSS Modules, PostgreSQL, sqlc and goose.

## Start developing

Install Go **1.27.1**, Node **24.22.1**, pnpm **12.15.0**, and Docker.

```sh
make setup
make dev
```

Open http://localhost:5174. `make dev` starts a dedicated PostgreSQL container,
applies migrations, and starts Go (Air reload) plus Vite. Ctrl-C stops both app
processes; PostgreSQL remains available. `make db-down` stops it without deleting
data. Development uses database port **56432**, API port **8094**, and frontend
port **5174**. These are separate from Kura V3.

Use the `localhost` name, not `127.0.0.1`: browsers only allow passkeys on a
domain name, so the app redirects loopback-IP visits to `localhost`, and the
server refuses passkey ceremonies begun at an IP address with that guidance.

`make setup` copies `.env.example` only if `.env` does not exist. The checked-in
password is for a loopback-only disposable development database; replace it for
a VPS. Keep `POSTGRES_*` and `DATABASE_URL` consistent. PostgreSQL bootstrap
credentials only take effect on first initialization of its volume.

## Commands

| Command | Purpose |
| --- | --- |
| `make dev` | Database, migrations, Go reload and Vite |
| `make check` | Formatting, lint, layers, types, generation drift, all tests, Compose validation and production build |
| `make test-go` | Go unit/HTTP/storage tests, with race detection |
| `make test-integration` | A temporary PostgreSQL container: migrations and job lease/retry behavior |
| `make test-web` | Vitest + React Testing Library |
| `make generate` | Regenerate sqlc bindings and mockery mocks |
| `make migration name=description` | Create a goose migration |
| `node --env-file=.env scripts/run.mjs go run -tags dev ./cmd/kura promote <username> <role>` | Set a role from the operator shell (bootstrap the first super admin) |
| `make migrate` | Apply migrations to the configured database |
| `node --env-file=.env scripts/run.mjs go run -tags dev ./cmd/kura seed <dir> [dir...]` | Import sample picture folders as board posts (duplicates skipped) |
| `make format` | Format Go and frontend source |
| `make build` | Build Vite and embed its assets into `bin/kura` |

Go tools are pinned in `go.mod`; pnpm dependencies are pinned in the lockfile.
Use `pnpm`, not npm/yarn. Generation checks compare file contents and never
stage changes.

The `dev` build tag lets Go tests run without a frontend build. A production
build intentionally requires the real frontend output; `make build` supplies it.

## What is implemented

- React image-board shell, CSS Modules and semantic theme variables, with four
  themes (Kura Dark, Safebooru, Rule34, Umineko) picked from the nav.
- Accounts with passwords, passkeys (WebAuthn) and recovery keys, backed by
  PostgreSQL sessions with HttpOnly cookies. Public reads never imply write
  access: uploads, favorites, lists and tagging require sign-in.
- Uploads (images, GIFs, MP4/WebM) with content-hash deduplication, thumbnails
  and dimension probing. Posts have no title; they are found by tags and the
  artist credit. The upload dialog closes itself on success and reports that
  new pictures land at the top of the board and in **Me → Contributions**,
  which lists everything the signed-in user has added.
- SFW/NSFW rating per post with server-side gating: NSFW posts are hidden
  from anonymous visitors and from accounts until they opt in with an 18+
  confirmation in Me → Sign-in & security.
- Tag search with suggestions, favorites, public/private lists, profiles with
  contribution counts, and a `seed` command that imports sample folders.
- Roles and change history (below): staff tiers with an append-only audit
  log and one-click rollback for post and role changes.
- A real `GET /api/v1/site` route reading site settings through every backend
  layer; frontend loading, connection-error and retry states.
- Separate `/health/live` and `/health/ready` endpoints and consistent API errors
  with request identifiers. Unknown API/media paths cannot fall through to the SPA.
- PostgreSQL migrations, generated query bindings, interface mocks and layer checks.
- Durable job persistence: atomic claims, expiring leases, bounded retries,
  ownership tokens and rejection of stale completion attempts. The worker runner
  is wired into app lifecycle but remains idle until feature handlers are registered.
- Local byte-preserving storage with opaque keys, atomic publication and failed-write
  cleanup. Storage is not exposed as a public static directory.
- VPS Compose configuration, embedded-frontend production build, CI and graceful shutdown.

## Responsive layout

The UI is designed for phone, tablet and desktop widths:

- **Nav** stays on one compact row at every width. Long display names truncate
  with an ellipsis next to the avatar; the theme picker shrinks on phones, and
  “Create account” drops off only below 380px (it remains on the sign-in page).
- **Board** shows two thumbnails per row on phones. The sticky search bar takes
  the full row on tablets (≤900px) with layout controls wrapping beneath it;
  on phones the desktop column/width pickers hide, leaving search and Upload.
  There is no horizontal scrolling at 390px, 768px or 820px.
- **Me, posts and forms** use full-width fields on small screens, the Me tabs
  become an icon-only scrollable row, post tag forms stretch instead of
  truncating, and form text is 16px on phones so iOS Safari does not zoom on
  focus. The upload dialog becomes a bottom sheet with full-width actions.

## Code map

```text
cmd/kura/                 serve, migrate, seed and audit commands
internal/app/             dependency construction and lifecycle
internal/controllers/     HTTP parsing and responses
internal/accounts/        accounts, passkeys, sessions and recovery keys
internal/board/           posts, uploads, thumbnails and search index
internal/collections/     favorites and lists
internal/site/            site settings behavior
internal/jobs/            worker behavior
internal/repository/      persistence composition and transactions
internal/dao/             SQL access and generated sqlc bindings
internal/db/              database connection and embedded migrations
internal/model/           domain/read models
internal/dto/             handwritten API contracts
internal/storage/         local storage adapter and storage contract
internal/webui/           production frontend embedding
frontend/src/components/  reusable rendering
frontend/src/pages/       page rendering
frontend/src/routes/      TanStack route composition
frontend/src/hooks/       queries, mutations and orchestration
frontend/src/api/         HTTP adapter and query-key definitions
frontend/src/types/       handwritten TypeScript API contracts
frontend/src/styles/      global reset and semantic theme variables
```

Controllers call services; services receive persistence interfaces; repositories
compose DAO operations. Fiber stays at the HTTP/application boundary. Domain
services must not contain SQL. Repositories own multi-statement transactions.
`internal/app` explicitly constructs the dependencies; no DI container is used.
Frontend pages/components use hooks, and transport stays in `api`. Pure domain
logic belongs in `frontend/src/domain` when needed. CSS Modules sit beside their
components. Additional themes will override the semantic CSS variables.

Go API DTOs and `frontend/src/types/api.ts` are maintained together by hand.
Changing one requires reviewing the other and the HTTP/client tests. sqlc output
is a database contract, not a public response schema. Never edit generated files.

## Jobs and storage

Each job handler must be idempotent: delivery is at least once. The initial runner
uses a 45-second handler deadline and 60-second lease; handlers must honor context
cancellation. Add lease renewal before introducing longer-running transcodes.
Only registered job kinds are claimed. Exhausted expired leases become failed;
failed jobs are not silently retried forever. No job-management UI exists yet.

The local adapter writes bytes unchanged under opaque UUID keys. It never guesses
media visibility from a file path. Future media routes must check access before
opening a stored object. `.partial` files can remain after process/power loss;
a future age-bounded staging cleanup must distinguish abandoned and active writes.

## VPS deployment

Copy the repository to the VPS and create an untracked `.env`. Set a strong
`POSTGRES_PASSWORD`, matching `DATABASE_URL` with **host `postgres`, port `5432`**,
`KURA_DOMAIN` to your domain, and `KURA_PUBLIC_URL=https://your-domain`.
URL-encode any reserved characters in the database password used inside the URL.
Point DNS to the VPS and allow inbound TCP 80/443 (UDP 443 is optional for HTTP/3).

```sh
docker compose -f compose.prod.yml build
docker compose -f compose.prod.yml up -d --wait postgres
docker compose -f compose.prod.yml run --rm migrate
docker compose -f compose.prod.yml up -d app caddy
```

Caddy terminates HTTPS; the Go app and database have no published host ports.
The application runs as an unprivileged user with a read-only root filesystem;
its writable media volume and temporary directory are separate. The image contains
FFmpeg for future media jobs. PostgreSQL, media and Caddy state use named volumes.
Updates replace the application image without replacing those volumes. Do not run
`docker compose down -v` against data you want to retain.

Migrations are an explicit deployment step. A failed migration must stop the
upgrade. Back up before schema changes; a down migration can discard data.

## Backup and restore

Before real media is imported, establish scheduled, encrypted off-host backups and
perform a restore drill. PostgreSQL and original media are both required; saving
one alone is insufficient. For a consistent simple backup, stop the app, dump
PostgreSQL, snapshot/copy the media volume, and restart the app. Keep the deployment
configuration and required secrets in a separate protected backup.

Restore into a separate Compose project/volumes: restore PostgreSQL and media from
the same backup set, use the matching application version, and check database
readiness plus original-byte integrity before opening access. Automated backup
scripts and an operator restore drill are not part of this initial scaffold.

## Roles

Visitor is anonymous (no account): SFW board, search and public profiles/lists
only — no uploads, tags, favorites or NSFW. Signed-in accounts hold one stored
role:

| Role | Can do (in addition to the tier below) |
| --- | --- |
| Member | Upload, communal tagging, favorites, lists, own profile. The default for new accounts. |
| Moderator | Fix tags/artist/rating on any post, hide posts from the board, roll back content changes. No permanent deletion, no user management. |
| Admin | Promote/demote Member <-> Moderator, delete posts (soft — restorable from history), roll back role and content changes. Cannot touch Admins or Super Admins. |
| Super Admin | Everything an Admin can do, plus create/promote/demote Admins and other Super Admins. At least one Super Admin must always exist. |

Public read access never implies write access, and every staff check is
enforced server-side (the UI only hides what the API would reject). Bootstrap
the first Super Admin from the operator shell — shell/DB access is already
operator trust, and the promotion is still written to history as `cli:promote`:

```sh
node --env-file=.env scripts/run.mjs go run -tags dev ./cmd/kura promote <username> super_admin
```

After that, Super Admins and Admins manage roles in the app under **Staff**
(`/admin`, visible to Moderators and above). Staff review everything that is
off the board under **Staff → Hidden & deleted** (`/admin/hidden`), split into
hidden vs deleted, with one-click Restore.

On a post page, tag/artist/rating edits are staged locally and written by a
single **Save changes** (one history entry per save, not per tag), the picture
itself and its **Full size** link open the raw file in a new tab, and staff
actions are color-coded: Hide in warning amber, Delete in danger red, Restore
in green.

## Change history and rollback

Every change to a post entry (creation, tag/artist/rating edits, hide, unhide,
soft-delete) and every role change writes an append-only `change_history` row:
actor, time, action, and the full before/after snapshot (post:
artist/tags/rating/hidden; user: role). Rows are never edited or deleted.

Rollback restores one row's before snapshot and writes a new `rollback` row
pointing at it, so undoing a change — a rogue moderator's tag spree, a mistaken
approval, a role change you reconsider — never erases the evidence, and a
rollback can itself be rolled back (redo). Moderators and above roll back post
changes; role rollbacks re-check the same hierarchy as a fresh role change.
Find history on each post page (any signed-in viewer) and the staff-wide feed
under **Staff**. Hides and deletes are soft: original bytes are always
preserved, so a hidden/deleted post can always be brought back.
