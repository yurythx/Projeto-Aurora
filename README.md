# Projeto Nova — Administração Municipal & Gestão de Contratos

Uma plataforma corporativa modular construída para automatizar o ecossistema da **Administração Municipal**, focada na ingestão, monitoramento e gestão de **Contratos e Fiscais de Contrato** através da captura automatizada do Diário Oficial.

Construída como um **Monólito Modular** (API Go + Worker Go + Frontend Next.js), a plataforma centraliza integrações assíncronas em um núcleo orientado a eventos: PostgreSQL para estado e RabbitMQ para mensageria resiliente.

---

## 🏗️ Arquitetura e Engenharia

A arquitetura do Projeto Nova foi desenhada utilizando os mesmos padrões de resiliência e escalabilidade de grandes empresas de tecnologia (Enterprise & Big Techs). 

O sistema conta com:
- **Padrão Outbox Transacional:** Eventos não são perdidos em caso de queda do broker. São salvos no banco de dados na mesma transação de negócio e despachados assincronamente.
- **Idempotência:** Contratos e publicações do diário oficial nunca são duplicados se os workers processarem a mesma mensagem duas vezes.
- **Circuit Breaker e Retry Exponencial:** Integrações externas são blindadas contra quedas de APIs de terceiros.
- **WebSockets em Tempo Real:** Fiscais de contrato são notificados instantaneamente ao detectar uma nova publicação no Diário Oficial.
- **Auditoria Imutável:** Ações críticas, como mover um contrato no Kanban, geram registros imutáveis na tabela `audit_logs`, protegidos contra `UPDATE` e `DELETE` no nível do banco de dados (ideal para exigências do TCE).

---

## 📂 Estrutura do Projeto (A Árvore de Diretórios)

A aplicação é dividida em frontend (Next.js) e backend (Go), orquestrados via Docker Compose.

```text
.
├── backend/                   # ⚙️ Código fonte do Backend (Golang)
│   ├── cmd/                   # Pontos de entrada (Binários executáveis)
│   │   ├── api/               # Roda a API HTTP REST e WebSockets.
│   │   ├── worker/            # Roda os consumidores RabbitMQ e jobs em background.
│   │   └── seedadmin/         # CLI para gerar o primeiro usuário administrador.
│   ├── internal/              # Código privado da aplicação (Não pode ser importado por terceiros)
│   │   ├── app/               # Inicialização de dependências (Banco, RabbitMQ, Logger).
│   │   ├── domain/            # Modelos de núcleo puro, erros de domínio e interfaces abstratas.
│   │   ├── modules/           # Módulos de negócio usando Clean Architecture (DDD)
│   │   │   ├── contratos/     # Módulo focado na Gestão de Contratos (Kanban, Workers).
│   │   │   ├── diario_oficial/# Módulo de Scraping, Parser de texto e extração de publicações.
│   │   │   └── notifications/ # Módulo responsável pelo roteamento de WebSocket.
│   │   └── platform/          # Código de infraestrutura transversal (compartilhado por todos os módulos)
│   │       ├── audit/         # Criação de logs inalteráveis.
│   │       ├── auth/          # Autenticação JWT / OIDC (Keycloak).
│   │       ├── database/      # Conexão e transações com o PostgreSQL (pgx).
│   │       ├── idempotency/   # Lógica para barrar processamento duplicado.
│   │       ├── messaging/     # Encapsula o RabbitMQ (Publisher, Consumer).
│   │       ├── outbox/        # Implementação do Transactional Outbox Pattern.
│   │       ├── resilience/    # Circuit Breaker e ferramentas de tolerância a falhas.
│   │       └── ws/            # Gerenciamento de conexões ativas de WebSocket.
│   ├── migrations/            # Arquivos SQL de controle de versão do banco (usando Goose).
│   └── pkg/                   # Pacotes auxiliares genéricos (ex: formatadores de erro HTTP).
├── deploy/                    # Configurações de deploy (RabbitMQ conf).
├── docs/                      # Documentação técnica
│   ├── adr/                   # Architecture Decision Records (Registros de decisões de design).
│   └── openapi.yaml           # Especificação OpenAPI/Swagger da REST API.
├── frontend/                  # 🎨 Código fonte do Frontend (Next.js / React)
│   ├── e2e/                   # Testes End-to-End usando Playwright.
│   ├── public/                # Assets estáticos (Imagens, ícones, fontes).
│   ├── src/
│   │   ├── app/               # App Router do Next.js (Roteamento baseado em pastas)
│   │   │   ├── (protected)/   # Rotas que exigem login (Contratos, Dashboard, Diário Oficial).
│   │   │   ├── login/         # Tela de autenticação local.
│   │   │   └── sobre/         # Página pública detalhando a integração da API.
│   │   ├── components/        # Componentes de UI reaproveitáveis (Cards, Botões, Tabelas).
│   │   ├── hooks/             # Custom hooks do React (ex: uso do WebSocket, polling SWR).
│   │   ├── lib/               # Bibliotecas cliente (Axios client, utilitários, autenticação).
│   │   └── types/             # Definições estáticas do TypeScript mapeando os DTOs do Go.
├── docker-compose.yml         # Declaração dos 5 serviços principais (postgres, rabbitmq, backend, worker, frontend).
└── docker-compose.dev.yml     # Override local expondo portas para debug (5432, 5672).
```

