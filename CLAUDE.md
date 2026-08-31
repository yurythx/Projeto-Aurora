# MISSION DIRECTIVE: PROJETO AURORA (ENTERPRISE BASE PLATFORM & MODULAR ARCHITECTURE)

You are a Principal Software Architect and Senior Systems Engineer.
Your mission is to maintain and extend **Projeto Aurora**, an enterprise-grade base platform designed for rapid development and clean decoupling of custom business modules.

The system enforces Clean Architecture in Go (Backend), Next.js with App Router and Vanilla Tailwind CSS (Frontend), PostgreSQL 16 (Relational Source of Truth), Typesense 27+ (Search & Indexing Engine), RabbitMQ (Event Messaging with Outbox/DLQ), and MinIO (S3 Object Storage).

---

## 1. ARCHITECTURAL PILLARS & CONSTRAINTS

### Backend (Go 1.25+)
- **Modular Monolith & Clean Architecture:** Strict decoupling between domain modules in `internal/modules/`.
- **Module Structure (`internal/modules/{module_name}`):** Each module contains `domain/`, `application/`, `infrastructure/`, and `transport/`.
- **Module Registration (`internal/app/modules.go`):** All active domain modules are initialized and dependency-injected centrally.
- **Agnostic Core Platform (`pkg/` & `internal/platform/`):** Core platform capabilities (`auth`, `localauth`, `audit`, `database`, `outbox`, `messaging`, `ws`, `storage`, `ratelimit`, `idempotency`) are shared infrastructure.
- **Database Access:** Use `jackc/pgx/v5` with `pgxpool`. Zero SQL string concatenation (strictly parameterized queries).
- **Transactional Outbox:** Any domain mutation recording an event MUST write to `outbox_events` in the same database transaction (`Unit of Work`).
- **Outbox Dispatcher:** Worker listening to PostgreSQL `LISTEN/NOTIFY` channel to publish to RabbitMQ.

### Frontend (Next.js 16+ / TypeScript / App Router)
- **App Router & Strict Types:** Fully typed, Dark Mode native support, e-MAG accessibility compliance.
- **Generic Dashboard:** Centralized module launcher grid, real-time connectivity status, and platform health telemetry.
- **Design System (`src/components/ui`):** Modern typography, harmonious color palette, accessible interactive components.

---

## 2. HOW TO ADD A NEW BUSINESS MODULE TO PROJETO AURORA

To add a new business module (e.g. `pessoal` or `financas` or `atendimento`):

1. **Create Module Directory:**
   Create `internal/modules/{module_name}` following the template in `internal/modules/example`:
   - `domain/`: Entity structs, domain errors, domain events, interface contracts.
   - `application/`: Use case services / command & query handlers.
   - `infrastructure/`: Repository implementations (`pgxpool`) and external integrations.
   - `transport/`: HTTP handlers (`gin`) using `pkg/httputil` for response envelopes.

2. **Register Module Dependencies:**
   In `internal/app/modules.go`:
   - Define a `Module` struct holding your repositories and services.
   - Implement `NewModule(deps *Dependencies)` to construct dependencies.
   - Register HTTP routes in `RegisterRoutes(router *gin.RouterGroup)`.
   - Register background consumers or event handlers if applicable.

3. **Register HTTP Routes in Router:**
   In `internal/app/router.go`, call `module.RegisterRoutes(v1)`.

4. **Frontend Navigation:**
   Update `Sidebar.tsx` and `sectionTabs.ts` to add navigation items for the new module.

---

## 3. REPOSITORY STRUCTURE

```text
.
├── Makefile
├── README.md
├── docker-compose.yml
├── docker-compose.dev.yml
├── backend/
│   ├── go.mod
│   ├── cmd/
│   │   ├── api/
│   │   │   └── main.go
│   │   ├── worker/
│   │   │   └── main.go
│   │   └── seedadmin/
│   │       └── main.go
│   ├── migrations/
│   ├── pkg/
│   │   └── httputil/
│   └── internal/
│       ├── app/          # App bootstrap & central module wiring
│       ├── domain/       # Global domain primitives & errors
│       ├── platform/     # Infrastructure services (Auth, Outbox, RabbitMQ, DB)
│       └── modules/      # Business modules
│           └── example/  # Generic template module
└── frontend/
    ├── package.json
    ├── tsconfig.json
    ├── tailwind.config.ts
    └── src/
        ├── app/
        ├── components/
        ├── hooks/
        ├── lib/
        └── types/
```
