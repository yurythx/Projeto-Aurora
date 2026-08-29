import { apiClient } from "@/lib/api/client";
import { publicationDateFilter } from "@/lib/search-tools";

export interface PersonnelAct {
  id: string;
  edition_number: number;
  edition_type: string;
  publication_date: number;
  act_type: "NOMEACAO_EFETIVO" | "NOMEACAO_COMISSIONADO" | "CONTRATACAO_TEMPORARIA" | "EXONERACAO" | "RESCISAO" | "RELOTACAO" | "DESIGNACAO_FUNCAO";
  person_name: string;
  person_cpf?: string;
  person_matricula?: string;
  job_role: string;
  secretaria: string;
  das_level?: string;
  salary_value?: number;
  portaria_number?: string;
  full_act_text: string;
  pdf_page_number: number;
  pdf_storage_url: string;
  /** high = extração corroborada (CPF/matrícula/DAS); medium = só nome plausível. */
  confidence?: "high" | "medium";
}

export interface GazetteArticle {
  id: string;
  edition_number: number;
  edition_type: string;
  publication_date: number;
  page_number: number;
  contract_numbers: string[];
  cnpjs: string[];
  officials_named: string[];
  content: string;
  pdf_storage_url?: string;
}

export interface TypesenseSearchResponse<T> {
  found: number;
  page: number;
  hits: Array<{
    document: T;
    highlights: Array<{
      field: string;
      snippet: string;
      snippets?: string[];
    }>;
  }>;
  facet_counts?: Array<{
    field_name: string;
    counts: Array<{
      value: string;
      count: number;
    }>;
  }>;
}

const TYPESENSE_HOST = process.env.NEXT_PUBLIC_TYPESENSE_URL || "http://localhost:8108";

/**
 * A chave do Typesense NÃO vem mais de env pública (era a chave admin — dava
 * pra apagar coleções pelo devtools). O backend gera uma key restrita a
 * `documents:search` e a entrega por este endpoint autenticado; cacheamos a
 * promessa por sessão. O schema das coleções é responsabilidade EXCLUSIVA do
 * backend (EnsureCollections) — o frontend só lê.
 */
let searchKeyPromise: Promise<string> | null = null;
function getSearchKey(): Promise<string> {
  if (!searchKeyPromise) {
    searchKeyPromise = apiClient
      .get<{ search_key: string }>("v1/diario-oficial/search-config")
      .then((r) => r.data?.search_key || "")
      .catch(() => {
        searchKeyPromise = null; // permite nova tentativa depois
        return "";
      });
  }
  return searchKeyPromise;
}

/** Estrutura vazia mas tipada — usada como retorno seguro quando o Typesense
 * não tem dados mas a requisição foi tecnicamente bem-sucedida. */
function emptyTypesenseResponse<T>(): TypesenseSearchResponse<T> {
  return { found: 0, page: 1, hits: [], facet_counts: [] };
}

/**
 * Anexa a âncora de página ao PDF da edição quando a página é conhecida (>1),
 * para o link abrir direto na página do ato/contrato filtrado. Retorna string
 * vazia (não um link genérico) quando não há URL — quem renderiza decide se
 * mostra o link.
 */
export function pdfUrlWithPage(url: string | undefined, page: number | undefined): string {
  if (!url) return "";
  const p = Number(page) || 1;
  if (p > 1 && !url.includes("#page=")) return `${url}#page=${p}`;
  return url;
}

function normalizeText(text: string | undefined): string {
  if (!text) return "";
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase();
}

