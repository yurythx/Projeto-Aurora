# 📖 Documentação Técnica: Arquitetura de ETL, Streaming e Indexação do Diário Oficial (DIORONDON-E)

Esta documentação detalha a arquitetura do pipeline de ingestão (**ETL - Extract, Transform, Load**), a estratégia de armazenamento em streaming e o sistema de busca indexada implementado para o Diário Oficial de Rondonópolis na plataforma NIX.

---

## ❓ 1. Os PDFs do Diário Oficial ocupam o disco da sua máquina?

**NÃO.** O sistema utiliza **Processamento em Streaming de Memória** (Zero-Disk Storage).

### Como funciona o consumo de espaço:
1. **Sem Downloads no Disco**: O worker abre uma conexão HTTP de leitura com o servidor oficial da prefeitura. O PDF é processado diretamente da memória RAM através do utilitário `pdftotext` consumindo um fluxo de dados (`io.Reader`).
2. **Armazenamento de Extratos no PostgreSQL**: Em vez de guardar o arquivo PDF completo (que pode ter de 10 MB a 50 MB por edição), o pipeline fatiou e extraiu **apenas os textos de atos administrativos e contratos** (ocupando cerca de ~5 KB por edição).
3. **Links Oficiais Preservados**: O banco guarda a URL remota do PDF original (`pdf_url`). Quando o usuário clica em **"Abrir PDF Oficial →"**, o navegador abre o documento diretamente do portal oficial da Prefeitura de Rondonópolis.

> 💡 **Resultado Prático**: 1.000 edições completas do Diário Oficial (que somariam ~15 GB de arquivos PDF no disco) ocupam **menos de 10 MB** na base de dados PostgreSQL.

---

## 🔄 2. Fluxo de Funcionamento do Pipeline (ETL & Vigia)

```mermaid
flowchart TD
    A[Scheduler / Vigia (Watcher)] -->|Verifica a cada 15 min| B[API / Portal Oficial Rondonópolis]
    B -->|Lista edições disponíveis| C{Já existe no Banco?}
    C -->|Sim| D[Ignora / Mantém Status]
    C -->|Não| E[Registra Edição como PENDING]
    E --> F[Worker Pool Concorrente (5 Goroutines)]
    F -->|HTTP Stream (GET PDF)| G[pdftotext -layout (Processamento em Memória)]
    G -->|Texto Bruto| H[Parser Regex (Atos de Pessoal e Contratos)]
    H -->|Gera Hash SHA-256 idempotente| I[Persistência Atômica no PostgreSQL]
    I -->|Grava Achados + Atualiza Status| J[(PostgreSQL: diario_oficial_findings)]
```

---

## 🗄️ 3. Modelo de Dados e Índices de Alta Performance

### 3.1. Tabela `diario_oficial_editions` (Controle do Pipeline)
Registra o catálogo de diários oficiais e o status de ingestão do Worker Pool.

| Coluna | Tipo | Descrição |
| :--- | :--- | :--- |
| `id` | `BIGSERIAL` | Identificador único da edição |
| `edition_number` | `VARCHAR(32)` | Número da edição (ex: "6263") - `UNIQUE` |
| `edition_date` | `TIMESTAMP` | Data oficial da publicação |
| `pdf_url` | `TEXT` | Link remoto do PDF no portal oficial |
| `status` | `VARCHAR(20)` | `PENDING`, `PROCESSING`, `COMPLETED`, `FAILED` |
| `records_count` | `INTEGER` | Quantidade de atos/contratos extraídos |
| `error_message` | `TEXT` | Mensagem de erro em caso de falha |

---

### 3.2. Tabela `diario_oficial_findings` (Achados e Atos Indexados)
Armazena o texto fatiado de exonerações, nomeações, relotações e contratos.

| Coluna | Tipo | Descrição |
| :--- | :--- | :--- |
| `id` | `UUID` | ID do registro |
| `edition_id` | `BIGINT` | Referência à edição |
| `external_id` | `VARCHAR(128)` | Hash SHA-256 para prevenir duplicidade (`UNIQUE`) |
| `act_type` | `VARCHAR(64)` | Tipo do Ato (`EXONERACAO`, `NOMEACAO`, `CONTRATO`, etc.) |
| `servidor_nome` | `VARCHAR(255)` | Nome do servidor público (se houver) |
| `cpf` | `VARCHAR(14)` | CPF do servidor/fiscal |
| `matricula` | `VARCHAR(64)` | Matrícula funcional |
| `empresa_nome` | `VARCHAR(255)` | Razão Social da empresa contratada |
| `cnpj` | `VARCHAR(18)` | CNPJ da empresa |
| `valor` | `NUMERIC(15,2)` | Valor monetário do contrato/aditivo |
| `raw_content` | `TEXT` | Trecho/extrato bruto do ato fatiado |

---

## ⚡ 4. Mecanismo de Busca Otimizado (Trigramas + Full-Text Search)

