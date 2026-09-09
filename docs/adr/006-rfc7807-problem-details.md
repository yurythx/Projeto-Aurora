# 006 — Respostas de erro no padrão RFC 7807 (Problem Details)

- **Status:** Proposto (aguarda decisão de versionamento da API)
- **Data:** 2026-09-09
- **Autores:** Engenharia Projeto Aurora

## Contexto

Gap G-13 da auditoria de conformidade. Hoje todo erro da API sai no
envelope próprio da plataforma:

```json
{ "data": null, "error": { "code": "VALIDATION_ERROR", "message": "..." } }
```

com `Content-Type: application/json`. A interoperabilidade e-PING e o
padrão RFC 7807 pedem `application/problem+json` com os campos
`type` (URI), `title`, `status`, `detail` e `instance`.

Trocar o formato do corpo de erro é uma **quebra de contrato**: todo
cliente atual (o frontend Next.js inclusive) lê `error.code` /
`error.message`.

## Decisão (proposta)

Duas opções, a decidir com os consumidores da API:

### Opção A — Nova major `/api/v2` (preferida)

- `/api/v1` continua com o envelope atual, sem data de fim anunciada.
- `/api/v2` responde `application/problem+json`. `type` é uma URI estável
  de um catálogo de erros publicado (ex.: `https://<host>/errors/validation-error`).
- `apperrors.Error` ganha um método `ProblemDetails(instance string)` que
  a camada de transporte serializa; o `code` legível por máquina é
  mantido como membro de extensão (`"code": "VALIDATION_ERROR"`).
- `openapi.yaml` documenta os dois.

### Opção B — Content negotiation por `Accept`

- Um cliente que manda `Accept: application/problem+json` recebe o
  formato RFC 7807; o default continua o envelope atual.
- Menos limpo (o mesmo endpoint com dois formatos de erro), mas sem nova
  árvore de rotas.

## Consequências

- Enquanto não houver decisão, `httputil.WriteError` permanece como está.
- A implementação não bloqueia nenhum outro item do roadmap.
- Quando decidido, este ADR passa a **Aceito** e ganha a seção de
  implementação.