---

## 🚀 Como iniciar o projeto

**Pré-requisitos:** Docker, Docker Compose, Git.

**1. Configurar Ambiente**
Copie o arquivo de exemplo de variáveis de ambiente:
```bash
cp .env.example .env
```
Gere uma chave privada RSA para a autenticação de contas locais:
```bash
mkdir -p secrets
openssl genrsa -out secrets/local_auth_private_key.pem 2048
chmod 644 secrets/local_auth_private_key.pem
```

**2. Subir os Containers**
Suba a infraestrutura completa na rede Docker interna:
```bash
docker compose up --build -d
```

**3. Criar usuário administrador**
Se for seu primeiro uso, gere uma conta local:
```bash
make seed-admin
# Anote a senha aleatória que aparecerá no terminal!
```

**4. Acessar**
* Frontend (Painel e Kanban): [http://localhost:3000](http://localhost:3000)
* Healthcheck da API (Backend): [http://localhost:8000/health](http://localhost:8000/health)

---

## 🔒 Tecnologias Base

* **Frontend:** Next.js (React), TypeScript, Tailwind CSS, SWR, dnd-kit (Kanban), Zustand, WebSockets.
* **Backend:** Go (Golang), Mux (Rotas), pgx (Database Driver).
* **Mensageria:** RabbitMQ (Topic Exchanges, Dead-Letter Queues nativas).
* **Motor de Busca & Inteligência:** Typesense 27+ (Pesquisa textual sub-segundo, facetas em tempo real e parser de atos de RH).
* **Banco de Dados:** PostgreSQL 16.
* **Armazenamento de Objetos (S3):** MinIO.
* **Auditoria de Segurança:** Gitleaks, Trivy (escaneamento das imagens Docker geradas).

---

## 🔎 Motor de Busca Typesense & Inteligência de Diários Oficiais

O Projeto Nova utiliza o **Typesense 27+** para fornecer buscas ultrarrápidas (< 15ms) e tolerantes a erros de digitação sobre as publicações do Diário Oficial de Rondonópolis (**DIORONDON-E**).

### Coleções do Typesense:
1. **`diorondon_articles`**: Extratos de contratos, CNPJs de fornecedores, matérias e editais municipais.
2. **`diorondon_personnel_acts`**: Inteligência de Atos de Pessoal (Nomeações de Efetivos/Comissionados, Exonerações, Remunerações/DAS, Secretarias, CPFs e Matrículas).

Para a documentação completa dos schemas, exemplos de requisições cURL e arquitetura de integração, consulte:
👉 **[Documentação do Motor de Busca Typesense](docs/typesense_search_engine.md)**