Para garantir respostas em menos de 5 milissegundos sem depender da API externa nas consultas, foram aplicados 3 níveis de indexação no PostgreSQL:

1. **Índices Trigramas (`pg_trgm`)**:
   ```sql
   CREATE INDEX idx_diario_findings_servidor_trgm ON diario_oficial_findings USING gin (servidor_nome gin_trgm_ops);
   CREATE INDEX idx_diario_findings_empresa_trgm ON diario_oficial_findings USING gin (empresa_nome gin_trgm_ops);
   ```
   *Permite busca por aproximação e pedaços de nomes mesmo com erros de digitação.*

2. **Full-Text Search (`tsvector`)**:
   ```sql
   CREATE INDEX idx_diario_findings_fts ON diario_oficial_findings USING gin (to_tsvector('portuguese', raw_content));
   ```
   *Indexa palavras inteiras do texto do Diário Oficial em português.*

3. **Índices de Documentos (CPF, CNPJ, Matrícula)**:
   ```sql
   CREATE INDEX idx_diario_findings_cpf ON diario_oficial_findings(cpf) WHERE cpf IS NOT NULL;
   CREATE INDEX idx_diario_findings_cnpj ON diario_oficial_findings(cnpj) WHERE cnpj IS NOT NULL;
   ```

---

## 🛡️ 5. Resiliência e Idempotência (Garantia de Não-Duplicação)

* **Chave Idempotente Determinística**: Cada achado tem um `external_id` gerado via `SHA256(edition_number + act_type + servidor_nome + cpf + snippet)`.
* **Cláusula SQL**: `ON CONFLICT (external_id) DO NOTHING`. Se o vigia reprocessar uma edição por qualquer motivo, nenhum registro duplicado será inserido.
* **Transação Atômica**: Se uma edição falhar no meio do PDF, toda a transação daquela edição sofre `ROLLBACK` e a edição recebe status `FAILED` com mensagem descritiva para auditoria.

---

## 📊 6. Tabela Comparativa de Desempenho

| Métrica | Abordagem Antiga (API Direct / File System) | Nossa Arquitetura (Streaming + PostgreSQL Indexado) |
| :--- | :--- | :--- |
| **Tempo de Busca por Nome/CPF** | 2.500 ms - 10.000 ms | **2 ms - 5 ms** |
| **Uso de Disco Local (1.000 edições)** | ~15 GB (Downloads de PDF) | **< 10 MB** (Apenas texto útil) |
| **Carga no Servidor da Prefeitura** | Alta (Dispara múltiplos downloads) | Mínima (Apenas 1 leitura por edição nova) |
| **Busca por Substring / Trigramas** | Inexistente | Sim (Nativa via PostgreSQL `pg_trgm`) |

---

## 🔍 7. Módulo de Auditoria Funcional de Pessoal e Contratos (Busca Por CPF)

A plataforma disponibiliza um motor de auditoria administrativa projetado para rastreabilidade contínua de servidores municipais e contratos de Rondonópolis - MT:

### 👤 7.1. Auditoria de Pessoal (RH & Vínculos)
* **Chave Primária Normalizada**: A pesquisa por CPF (com ou sem formatação, ex: `02194688188` ou `021.946.881-88`) ativa a busca com expressão regular (`regexp`) na base indexada.
* **Linha do Tempo Funcional**: O modal de auditoria consolida a história completa do servidor:
  * **Nomeação / Contratação**: Data, Portaria, Cargo e Órgão de Lotação.
  * **Relotação / Mudança de Setor**: Alterações administrativas e portarias correspondentes.
  * **Exoneração**: Término de vínculo ou encerramento do cargo em comissão.
* **Tabela de Direção e Assessoramento (DAS)**: Classificação automática dos cargos em níveis (de DAS-1 a DAS-6) com referência de remuneração média praticada no município.

### 📜 7.2. Auditoria de Contratos Públicos & Fiscalização
* **Fiscal Titular e Suplente**: Mapeia todas as Portarias de Designação de Fiscalização onde o CPF/Nome do usuário aparece como Fiscal ou Suplente.
* **Cruzamento de Empresas & CNPJ**: Associa os contratos públicos auditados às empresas contratadas, objetos contratuais e montantes financeiros globais.
* **Emissão de Certidões de Auditoria**: Geração em um clique de certidões e extratos formatados em texto puro para juntada de provas e relatórios fiscais.

---

## 🚀 8. Motor de Busca de Alta Performance Typesense 27+

Além da persistência relacional no PostgreSQL, o pipeline de ingestão indexa todos os extratos de diário e atos funcionais de pessoal no motor de busca **Typesense 27+** (Porta `8108`).

O Typesense fornece:
- **Busca em sub-milissegundos (< 15ms)** com tolerância nativa a erros de digitação (*typo tolerance*).
- **Facetas Dinâmicas em Tempo Real**: Filtros por Secretaria, Nível DAS, Tipo de Ato e Categoria da Edição.
- **Coleções Mapeadas**: `diorondon_articles` (Conteúdo integral) e `diorondon_personnel_acts` (Atos de RH).