function filterHREvents(events: any[], query?: string, actType?: string, secretaria?: string, dasLevel?: string) {
  const safeEvents = Array.isArray(events) ? events : [];
  const qNorm = normalizeText(query);
  const secNorm = normalizeText(secretaria);

  return safeEvents.filter((ev) => {
    const matchesQuery =
      !qNorm ||
      [ev.servidor, ev.cargo, ev.secretaria, ev.portaria_number, ev.servidor_cpf, ev.servidor_matricula, ev.context_snippet].some(
        (field) => normalizeText(field).includes(qNorm)
      );

    let evType = ev.type;
    if (ev.type === "NOMEACAO") evType = "NOMEACAO_COMISSIONADO";
    else if (ev.type === "MUDANCA_SETOR") evType = "RELOTACAO";

    const matchesActType = !actType || ev.type === actType || evType === actType;
    const matchesSecretaria = !secNorm || normalizeText(ev.secretaria).includes(secNorm);
    const matchesDas = !dasLevel || ev.das_level === dasLevel;

    return matchesQuery && matchesActType && matchesSecretaria && matchesDas;
  });
}

function filterContracts(contracts: any[], query?: string) {
  const safeContracts = Array.isArray(contracts) ? contracts : [];
  const qNorm = normalizeText(query);
  return safeContracts.filter((c) => {
    return (
      !qNorm ||
      [c.contract_number, c.contractor, c.contractor_cnpj, c.fiscal_nome, c.fiscal_cpf, c.portaria_number, c.object].some(
        (field) => normalizeText(field).includes(qNorm)
      )
    );
  });
}

