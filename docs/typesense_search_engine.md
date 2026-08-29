# Engine de Busca & Inteligência de Diários Oficiais com Typesense

No **Projeto Nova**, o motor de busca e inteligência analítica do Diário Oficial de Rondonópolis (**DIORONDON-E**) é alimentado pelo **Typesense 27+**. O Typesense foi selecionado por fornecer busca em texto integral sub-milissegundo com tolerância nativa a erros de digitação (*typo tolerance*), suporte a facetas dinâmicas em tempo real e baixíssimo consumo de recursos.

---

## 1. Arquitetura da Integração

O Typesense roda como um serviço containerizado exposto internamente e externamente na porta `8108`.

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                          FLUXO DE INDEXAÇÃO E BUSCA                         │
└─────────────────────────────────────────────────────────────────────────────┘

 [PDF DIORONDON] ──► [Worker Ingestion / Parser] ──► [Typesense Engine :8108]
                                                           ▲
                                                           │ (REST HTTP / Facetas)
                                                           ▼
                                                 [Frontend Next.js]
                                                 - /pessoal (Atos RH)
                                                 - /diario (Texto Integral)
```

- **Serviço Docker:** `typesense` (imagem `typesense/typesense:27.1`)
- **Porta HTTP:** `8108`
- **Autenticação:** Header HTTP `X-TYPESENSE-API-KEY: xyz123secret`
- **Persistência:** Volume Docker `typesense_data`

---

## 2. Coleções e Schemas Mapeados

O Typesense no Projeto Nova gerencia duas coleções principais:

### 2.1. Coleção 1: `diorondon_articles` (Publicações e Contratos Gerais)

Armazena matérias, extratos de contratos, editais, decretos e publicações gerais do município.

| Campo | Tipo | Facetado | Descrição |
|---|---|---|---|
| `id` | `string` | Não | Identificador único do documento (SHA256 da edição + offset). |
| `edition_number` | `int32` | **Sim** | Número oficial da edição do Diário Oficial. |
| `edition_type` | `string` | **Sim** | Categoria da edição (`ORDINARIA`, `SUPLEMENTAR`, `EXTRA`). |
| `publication_date` | `int64` | **Sim** | Timestamp da data de publicação (segundos). |
| `page_number` | `int32` | Não | Número da página no PDF original. |
| `contract_numbers` | `string[]` | **Sim** | Números de contratos encontrados no texto (ex: `491/2025`). |
| `cnpjs` | `string[]` | **Sim** | CNPJs de empresas contratadas citadas no texto. |
| `officials_named` | `string[]` | **Sim** | Nomes de fiscais ou autoridades citadas na publicação. |
| `content` | `string` | Não | Conteúdo integral do texto extraído. |

---

### 2.2. Coleção 2: `diorondon_personnel_acts` (Inteligência de Atos de Pessoal / RH)

Armazena movimentações funcionais de servidores públicos municipais extraídas semanticamente pelo parser de IA/Regex.

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

#### Tipos de Atos Suportados (`act_type`):
- `NOMEACAO_EFETIVO`: Nomeação de concurso público.
- `NOMEACAO_COMISSIONADO`: Nomeação para cargo de confiança (DAS).
- `CONTRATACAO_TEMPORARIA`: Contrato por tempo determinado (seletivos).
- `EXONERACAO`: Desligamento de cargo comissionado ou efetivo.
- `RESCISAO`: Encerramento de contrato temporário.
- `RELOTACAO`: Transferência de servidor entre secretarias.
- `DESIGNACAO_FUNCAO`: Concessão de função gratificada (FG).

---

## 3. Pipeline de Ingestão e Processamento

1. **Captura do Diário:** O worker faz download do PDF original do DIORONDON.
2. **Checagem de Idempotência:** O sistema gera o hash `SHA-256` do arquivo. Se já processado, o pipeline ignora a duplicata.
3. **Parsing de Texto & OCR:** O texto das páginas é extraído.
4. **Extração Semântica de RH (`hr_parser.go`):** Identifica expressões de nomeação/exoneração, extrai o nome limpo do servidor, nível DAS (ex: `DAS-1` a `DAS-6`), secretaria correspondente e portaria.
5. **Indexação no Typesense:** Envia via HTTP `POST /collections/{collection_name}/documents?action=upsert` para garantir atualização sem duplicidade.

---

## 4. Guia da API de Consulta Typesense

### 4.1. Exemplo 1: Buscar Nomeações e Exonerações por Palavra-Chave e Secretaria

**Requisição HTTP:**
```bash
curl -X GET "http://localhost:8108/collections/diorondon_personnel_acts/documents/search?\
q=Silva&\
query_by=person_name,job_role,full_act_text&\
filter_by=secretaria:=SECRETARIA MUNICIPAL DE SAÚDE && act_type:=NOMEACAO_COMISSIONADO&\
facet_by=act_type,secretaria,das_level&\
sort_by=publication_date:desc&\
page=1&per_page=20" \
-H "X-TYPESENSE-API-KEY: xyz123secret"
```

**Resposta JSON Exemplo:**
```json
{
  "found": 12,
  "page": 1,
  "hits": [
    {
      "document": {
        "id": "act-2026-081-12",
        "edition_number": 5891,
        "edition_type": "ORDINARIA",
        "publication_date": 1787836800,
        "act_type": "NOMEACAO_COMISSIONADO",
        "person_name": "Carlos Eduardo Silva",
        "person_matricula": "MAT-2026-091",
        "job_role": "Gerente de Vigilância Sanitária",
        "secretaria": "SECRETARIA MUNICIPAL DE SAÚDE",
        "das_level": "DAS-2",
        "salary_value": 7450.00,
        "portaria_number": "PORT-412/2026",
        "full_act_text": "NOMEAR Carlos Eduardo Silva para exercer o cargo de Gerente de Vigilância Sanitária...",
        "pdf_page_number": 4,
        "pdf_storage_url": "http://localhost:9000/demands/diorondon-5891.pdf"
      },
      "highlights": [
        {
          "field": "person_name",
          "snippet": "Carlos Eduardo <mark>Silva</mark>"
        }
      ]
    }
  ],
  "facet_counts": [
    {
      "field_name": "das_level",
      "counts": [
        { "value": "DAS-2", "count": 8 },
        { "value": "DAS-1", "count": 4 }
      ]
    }
  ]
}
```

---

### 4.2. Exemplo 2: Buscar Contratos e Editais na Coleção de Artigos Gerais

**Requisição HTTP:**
```bash
curl -X GET "http://localhost:8108/collections/diorondon_articles/documents/search?\
q=Construções&\
query_by=content,contract_numbers,cnpjs&\
facet_by=edition_type&\
sort_by=publication_date:desc" \
-H "X-TYPESENSE-API-KEY: xyz123secret"
```

---

## 5. Implementação no Código da Aplicação

### 5.1. Backend Go (`pkg/typesense/client.go`)
O backend expõe a struct `Client` para auto-inicializar as coleções na inicialização da aplicação:

```go
client := typesense.NewClient("http://typesense:8108", "xyz123secret")
if err := client.EnsureCollections(ctx); err != nil {
    log.Fatalf("Erro ao configurar coleções do Typesense: %v", err)
}
```

### 5.2. Frontend Next.js (`src/lib/typesense-client.ts`)
O frontend consulta o Typesense diretamente via requisições `fetch` com suporte a facetas dinâmicas:

```typescript
import { searchPersonnelActs } from "@/lib/typesense-client";

const resultados = await searchPersonnelActs({
  query: "Maria",
  actType: "EXONERACAO",
  secretaria: "SECRETARIA DE EDUCAÇÃO",
  page: 1,
  perPage: 25
});
```

---

## 6. Vantagens Comerciais e de Desempenho

1. **Sub-segundo:** Resultados entregues em **< 15 milissegundos**.
2. **Search-as-you-type:** Atualizações instantâneas à medida que o usuário digita na interface.
3. **Auditoria de Transparência Municipal:** Permite que gestores e o Tribunal de Contas monitorem nomeações, salários comissionados e empresas contratadas no município de Rondonópolis com clareza absoluta.