Consulte o documento completo com schemas e exemplos cURL:
👉 **[Guia Completo da Engine de Busca Typesense](typesense_search_engine.md)**



---

## 🔧 9. Evoluções do pipeline (migrations 000031–000039)

Além da estrutura base (000028), o pipeline ganhou:

### 9.1. Enriquecimento e confiança dos findings
- **000031** — colunas estruturadas em `diario_oficial_findings`: `secretaria`,
  `job_role`, `das_level`, `portaria_number`, `pdf_page_number`, `edition_date`
  (data real de publicação — chave de ordenação cronológica; `created_at` é só
  o instante da ingestão).
- **000033** — coluna `confidence` (`high` | `medium` | `low`). A extração por
  regex sobre prosa tem teto de precisão; em vez de descartar o duvidoso ou
  poluir a busca, cada finding carrega seu grau. **A busca do usuário
  (Typesense) recebe só `high`+`medium`**; `low` fica no PostgreSQL para
  auditoria e para a fila de revisão.
- **000039** — índices GIN trigram em `raw_content` e `portaria_number` para o
  casador Contrato↔Diário (ILIKE `%numero%` deixa de ser seq scan).

### 9.2. Retry de edições FAILED
- **000032** — coluna `retry_count`. O watcher reencaminha edições `FAILED`
  para `PENDING` enquanto `retry_count < 3` (incrementando-o); depois disso a
  edição é abandonada com a mensagem de erro preservada.

### 9.3. Fila de revisão manual
- **000035** — `reviewed_at` / `reviewed_by` / `review_note` +
  índice parcial `WHERE confidence='low' AND reviewed_at IS NULL`.
- Endpoints: `GET .../rondonopolis/review-queue` (lista os `low` não
  revisados), `PATCH .../review-queue/{id}` (**promover** — corrige os campos,
  sobe a confiança, carimba a revisão, dispara reindex da edição no Typesense),
  `POST .../review-queue/{id}/ack` (**marcar como revisado** — é low mas
  legítimo; sai da fila sem entrar na busca), `DELETE .../review-queue/{id}`
  (**descartar** — ruído do parser). Escrita exige `diario_oficial:manage` +
  rate limit por usuário. Tela: `/diario/revisao`.

### 9.4. Indexação no Typesense (reindexador)
- `worker/reindex.go` — reconstrói `diorondon_personnel_acts` /
  `diorondon_articles` a partir da fonte da verdade (as linhas de
  `diario_oficial_findings` no PostgreSQL). Antes disso o Typesense só era
  escrito no instante da ingestão; edições já `COMPLETED` antes do cliente
  existir ficavam fora do índice.
- **Backfill no boot**: o worker reconcilia as coleções na inicialização.
- **Reindex por edição**: após promover um finding na fila de revisão, só
  aquela edição é reindexada (delete-by-filter + reimport).
- Comando one-shot: `worker reindex-diario [--recreate]` (Makefile:
  `make diario-reindex`).

### 9.5. Chave de busca *search-only* (000034)
- A busca do frontend vai **direto ao Typesense** do navegador. A chave admin
  nunca é exposta: `TypesenseSearchKeyManager` cria uma *scoped search-only
  key* (`actions: ["documents:search"]`, restrita às duas coleções), guardada
  em `diario_oficial_kv` (migration 000034) e servida por
  `GET /api/v1/diario-oficial/search-config` (autenticado). O container
  Typesense sobe com `--cors-domains` restrito. Ver
  `docs/typesense_search_engine.md` §7.

### 9.6. Casador automático Contrato ↔ Diário Oficial
- Worker `contratos.diario_matcher` (boot + a cada 6h): casa cada contrato
  não-rascunho por **número** (no `portaria_number` ou no texto) ou por
  **CNPJ**, vincula uma `contrato_diario_refs` por edição (idempotente) e abre
  alertas de fiscalização — `MOVIMENTACAO_PESSOAL` (fiscal exonerado/relotado)
  e `SEM_VINCULO_DIARIO` (contrato vigente sem nenhuma publicação vinculada),
  deduplicados em `contrato_diario_alertas` (migration 000036).
- **Atomicidade (Transactional Outbox)**: para cada contrato, os `INSERT` das
  refs/alertas e os eventos `contrato.diario_ref.linked` / `contrato.fiscal_alert`
  são gravados na **mesma transação** (`database.WithTx`); os eventos chegam ao
  `NotificationCenter` do frontend via o Hub de WebSocket.

### 9.7. Telas de consulta
- `/pessoal` — inteligência de atos de pessoal (Typesense, com facetas de
  secretaria/DAS/tipo de ato, filtro por período, export CSV, permalink).
- `/diario` — busca full-text nos extratos/artigos (mesmos filtro/período/CSV/
  permalink). O trecho de destaque é renderizado com `SafeHighlight`
  (escapa tudo, só o `<mark>` vira elemento) — **nunca** `dangerouslySetInnerHTML`.
