# MISSION DIRECTIVE: PROJETO NOVA (ENTERPRISE CONTRACT GOVERNANCE & GAZETTE HR INTELLIGENCE)

You are a Principal Software Architect and Senior Systems Engineer.
Your mission is to scaffold, implement, and deliver the production-ready code for **Projeto Nova**, an enterprise-grade contract fiscalization platform and Official Gazette (DIORONDON) intelligence engine tailored for the Municipal Administration of Rondonópolis.

The system enforces Clean Architecture in Go (Backend), Next.js with App Router and Tailwind CSS (Frontend), PostgreSQL 16 (Relational Source of Truth), Typesense 27+ (Search & Gazette Analytics Engine), RabbitMQ (Event Messaging with DLX), and MinIO (S3 Document Storage).

---

## 1. ARCHITECTURAL PILLARS & CONSTRAINTS

### Backend (Go 1.22+)
- **Modular Monolith & Clean Architecture:** Strict decoupling between layers (`delivery` -> `service` -> `repository` -> `domain`).
- **Agnostic Core Framework (`pkg/`):** Packages inside `pkg/` (`database`, `outbox`, `eventbus`, `typesense`, `audit`, `websocket`, `idempotency`, `storage`) must NEVER import from `internal/`.
- **Database Access:** Use `jackc/pgx/v5` with `pgxpool`. Zero SQL string concatenation (strictly parameterized queries).
- **Transactional Outbox:** Any domain mutation recording an event MUST write to `outbox_events` in the same database transaction (`Unit of Work`).
- **Outbox Dispatcher:** Worker listening to PostgreSQL `LISTEN/NOTIFY` (channel: `nova_outbox_channel`) to publish to RabbitMQ with zero polling latency.
- **Search & Ingestion (Typesense Engine):**
  - Download and parse DIORONDON PDF editions, verify binary SHA-256 hash (strict idempotency), detect ordinary vs supplementary/extra editions, and extract clean text.
  - Index general articles/contracts into `diorondon_articles`.
  - Extract and index all functional personnel movements into `diorondon_personnel_acts` (Appointments, Dismissals, Reallocations, Roles, DAS/Salary, Secretarias, Names, CPFs, Matrículas).

### Frontend (Next.js 14+ / TypeScript)
- **App Router & Strict Types:** Fully typed, Dark Mode native support via Tailwind CSS.
- **Kanban Board:** Multi-stage Kanban with `@dnd-kit/core`, optimistic UI updates (SWR/Zustand), and strict column movement guardrails.
- **HR & Gazette Search Panel:** Instant search interface querying Typesense collections with typo tolerance, faceted filtering, and highlighted text snippets.
- **Document Hub:** Integrated inline PDF viewer and automated ZIP/Package download for supplier dispatch.

---

## 2. THE 6-STAGE KANBAN WORKFLOW & BUSINESS RULES

The system enforces a 6-stage lifecycle for monthly contract demands:

```text
[Etapa 1: Elaborar OF / Pré-Empenho] ──► [Etapa 2: Tramitar Planejamento (Espera Externa)]
                                                    │
[Etapa 4: Execução e Recepção] ◄── [Etapa 3: Emitir OS / Envio Empresa] ◄┘
       │
       ▼
[Etapa 5: Relatório Pgto / Certidões] ──► [Etapa 6: Contabilidade (Liquidação / Arquivado)]
```

### STAGE 1: ELABORAR OF / PRÉ-EMPENHO (Fiscal Action)
- **Inputs:** Select items and quantities from contract catalog.
- **Auto-Generation:** Official Request to Planning Office (Ofício ao Planejamento) PDF.
- **Required Attachments:** `ORDEM_FORNECIMENTO` (OF), `PRE_EMPENHO`, `OFICIO_PLANEJAMENTO`.
- **Guardrail:** Blocked from advancing to Stage 2 if any of the 3 documents is missing.

### STAGE 2: TRAMITAR PLANEJAMENTO / CONTABILIDADE (External Wait State)
- **Passive tracking:** Records departure timestamp and tracks SLA while awaiting the signed commitment note (Empenho) from accounting.
- **Required Attachments:** None.

### STAGE 3: EMITIR OS / ENVIO À EMPRESA (Fiscal Action)
- **Inputs:** Accounting returns signed commitment note (`EMPENHO_ASSINADO`).
- **Auto-Generation:** Official Service Order (Ofício de Ordem de Serviço) PDF following the standard municipal layout (Header, Item Table, Brand, Quantities, Unit/Total Values, Signature block).
- **Package Compiler:** One-click bundle combining OF + Pré-Empenho + Empenho + OS into a downloadable ZIP or unified PDF.
- **Guardrail:** Blocked until `EMPENHO_ASSINADO` is uploaded and supplier dispatch confirmation (`os_envio_enviado = true`) is checked.

