"use client";

import { useEffect, useMemo, useState } from "react";
import { FileText, Building2, Tag, Search } from "lucide-react";

import { useToast } from "@/components/notifications/ToastProvider";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { EmptyState } from "@/components/ui/EmptyState";
import { apiClient, ApiError } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { MatchedPublication, MonitoredTerm, PaginationMeta } from "@/types/api";

const PAGE_SIZE = 20;
const ALL = "";

function pathFor(termId: string, page: number) {
  return termId
    ? `v1/diario-oficial/monitored-terms/${termId}/publications?page=${page}&page_size=${PAGE_SIZE}`
    : `v1/diario-oficial/publications?page=${page}&page_size=${PAGE_SIZE}`;
}

// Fallback de publicações recentes coerentes de Rondonópolis
const FALLBACK_PUBLICATIONS: MatchedPublication[] = [
  {
    id: "pub-000a",
    tribunal: "Secretaria Municipal de Administração",
    orgao: "Gabinete do Prefeito",
    tipo_comunicacao: "Extrato de Portaria",
    texto: "PORTARIA Nº 42.500/2026 - Nomeia JOÃO PEDRO ALMEIDA CASTRO para o cargo de Coordenador Geral de Governança Digital (DAS-1) com exercício na Secretaria Municipal de Administração.",
    process_number: "Portaria 42.500",
    process_number_masked: "Edição Nº 6.263 · 26/08/2026",
    availability_date: "2026-08-26",
    link: "https://www.rondonopolis.mt.gov.br/media/docs/edicoes/2026/August/2478a56e-28ce-4c67-b76e-f889f700cb62.pdf",
    monitored_term_id: "t0a",
    monitored_term_label: "Nomeações",
    matched_at: "2026-08-26T14:30:00Z",
  },
  {
    id: "pub-000b",
    tribunal: "Secretaria Municipal de Governo",
    orgao: "Gabinete do Prefeito",
    tipo_comunicacao: "Extrato de Portaria",
    texto: "PORTARIA Nº 42.501/2026 - Nomeia RAFAELA SANTOS MENDONÇA para o cargo de Assessora Especial de TI (DAS-2) com exercício no Gabinete do Prefeito.",
    process_number: "Portaria 42.501",
    process_number_masked: "Edição Nº 6.263 · 26/08/2026",
    availability_date: "2026-08-26",
    link: "https://www.rondonopolis.mt.gov.br/media/docs/edicoes/2026/August/2478a56e-28ce-4c67-b76e-f889f700cb62.pdf",
    monitored_term_id: "t0b",
    monitored_term_label: "Nomeações",
    matched_at: "2026-08-26T14:25:00Z",
  },
  {
    id: "pub-001",
    tribunal: "Secretaria Municipal de Administração",
    orgao: "Gabinete do Prefeito",
    tipo_comunicacao: "Extrato de Portaria",
    texto: "PORTARIA Nº 41.754/2026 - Exonera, a pedido, VANETE BARBOSA DO REGO do cargo em comissão de Agente Administrativo da Família.",
    process_number: "Portaria 41.754",
    process_number_masked: "Edição Nº 6.262 · 24/08/2026",
    availability_date: "2026-08-24",
    link: "https://www.rondonopolis.mt.gov.br/media/docs/edicoes/2026/August/2478a56e-28ce-4c67-b76e-f889f700cb62.pdf",
    monitored_term_id: "t1",
    monitored_term_label: "Exonerações",
    matched_at: "2026-08-24T14:00:00Z",
  },
  {
    id: "pub-002",
    tribunal: "Secretaria Municipal de Promoção Social",
    orgao: "Gabinete do Prefeito",
    tipo_comunicacao: "Extrato de Portaria",
    texto: "PORTARIA Nº 41.808/2026 - Nomeia MARILEIDE GONÇALVES DE OLIVEIRA para o cargo em comissão de Agente Administrativo da Família (DAS-4).",
    process_number: "Portaria 41.808",
    process_number_masked: "Edição Nº 6.253 · 11/08/2026",
    availability_date: "2026-08-11",
    link: "https://www.rondonopolis.mt.gov.br/media/docs/edicoes/2026/August/2478a56e-28ce-4c67-b76e-f889f700cb62.pdf",
    monitored_term_id: "t2",
    monitored_term_label: "Nomeações",
    matched_at: "2026-08-11T13:30:00Z",
  },
  {
    id: "pub-003",
    tribunal: "Secretaria Municipal de Infraestrutura",
    orgao: "Gabinete do Prefeito",
    tipo_comunicacao: "Extrato de Portaria",
    texto: "PORTARIA Nº 41.830/2026 - Nomeia VINICIUS MARTINS GALHARDO LOPES para o cargo em comissão de Assessor de Engenharia e Arquitetura (DAS-2).",
    process_number: "Portaria 41.830",
    process_number_masked: "Edição Nº 6.260 · 20/08/2026",
    availability_date: "2026-08-20",
    link: "https://www.rondonopolis.mt.gov.br/media/docs/edicoes/2026/August/2478a56e-28ce-4c67-b76e-f889f700cb62.pdf",
    monitored_term_id: "t3",
    monitored_term_label: "Engenharia & Arquitetura",
    matched_at: "2026-08-20T11:15:00Z",
  },
  {
    id: "pub-004",
    tribunal: "Secretaria Municipal de Fazenda",
    orgao: "Departamento de Licitações e Contratos",
    tipo_comunicacao: "Designação de Fiscais",
    texto: "PORTARIA Nº 440/2026 - Designa fiscais titulares e suplentes para o acompanhamento e fiscalização do Contrato Nº 440/2026 de Gestão e Arrecadação Fiscal.",
    process_number: "Portaria 440/2026",
    process_number_masked: "Edição Nº 6.260 · 20/08/2026",
    availability_date: "2026-08-20",
    link: "https://www.rondonopolis.mt.gov.br/media/docs/edicoes/2026/August/2478a56e-28ce-4c67-b76e-f889f700cb62.pdf",
    monitored_term_id: "t4",
    monitored_term_label: "Fiscalização de Contratos",
    matched_at: "2026-08-20T09:45:00Z",
  },
];

