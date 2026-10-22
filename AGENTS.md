# Kura V4

- Go + Fiber, embedded React/Vite SPA, TanStack Router/Query, PostgreSQL.
- Preserve the layer structure: controllers -> domain services -> repositories -> DAOs. Controllers own HTTP; services own business rules; repositories own transactions; DAOs own SQL. Wire dependencies explicitly in internal/app.
- Frontend render components call hooks; hooks call API adapters. Pure domain logic imports neither React nor transport. Use CSS Modules and semantic CSS variables.
- Go DTOs and frontend/src/types/api.ts are handwritten: update both when the API changes. Do not add OpenAPI/codegen for the API or an ORM.
- pnpm only. Root Makefile is the developer entry point. No Playwright.
- Run make check for changes spanning the scaffold. Database integration tests use disposable testcontainers, never an existing library database.
- Run make generate after changing queries/schema or mocked interfaces. Never edit generated SQL bindings or mocks by hand.
- Preserve originals, credentials, .env, runtime databases and media. Do not commit them. No automatic original deletion or transformation.
- Session cookies and PostgreSQL-backed sessions are the agreed authentication design; account flows have not been built yet. Public read access must never imply anonymous write access.
- New work branches use codex/. Do not introduce extra infrastructure without a concrete feature need.