export async function searchPersonnelActs(params: {
  query?: string;
  actType?: string;
  secretaria?: string;
  dasLevel?: string;
  dateFrom?: number;
  dateTo?: number;
  page?: number;
  perPage?: number;
}): Promise<TypesenseSearchResponse<PersonnelAct>> {
  const { query = "", actType, secretaria, dasLevel, dateFrom, dateTo, page = 1, perPage = 20 } = params;

  // 1. Tenta buscar diretamente no Typesense (com auto-criação de coleção em 404)
  try {
    const filters: string[] = [];
    if (actType) filters.push(`act_type:=${actType}`);
    if (secretaria) filters.push(`secretaria:=${secretaria}`);
    if (dasLevel) filters.push(`das_level:=${dasLevel}`);
    filters.push(...publicationDateFilter(dateFrom, dateTo));

    const buildPersonnelUrl = () => {
      const u = new URL(`${TYPESENSE_HOST}/collections/diorondon_personnel_acts/documents/search`);
      u.searchParams.set("q", query || "*");
      u.searchParams.set("query_by", "person_name,job_role,secretaria,full_act_text,person_matricula,person_cpf,portaria_number");
      u.searchParams.set("facet_by", "act_type,secretaria,das_level,edition_type");
      u.searchParams.set("page", page.toString());
      u.searchParams.set("per_page", perPage.toString());
      u.searchParams.set("sort_by", query ? "publication_date:desc" : "person_name:asc");
      if (filters.length > 0) u.searchParams.set("filter_by", filters.join(" && "));
      return u;
    };

    const searchKey = await getSearchKey();
    if (searchKey) {
      const res = await fetch(buildPersonnelUrl().toString(), {
        headers: { "X-TYPESENSE-API-KEY": searchKey },
      });
      // 404 = coleção ainda não existe (pipeline não rodou) -> cai no fallback
      // do backend. O frontend não cria coleção (schema é do backend).
      if (res.ok) {
        return (await res.json()) as TypesenseSearchResponse<PersonnelAct>;
      }
    }
  } catch {
    // Falha de rede ou JSON inválido — continua para o fallback do backend
  }

  // 2. Fallback resiliente: Busca via API do Backend (PostgreSQL + Scraping Rondonópolis)
  try {
    const queryParams = new URLSearchParams();
    if (query) queryParams.set("search", query);
    if (actType) queryParams.set("event_type", actType);

    const apiRes = await apiClient.get<any[]>(`v1/diario-oficial/rondonopolis/hr-events?${queryParams.toString()}`);
    let rawEvents = Array.isArray(apiRes.data) ? apiRes.data : [];

    rawEvents = filterHREvents(rawEvents, query, actType, secretaria, dasLevel);

    if (!query) {
      rawEvents.sort((a, b) => {
        const nameA = a.servidor || "";
        const nameB = b.servidor || "";
        return nameA.localeCompare(nameB);
      });
    }

    const hits = rawEvents.map((ev, index) => {
      // O backend já entrega act_type no vocabulário canônico; o mapa abaixo
      // só cobre respostas antigas ainda em cache.
      const legacyActMap: Record<string, PersonnelAct["act_type"]> = {
        NOMEACAO: "NOMEACAO_COMISSIONADO",
        MUDANCA_SETOR: "RELOTACAO",
      };
      const actTypeFormatted = (legacyActMap[ev.type] ?? ev.type ?? "NOMEACAO_COMISSIONADO") as PersonnelAct["act_type"];

      // Sem fabricar dados: campo ausente vira vazio/undefined, nunca um valor
      // inventado (ex.: edição "6263", DAS-2, ou o link genérico do portal),
      // que fazia um resultado apontar para um diário que não é o dele.
      const doc: PersonnelAct = {
        id: `hr-event-${index}-${ev.servidor || "servidor"}`,
        edition_number: parseInt(ev.edition_number, 10) || 0,
        edition_type: ev.edition_type || "ORDINARIA",
        publication_date: ev.publication_date ? new Date(ev.publication_date).getTime() / 1000 : 0,
        act_type: actTypeFormatted,
        person_name: ev.servidor || "—",
        person_cpf: ev.servidor_cpf || "",
        person_matricula: ev.servidor_matricula || "",
        job_role: ev.cargo || "",
        secretaria: ev.secretaria || "Prefeitura Municipal de Rondonópolis",
        das_level: ev.das_level || undefined,
        salary_value: typeof ev.salary_value === "number" ? ev.salary_value : undefined,
        portaria_number: ev.portaria_number || "",
        full_act_text: ev.context_snippet || "",
        pdf_page_number: Number(ev.pdf_page_number) || 1,
        pdf_storage_url: ev.doc_url || "",
        confidence: ev.confidence === "high" ? "high" : "medium",
      };

      return {
        document: doc,
        highlights: [
          {
            field: "person_name",
            snippet: doc.person_name,
          },
        ],
      };
    });

    // Calcular contagens facetadas dinâmicas
    const secCounts: Record<string, number> = {};
    const dasCounts: Record<string, number> = {};

    hits.forEach((h) => {
      secCounts[h.document.secretaria] = (secCounts[h.document.secretaria] || 0) + 1;
      if (h.document.das_level) {
        dasCounts[h.document.das_level] = (dasCounts[h.document.das_level] || 0) + 1;
      }
    });

    return {
      found: hits.length,
      page: 1,
      hits,
      facet_counts: [
        {
          field_name: "secretaria",
          counts: Object.entries(secCounts).map(([value, count]) => ({ value, count })),
        },
        {
          field_name: "das_level",
          counts: Object.entries(dasCounts).map(([value, count]) => ({ value, count })),
        },
      ],
    };
  } catch (backendErr) {
    return emptyTypesenseResponse();
  }
}

