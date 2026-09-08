# Projeto Aurora — Plataforma Enterprise Base Governamental

O **Projeto Aurora** é uma plataforma corporativa modular construída segundo o padrão de **Arquitetura Microkernel (Plug-in Architecture)** em Go (Backend) e Next.js App Router (Frontend), desenvolvida para servir como **base enterprise genérica em estrita conformidade com as normas do Governo Federal Brasileiro** (SGD/MGI, e-MAG 2.0, DSGov, LGPD, Gov.br e OWASP) para o rápido desenvolvimento e desacoplamento de novos módulos de negócio municipais e estaduais.

Ele fornece um **Core System (Kernel)** robusto com infraestrutura pronta de segurança HTTP, mensageria, outbox transacional, autenticação OIDC Gov.br / Local, mascaramento PII, auditoria imutável, busca full-text em PostgreSQL e um Design System governamental oficial (DSGov / e-MAG).

---

## 🏛️ Conformidade Governamental (SGD/MGI) & Recursos Globais

O Projeto Aurora atende integralmente aos 5 módulos de conformidade exigidos pela Secretaria de Governo Digital (SGD/MGI):

1. **Módulo 1: Segurança HTTP & Headers Defensivos (Go):** Headers OWASP (`HSTS`, `CSP com Nonce`, `X-Frame DENY`, `X-Content-Type nosniff`), rate-limiting e timeouts de servidor anti-DoS.
2. **Módulo 2: Privacidade & LGPD (Go):** Mascaramento nativo de PII (`slog.LogValuer` em CPF, e-mail, telefone) e rastreabilidade correlacionada (`X-Request-ID`).
3. **Módulo 3: Autenticação Federada & OIDC Gov.br (Go + Next.js):** Validador JWKS com mapeamento dos Níveis de Confiabilidade (`Bronze`, `Prata`, `Ouro` - Portaria SGD/SEDGG Nº 2.154) e RESTful e-PING.
4. **Módulo 4: Acessibilidade Digital e-MAG (Next.js):** Conformidade e-MAG 2.0 / WCAG 2.1 AA com barra de atalhos por teclado (Alt+1..4), VLibras nativo, alto contraste e-MAG e redimensionamento de fonte (A+/A-/A).
5. **Módulo 5: Identidade Visual Governamental DSGov (Next.js):** Design System oficial GovBR-DS, rodapé unificado com canais de atendimento, LGPD/LAI e suporte White-Label dinâmico.

---

## 🏗️ Arquitetura Microkernel e Recursos Prontos

- **Arquitetura Microkernel (Core System + Plug-ins):** O Kernel central (`internal/platform`) gerencia a infraestrutura, resiliência e segurança, enquanto novos módulos de negócio (`internal/modules/`) funcionam como plug-ins isolados e desacoplados.
- **Clean Architecture nos Módulos:** Módulos de negócio com separação estrita de camadas (`domain`, `application`, `infrastructure`, `transport`).
- **Módulo Modelo Template (`example`):** Blueprint prático e de referência para novos plug-ins.
- **Autenticação Dupla:** Suporte a Keycloak SSO / Gov.br (OIDC) e autenticação local com chaves RSA / bcrypt e rate limiting.
- **Outbox Transacional & RabbitMQ:** Escrita atômica no PostgreSQL e publicação assíncrona no RabbitMQ com filas Dead-Letter Queue (DLQ).
- **Auditoria Imutável:** Registros de auditoria append-only protegidos no nível do PostgreSQL.
- **WebSocket Server:** Broker WebSocket para retransmissão de notificações em tempo real aos plug-ins.
- **Object Storage (MinIO S3):** Armazenamento de arquivos via URLs pré-assinadas.
- **Busca Full-Text Nativa:** Indexação e busca otimizada no PostgreSQL via `tsvector` e `pg_trgm`.

---

## 📂 Estrutura do Repositório

```text
.
├── backend/                   # ⚙️ Backend em Go (1.25+) — Arquitetura Microkernel
│   ├── cmd/
│   │   ├── api/               # API REST e Servidor WebSocket
│   │   ├── worker/            # Processador background RabbitMQ / Outbox
│   │   └── seedadmin/         # CLI para semente de usuário administrador local
│   ├── internal/
│   │   ├── app/               # Wiring central, injeção de dependências e roteamento
│   │   ├── domain/            # Tipos e erros primitivos de domínio
   │   ├── platform/          # 🛡️ Core System / Kernel (Auth, DB, Messaging, Outbox, Audit, WS)
│   │   └── modules/           # 🔌 Plug-ins / Módulos de Negócio (Users, Example, Integrations)
│   ├── migrations/            # Scripts de schema PostgreSQL (Goose)
│   └── pkg/                   # Utilitários genéricos (httputil)
├── frontend/                  # 🎨 Frontend (Next.js / TypeScript / React)
│   ├── src/
│   │   ├── app/               # Routes App Router Next.js
│   │   │   ├── (protected)/   # Rotas protegidas dos plug-ins (Dashboard, Integrações, Configurações)
│   │   │   ├── login/         # Login
│   │   │   └── sobre/         # Documentação da Plataforma
│   │   ├── components/        # Design System (ui, layout, branding, notifications)
│   │   ├── hooks/             # Custom React Hooks
│   │   ├── lib/               # Clientes API / WS / Auth
│   │   └── types/             # TypeScript DTOs
├── docker-compose.yml         # Serviços Docker (PostgreSQL, RabbitMQ, MinIO, API, Worker, Frontend)
├── docker-compose.dev.yml     # Exposição de portas em desenvolvimento
└── Makefile                   # Atalhos de build, testes, lint e migrations
```

---

## 🚀 Como Iniciar

### Pré-requisitos
- Docker & Docker Compose
- Go 1.25+ (opcional para rodar local fora do container)
- Node.js 20+ (opcional para rodar frontend fora do container)

### 1. Configurar Variáveis de Ambiente
```bash
cp .env.example .env
```

### 2. Gerar Chave RSA para Autenticação Local
```bash
mkdir -p secrets
openssl genrsa -out secrets/local_auth_private_key.pem 2048
chmod 644 secrets/local_auth_private_key.pem
```

### 3. Subir o Ambiente Docker
```bash
make dev
# Ou via docker compose:
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

### 4. Semear o Usuário Administrador Inicial
```bash
make seed-admin
```

### 5. Acessar a Aplicação
- **Frontend (Painel Aurora):** [http://localhost:3002](http://localhost:3002)
- **API Healthcheck:** [http://localhost:8002/health](http://localhost:8002/health)

---

## 🧪 Testes e Qualidade de Código

Para executar as suítes de teste automatizadas do backend e frontend:

```bash
# Executar todos os testes
make test

# Testes do Backend
cd backend && go test ./... -p 1

# Testes do Frontend
cd frontend && npm test
```

---

## 🛠️ Como Criar um Novo Módulo de Negócio

Para adicionar um novo módulo à aplicação (ex.: `patrimonio`):

1. **Scaffolding Automático:** Execute o script de geração de módulo:
   ```bash
   ./scripts/create-module.sh patrimonio
   ```
2. **Guia Completo:** Consulte a documentação detalhada da arquitetura em [`docs/GUIDE_MODULOS.md`](file:///home/adm.yuri@rondonopolis.local/Área de trabalho/Projetos/Projeto-Aurora/docs/GUIDE_MODULOS.md).
3. **Módulo Blueprint:** Utilize a implementação de referência em `backend/internal/modules/example/` e no menu **"Módulo Modelo"** (`/exemplos` no frontend).