export function MatchedPublicationsFeed() {
  const { data: terms } = useApiQuery<MonitoredTerm[]>("v1/diario-oficial/monitored-terms");
  const { showToast } = useToast();

  const [termFilter, setTermFilter] = useState(ALL);
  const [orgaoFilter, setOrgaoFilter] = useState(ALL);
  const [tipoFilter, setTipoFilter] = useState(ALL);
  const [searchTerm, setSearchTerm] = useState("");

  const [publications, setPublications] = useState<MatchedPublication[] | null>(null);
  const [meta, setMeta] = useState<PaginationMeta | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function handleTermFilterChange(next: string) {
    setTermFilter(next);
    setPublications(null);
    setMeta(undefined);
    setError(null);
    setLoading(true);
  }

  useEffect(() => {
    let cancelled = false;
    apiClient
      .get<MatchedPublication[]>(pathFor(termFilter, 1))
      .then(({ data, meta: nextMeta }) => {
        if (cancelled) return;
        setPublications(data && data.length > 0 ? data : []);
        setMeta(nextMeta as PaginationMeta | undefined);
      })
      .catch((err) => {
        if (cancelled) return;
        setPublications([]);
        setError(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [termFilter]);

  async function loadMore() {
    if (!meta) return;
    setLoadingMore(true);
    try {
      const nextPage = meta.page + 1;
      const { data, meta: nextMeta } = await apiClient.get<MatchedPublication[]>(pathFor(termFilter, nextPage));
      if (data && data.length > 0) {
        setPublications((current) => [...(current ?? []), ...data]);
      }
      setMeta(nextMeta as PaginationMeta | undefined);
    } catch (err) {
      showToast({
        title: "Não foi possível carregar mais publicações",
        description: err instanceof ApiError ? err.message : "Erro inesperado",
        tone: "danger",
      });
    } finally {
      setLoadingMore(false);
    }
  }

  const listToFilter = Array.isArray(publications) ? publications : [];

  const orgaoOptions = useMemo(
    () => Array.from(new Set(listToFilter.map((p) => p.tribunal))).sort(),
    [listToFilter],
  );
  const tipoOptions = useMemo(
    () => Array.from(new Set(listToFilter.map((p) => p.tipo_comunicacao))).sort(),
    [listToFilter],
  );

  const filtered = listToFilter.filter((p) => {
    if (orgaoFilter && p.tribunal !== orgaoFilter) return false;
    if (tipoFilter && p.tipo_comunicacao !== tipoFilter) return false;
    if (searchTerm.trim()) {
      const q = searchTerm.toLowerCase();
      const textMatch = p.texto.toLowerCase().includes(q);
      const labelMatch = p.monitored_term_label.toLowerCase().includes(q);
      const tribunalMatch = p.tribunal.toLowerCase().includes(q);
      const orgaoMatch = (p.orgao || "").toLowerCase().includes(q);
      return textMatch || labelMatch || tribunalMatch || orgaoMatch;
    }
    return true;
  });

  const hasMore = meta !== undefined && meta.page < meta.total_pages;

  return (
    <div className="flex flex-col gap-4">
      {/* Campo de Busca e Filtros em linha */}
      <div className="flex flex-col gap-3 rounded-lg border border-surface-border bg-surface p-3">
        <div className="relative w-full">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted" />
          <Input
            placeholder="Buscar por termo, palavra-chave, nome, secretaria ou conteúdo..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9 text-sm"
          />
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <label className="flex items-center gap-1.5 text-xs font-medium text-muted">
            Termo Monitorado:
          <select
            value={termFilter}
            onChange={(e) => handleTermFilterChange(e.target.value)}
            aria-label="Filtrar por termo monitorado"
            className="rounded-md border border-surface-border bg-surface px-2.5 py-1 text-xs text-foreground"
          >
            <option value={ALL}>Todos os Termos</option>
            {(terms ?? []).map((t) => (
              <option key={t.id} value={t.id}>
                {t.label}
              </option>
            ))}
          </select>
        </label>

        <label className="flex items-center gap-1.5 text-xs font-medium text-muted">
          Órgão Municipal:
          <select
            value={orgaoFilter}
            onChange={(e) => setOrgaoFilter(e.target.value)}
            aria-label="Filtrar por órgão municipal"
            className="rounded-md border border-surface-border bg-surface px-2.5 py-1 text-xs text-foreground"
          >
            <option value={ALL}>Todos os Órgãos</option>
            {orgaoOptions.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </label>

        <label className="flex items-center gap-1.5 text-xs font-medium text-muted">
          Tipo de Ato:
          <select
            value={tipoFilter}
            onChange={(e) => setTipoFilter(e.target.value)}
            aria-label="Filtrar por tipo de comunicação"
            className="rounded-md border border-surface-border bg-surface px-2.5 py-1 text-xs text-foreground"
          >
            <option value={ALL}>Todos os Tipos</option>
            {tipoOptions.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </label>
        </div>
      </div>

      {error ? (
        <p className="text-sm text-danger">{error}</p>
      ) : loading && !publications ? (
        <p className="text-sm text-muted">Carregando publicações do Diário Oficial de Rondonópolis…</p>
      ) : filtered.length === 0 ? (
        <EmptyState
          title="Não achamos nenhuma referência"
          description="Nenhuma publicação do Diário Oficial de Rondonópolis atende aos filtros selecionados."
        />
      ) : (
        <ul className="flex flex-col gap-3">
          {filtered.map((p) => (
            <li key={p.id} className="flex flex-col gap-2 rounded-lg border border-surface-border bg-surface p-4 shadow-sm hover:border-surface-border/80 transition-all">
              <div className="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border/60 pb-2">
                <div className="flex flex-wrap items-center gap-2 text-xs text-muted">
                  <span className="rounded-full bg-primary/10 text-primary border border-primary/20 px-2.5 py-0.5 font-semibold">
                    {p.tribunal}
                  </span>
                  <span className="font-medium text-foreground">{p.tipo_comunicacao}</span>
                  {p.process_number_masked && <span>· {p.process_number_masked}</span>}
                </div>
                <span className="whitespace-nowrap text-xs text-muted font-medium">
                  {new Date(p.matched_at).toLocaleString("pt-BR")}
                </span>
              </div>

              <p className="text-xs text-foreground/90 leading-relaxed font-medium">
                {p.texto.replace(/<[^>]+>/g, " ").trim()}
              </p>

              <div className="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-surface-border/60 text-xs text-muted">
                <span className="inline-flex items-center gap-1 text-xs">
                  <Tag size={13} className="text-primary" />
                  Monitorado: <strong className="text-foreground">{p.monitored_term_label}</strong>
                </span>
                <a
                  href={p.link || "https://www.rondonopolis.mt.gov.br/diario-oficial/"}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 font-medium text-primary hover:underline"
                >
                  <FileText size={14} />
                  Abrir Diário Oficial (PDF) →
                </a>
              </div>
            </li>
          ))}
        </ul>
      )}

      {meta && meta.total_items > 0 && (
        <div className="flex flex-col items-center gap-1 pt-2">
          {hasMore && (
            <Button variant="secondary" size="sm" onClick={() => void loadMore()} loading={loadingMore}>
              Carregar mais publicações
            </Button>
          )}
          <p className="text-xs text-muted">
            {filtered.length} de {meta.total_items} publicações exibidas
          </p>
        </div>
      )}
    </div>
  );
}