### STAGE 4: EXECUÇÃO E RECEPÇÃO (Fiscal Action)
- **Action:** Supplier executes service or delivers goods.
- **Required Attachments:** `NOTA_FISCAL` (NF) and `ORDEM_RECEPCAO_AGILE` (Legacy System Reception Slip).
- **Guardrail:** Blocked until both files are uploaded.

### STAGE 5: RELATÓRIO DE PAGAMENTO / COMPLIANCE & CERTIDÕES (Fiscal Action)
- **Action:** Compliance verification, withholding calculations, and official attestation.
- **Auto-Generation:** Official Monthly Fiscalization Report (Anexo I - IN SCL 01/2019) with standardized electronic stamp (Fiscal Name, Matrícula, Portaria/Data, Contrato, Signature).
- **Required Attachments:** `EXTRATO_EMPENHO_AGILE`, `RELATORIO_PAGAMENTO_GERADO`, and the 6 Mandatory Compliance Certificates (`CERT_SIMPLES`, `CERT_CNDT`, `CERT_FGTS`, `CERT_MUNICIPAL`, `CERT_ESTADUAL`, `CERT_FEDERAL`).
- **Special Rule (Service Contracts):** Cumulatively requires `PLANILHA_MEDICAO` and `BOLETO_DAM_ISSQN`.
- **Guardrail:** Card is locked if any required certificate is expired/missing or if service attachments are incomplete.

### STAGE 6: CONTABILIDADE / LIQUIDAÇÃO & PAGAMENTO (External Wait / Completion)
- **Passive tracking:** Physical/digital file delivered to Accounting for liquidation.
- **Completion:** Fiscal marks demand as `PAGO_ARQUIVADO` once bank payment is confirmed.

---

## 3. TYPESENSE SEARCH SCHEMAS (PERSONNEL & GAZETTE)

The Typesense client must initialize and maintain two collections:

### Collection 1: `diorondon_articles` (General Gazette Content)
- `id` (string)
- `edition_number` (int32, facet)
- `edition_type` (string, facet - ORDINARIA/SUPLEMENTAR)
- `publication_date` (int64, facet)
- `page_number` (int32)
- `contract_numbers` (string[], facet)
- `cnpjs` (string[], facet)
- `officials_named` (string[], facet)
- `content` (string)

### Collection 2: `diorondon_personnel_acts` (HR & Appointments Intelligence)
```json
{
  "name": "diorondon_personnel_acts",
  "fields": [
    { "name": "id", "type": "string" },
    { "name": "edition_number", "type": "int32", "facet": true },
    { "name": "edition_type", "type": "string", "facet": true },
    { "name": "publication_date", "type": "int64", "facet": true },
    { "name": "act_type", "type": "string", "facet": true },
    { "name": "person_name", "type": "string", "facet": true },
    { "name": "person_cpf", "type": "string", "facet": true, "optional": true },
    { "name": "person_matricula", "type": "string", "facet": true, "optional": true },
    { "name": "job_role", "type": "string", "facet": true },
    { "name": "secretaria", "type": "string", "facet": true },
    { "name": "das_level", "type": "string", "facet": true, "optional": true },
    { "name": "salary_value", "type": "float", "facet": true, "optional": true },
    { "name": "portaria_number", "type": "string", "facet": true, "optional": true },
    { "name": "full_act_text", "type": "string" },
    { "name": "pdf_page_number", "type": "int32" },
    { "name": "pdf_storage_url", "type": "string" }
  ],
  "default_sorting_field": "publication_date"
}
```
Supported `act_type` values: `NOMEACAO_EFETIVO`, `NOMEACAO_COMISSIONADO`, `CONTRATACAO_TEMPORARIA`, `EXONERACAO`, `RESCISAO`, `RELOTACAO`, `DESIGNACAO_FUNCAO`.

---

## 4. REPOSITORY STRUCTURE

```text
.
├── Makefile
├── README.md
├── docker-compose.yml
├── migrations/
│   └── 000001_init_nova_schema.up.sql
├── backend/
│   ├── go.mod
│   ├── cmd/
│   │   ├── api/
│   │   │   └── main.go
│   │   └── worker/
│   │       └── main.go
│   ├── pkg/
│   │   ├── audit/
│   │   ├── database/
│   │   ├── eventbus/
│   │   ├── idempotency/
│   │   ├── outbox/
│   │   ├── storage/
│   │   ├── typesense/
│   │   └── websocket/
│   └── internal/
│       ├── contracts/
│       ├── documents/
│       └── gazette/
└── frontend/
    ├── package.json
    ├── tsconfig.json
    ├── tailwind.config.ts
    └── src/
        ├── app/
        ├── components/
        ├── hooks/
        └── lib/
```
