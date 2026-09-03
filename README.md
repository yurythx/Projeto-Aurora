# Projeto Aurora — Plataforma Enterprise Base Governamental

O **Projeto Aurora** é uma plataforma corporativa modular em Go (Backend) e Next.js App Router (Frontend), desenvolvida para servir como **base enterprise genérica em estrita conformidade com as normas do Governo Federal Brasileiro** (SGD/MGI, e-MAG 2.0, DSGov, LGPD, Gov.br e OWASP) para o rápido desenvolvimento e desacoplamento de novas aplicações corporativas e módulos municipais.

Ele fornece toda a infraestrutura pronta de segurança HTTP, mensageria, outbox transacional, autenticação OIDC Gov.br / Local, mascaramento PII, auditoria imutável, suporte a motor de busca e um Design System governamental oficial (DSGov / e-MAG).

---

## 🏛️ Conformidade Governamental (SGD/MGI) & Recursos Globais

O Projeto Aurora atende integralmente aos 5 módulos de conformidade exigidos pela Secretaria de Governo Digital (SGD/MGI):

1. **Módulo 1: Segurança HTTP & Headers Defensivos (Go):** Headers OWASP (`HSTS`, `CSP com Nonce`, `X-Frame DENY`, `X-Content-Type nosniff`), rate-limiting e timeouts de servidor anti-DoS.
2. **Módulo 2: Privacidade & LGPD (Go):** Mascaramento nativo de PII (`slog.LogValuer` em CPF, e-mail, telefone) e rastreabilidade correlacionada (`X-Request-ID`).
3. **Módulo 3: Autenticação Federada & OIDC Gov.br (Go + Next.js):** Validador JWKS com mapeamento dos Níveis de Confiabilidade (`Bronze`, `Prata`, `Ouro` - Portaria SGD/SEDGG Nº 2.154) e RESTful e-PING.
4. **Módulo 4: Acessibilidade Digital e-MAG (Next.js):** Conformidade e-MAG 2.0 / WCAG 2.1 AA com barra de atalhos por teclado (Alt+1..4), VLibras nativo, alto contraste e-MAG e redimensionamento de fonte (A+/A-/A).
5. **Módulo 5: Identidade Visual Governamental DSGov (Next.js):** Design System oficial GovBR-DS, rodapé unificado com canais de atendimento, LGPD/LAI e suporte White-Label dinâmico.

---

## 🏗️ Arquitetura e Recursos Prontos

- **Monólito Modular & Clean Architecture:** Estrutura pronta para acoplamento de módulos em `internal/modules/`.
- **Módulo Modelo Template (`example`):** Exemplo completo de módulo com camadas Clean Architecture (`domain`, `application`, `infrastructure`, `transport`).
- **Autenticação Dupla:** Suporte a Keycloak SSO / Gov.br (OIDC) e autenticação local com chaves RSA / bcrypt e rate limiting.
- **Outbox Transacional & RabbitMQ:** Escrita atômica no PostgreSQL e publicação assíncrona no RabbitMQ com filas Dead-Letter Queue (DLQ).
- **Auditoria Imutável:** Registros de auditoria append-only protegidos no nível do PostgreSQL.
- **WebSocket Server:** Broker WebSocket para retransmissão de notificações em tempo real.
- **Object Storage (MinIO S3):** Armazenamento de arquivos via URLs pré-assinadas.
- **Motor de Busca (Typesense 27+):** Motor de busca ultrarrápido configurado para consumo seguro via chave restrita.

---

## 📂 Estrutura do Repositório

```text
.
├── backend/                   # ⚙️ Backend (Go 1.25+)
│   ├── cmd/
│   │   ├── api/               # API REST e Servidor WebSocket
│   │   ├── worker/            # Processador background RabbitMQ / Outbox
│   │   └── seedadmin/         # CLI para semente de usuário administrador local
│   ├── internal/
│   │   ├── app/               # Injeção central de dependências e roteamento
│   │   ├── domain/            # Tipos e erros primitivos de domínio
│   │   ├── platform/          # Infraestrutura compartilhada (Auth, DB, Messaging, Outbox, Audit, WS)
│   │   └── modules/           # Módulos de Negócio
│   │       └── example/       # Módulo Modelo / Template Genérico
│   ├── migrations/            # Scripts de schema PostgreSQL (Goose)
│   └── pkg/                   # Utilitários genéricos (httputil)
├── frontend/                  # 🎨 Frontend (Next.js / TypeScript / React)
│   ├── src/
│   │   ├── app/               # Routes App Router Next.js
│   │   │   ├── (protected)/   # Rotas protegidas (Dashboard, Integrações, Configurações)
│   │   │   ├── login/         # Login
│   │   │   └── sobre/         # Documentação da Plataforma
│   │   ├── components/        # Design System (ui, layout, branding, notifications)
│   │   ├── hooks/             # Custom React Hooks
│   │   ├── lib/               # Clientes API / WS / Auth
│   │   └── types/             # TypeScript DTOs
├── docker-compose.yml         # Serviços Docker (PostgreSQL, RabbitMQ, MinIO, Typesense, API, Worker, Frontend)
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