export async function searchGazetteArticles(params: {
  query?: string;
  editionType?: string;
  dateFrom?: number;
  dateTo?: number;
  page?: number;
  perPage?: number;
}): Promise<TypesenseSearchResponse<GazetteArticle>> {
  const { query = "", editionType, dateFrom, dateTo, page = 1, perPage = 20 } = params;

  // 1. Tenta buscar no Typesense (com auto-criação de coleção em 404)
  try {
    const filters: string[] = [];
    if (editionType) filters.push(`edition_type:=${editionType}`);
    filters.push(...publicationDateFilter(dateFrom, dateTo));

    const buildArticlesUrl = () => {
      const u = new URL(`${TYPESENSE_HOST}/collections/diorondon_articles/documents/search`);
      u.searchParams.set("q", query || "*");
      u.searchParams.set("query_by", "content,contract_numbers,cnpjs,officials_named");
      u.searchParams.set("facet_by", "edition_type,contract_numbers,cnpjs");
      u.searchParams.set("page", page.toString());
      u.searchParams.set("per_page", perPage.toString());
      u.searchParams.set("sort_by", "publication_date:desc");
      if (filters.length > 0) u.searchParams.set("filter_by", filters.join(" && "));
      return u;
    };

    const searchKey = await getSearchKey();
    let res: Response | null = null;
    if (searchKey) {
      res = await fetch(buildArticlesUrl().toString(), {
        headers: { "X-TYPESENSE-API-KEY": searchKey },
      });
    }
    if (!res) throw new Error("sem chave de busca");

    if (res.ok) {
      const data: TypesenseSearchResponse<GazetteArticle> = await res.json();
      return data;
    }
  } catch {
    // Falha de rede ou JSON inválido — continua para o fallback do backend
  }

  // 2. Fallback para a API de Contratos/Raw Feed
  try {
    const queryParams = new URLSearchParams();
    if (query) queryParams.set("search", query);

    const apiRes = await apiClient.get<any[]>(`v1/diario-oficial/rondonopolis/contracts?${queryParams.toString()}`);
    let contracts = Array.isArray(apiRes.data) ? apiRes.data : [];

    contracts = filterContracts(contracts, query);

    const hits = contracts.map((c, index) => {
      const doc: GazetteArticle = {
        id: c.id || `contract-${index}`,
        edition_number: parseInt(c.edition_number, 10) || 0,
        edition_type: c.edition_type || "ORDINARIA",
        publication_date: c.publication_date ? new Date(c.publication_date).getTime() / 1000 : 0,
        page_number: Number(c.pdf_page_number) || 1,
        contract_numbers: [c.contract_number || "Contrato"],
        cnpjs: [c.contractor_cnpj || ""],
        officials_named: [c.fiscal_nome || ""],
        content: c.object
          ? `EXTRATO DE CONTRATO — DIÁRIO OFICIAL DE RONDONÓPOLIS: ${c.contract_number} — ${c.contractor} (CNPJ ${c.contractor_cnpj}). Objeto: ${c.object}. Valor Total: ${c.value}. Fiscal Titular: ${c.fiscal_nome} (CPF ${c.fiscal_cpf}). Portaria nº ${c.portaria_number}.`
          : c.object || "",
        pdf_storage_url: c.doc_url || "",
      };

      return {
        document: doc,
        highlights: [
          {
            field: "content",
            snippet: doc.content,
          },
        ],
      };
    });

    return {
      found: hits.length,
      page: 1,
      hits,
      facet_counts: [
        {
          field_name: "edition_type",
          counts: [{ value: "ORDINARIA", count: hits.length }],
        },
      ],
    };
  } catch (backendErr) {
    return emptyTypesenseResponse();
  }
}

export async function getTypesenseHealth(): Promise<{ healthy: boolean; message: string; docCount?: number }> {
  try {
    // /health é público (sem key).
    const res = await fetch(`${TYPESENSE_HOST}/health`, { method: "GET" });
    if (!res.ok) {
      return { healthy: false, message: `Typesense retornou status HTTP ${res.status}` };
    }

    // Contagem via search (a key search-only não lê metadados de coleção).
    let docCount = 0;
    try {
      const key = await getSearchKey();
      if (key) {
        const countRes = await fetch(
          `${TYPESENSE_HOST}/collections/diorondon_personnel_acts/documents/search?q=*&per_page=0`,
          { headers: { "X-TYPESENSE-API-KEY": key } }
        );
        if (countRes.ok) docCount = (await countRes.json()).found || 0;
      }
    } catch {}

    return { healthy: true, message: "Motor de busca Typesense operacional", docCount };
  } catch (err: any) {
    return { healthy: false, message: err.message || "Falha na conexão com o Typesense" };
  }
}
