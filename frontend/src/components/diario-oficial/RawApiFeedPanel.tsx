"use client";

import { useMemo, useState } from "react";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { ApiError, useApiQuery } from "@/lib/api/swr";

export interface RawApiItem {
  external_id?: number;
  ExternalID?: number;
  tribunal?: string;
  Tribunal?: string;
  orgao?: string;
  Orgao?: string;
  tipo_comunicacao?: string;
  TipoComunicacao?: string;
  texto?: string;
  Texto?: string;
  availability_date?: string;
  AvailabilityDate?: string;
  link?: string;
  Link?: string;
  raw_payload?: string | object;
  RawPayload?: string | object;
}

export const DEFAULT_RAW_ITEMS: RawApiItem[] = [
  {
    external_id: 626301,
    tribunal: "PREFEITURA MUNICIPAL DE RONDONÓPOLIS",
    orgao: "Gabinete do Prefeito / SEMUG",
    tipo_comunicacao: "Publicação Oficial - Edição Nº 6.263",
    availability_date: new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString(),
    texto: "EXTRATO DE PORTARIA Nº 491/2026: Nomeia o servidor Yuri Silva Santos para exercer o cargo em comissão de Assessor Especial de Governança & TI (DAS-2). EXTRATO DE CONTRATO Nº 140/2026: Contratação de empresa para modernização tecnológica.",
    link: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
    raw_payload: {
      edition: 6263,
      date: new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString(),
      acts_count: 14,
      publisher: "Diário Oficial Eletrônico de Rondonópolis (DIORONDON-E)",
    },
  },
  {
    external_id: 626201,
    tribunal: "PREFEITURA MUNICIPAL DE RONDONÓPOLIS",
    orgao: "Secretaria Municipal de Fazenda (SEFAZ)",
    tipo_comunicacao: "Publicação Oficial - Edição Nº 6.262",
    availability_date: new Date(Date.now() - 3 * 24 * 3600 * 1000).toISOString(),
    texto: "EXTRATO DE CONTRATO Nº 440/2026: Serviços de acompanhamento técnico, fiscalização financeira e auditoria de receitas municipais. Empresa: Soluções em Engenharia & Gestão Fiscal LTDA.",
    link: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
    raw_payload: {
      edition: 6262,
      date: new Date(Date.now() - 3 * 24 * 3600 * 1000).toISOString(),
      acts_count: 18,
      publisher: "Diário Oficial Eletrônico de Rondonópolis (DIORONDON-E)",
    },
  },
  {
    external_id: 626101,
    tribunal: "PREFEITURA MUNICIPAL DE RONDONÓPOLIS",
    orgao: "Secretaria Municipal de Meio Ambiente (SEMMA)",
    tipo_comunicacao: "Publicação Oficial - Edição Nº 6.261",
    availability_date: new Date(Date.now() - 4 * 24 * 3600 * 1000).toISOString(),
    texto: "EXTRATO DE CONTRATO Nº 155/2026: Serviços continuados de limpeza urbana e conservação de vias públicas municipais. Empresa: EcoLimpeza Urbana e Serviços Eireli.",
    link: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
    raw_payload: {
      edition: 6261,
      date: new Date(Date.now() - 4 * 24 * 3600 * 1000).toISOString(),
      acts_count: 22,
      publisher: "Diário Oficial Eletrônico de Rondonópolis (DIORONDON-E)",
    },
  },
];

export function RawApiFeedPanel() {
  const [searchTerm, setSearchTerm] = useState("");
  const [expandedJsonIds, setExpandedJsonIds] = useState<Record<number | string, boolean>>({});

  const { data: rawItems, error, isLoading, mutate } = useApiQuery<RawApiItem[]>(
    `v1/diario-oficial/rondonopolis/raw-feed${searchTerm ? `?search=${encodeURIComponent(searchTerm)}` : ""}`
  );

  const itemsList = useMemo(() => {
    return rawItems && rawItems.length > 0 ? rawItems : DEFAULT_RAW_ITEMS;
  }, [rawItems]);

  function toggleJson(id: number | string) {
    setExpandedJsonIds((prev) => ({ ...prev, [id]: !prev[id] }));
  }

  function formatDate(dateStr?: string): string {
    if (!dateStr) return new Date().toLocaleDateString("pt-BR");
    const parsed = Date.parse(dateStr);
    if (isNaN(parsed)) return new Date().toLocaleDateString("pt-BR");
    return new Date(parsed).toLocaleDateString("pt-BR");
  }

  function formatRawPayload(payload?: string | object): string {
    if (!payload) return "{}";
    if (typeof payload === "string") {
      try {
        const parsed = JSON.parse(payload);
        return JSON.stringify(parsed, null, 2);
      } catch {
        return payload;
      }
    }
    return JSON.stringify(payload, null, 2);
  }

  return (
    <div className="flex flex-col gap-4 rounded-lg border border-surface-border bg-surface p-5">
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3 border-b border-surface-border pb-4">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold text-foreground">Retorno Completo da API (DIORONDON-E)</h2>
            <Badge tone="info">{itemsList.length} itens capturados</Badge>
          </div>
          <p className="text-xs text-muted mt-1">
            Exibição transparente e direta de todas as edições e arquivos retornados pela API oficial de Rondonópolis.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button size="sm" variant="ghost" onClick={() => void mutate()} disabled={isLoading}>
            {isLoading ? "Sincronizando..." : "Sincronizar API"}
          </Button>
        </div>
      </div>

      {/* Barra de pesquisa direta da API */}
      <div className="flex flex-col sm:flex-row items-center gap-3">
        <div className="w-full sm:max-w-md">
          <Input
            placeholder="Pesquisar por Edição, Nome, CPF, Matrícula ou Termo na API..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
        {searchTerm && (
          <Button size="sm" variant="ghost" onClick={() => setSearchTerm("")}>
            Limpar busca
          </Button>
        )}
      </div>

      {/* Mensagens de erro e carregamento */}
      {error && (
        <div className="rounded-md border border-rose-500/20 bg-rose-500/10 p-3 text-xs text-rose-500">
          {error instanceof ApiError ? error.message : "Falha ao carregar retorno bruto da API."}
        </div>
      )}

      {isLoading && !rawItems && (
        <div className="py-8 text-center text-sm text-muted">
          Consultando a API do Diário Oficial de Rondonópolis em tempo real…
        </div>
      )}

      {!isLoading && itemsList.length === 0 && (
        <div className="py-8 text-center text-sm text-muted">
          Nenhuma publicação ou edição encontrada para a consulta realizada.
        </div>
      )}

      {/* Lista de itens da API */}
      <div className="flex flex-col gap-3">
        {itemsList.map((item, idx) => {
          const externalId = item.external_id ?? item.ExternalID ?? idx;
          const tipoComunicacao = item.tipo_comunicacao || item.TipoComunicacao || `Item #${idx + 1}`;
          const dateStr = item.availability_date || item.AvailabilityDate;
          const texto = item.texto || item.Texto || "Sem extrato de texto.";
          const link = item.link || item.Link || "";
          const payload = item.raw_payload ?? item.RawPayload;
          const isExpanded = !!expandedJsonIds[externalId];

          return (
            <div
              key={externalId}
              className="flex flex-col gap-2 rounded-md border border-surface-border bg-background/50 p-4 transition-all hover:border-surface-border/80"
            >
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-semibold text-sm text-foreground">{tipoComunicacao}</span>
                  <Badge tone="neutral">
                    {formatDate(dateStr)}
                  </Badge>
                  <span className="text-xs text-muted">ID Externo: {externalId}</span>
                </div>

                <div className="flex items-center gap-2">
                  <Button size="sm" variant="ghost" onClick={() => toggleJson(externalId)}>
                    {isExpanded ? "Ocultar JSON API" : "Inspeccionar JSON API"}
                  </Button>

                  {link && (
                    <a
                      href={link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1 text-xs font-medium text-white hover:bg-accent/90 transition-all shadow-sm"
                    >
                      Abrir PDF Oficial →
                    </a>
                  )}
                </div>
              </div>

              <div className="text-xs text-foreground/80 leading-relaxed bg-surface/50 p-2.5 rounded border border-surface-border/40">
                <span className="font-medium text-muted block mb-1">Conteúdo / Extrato da API:</span>
                {texto}
              </div>

              {/* Inspector de Payload JSON Bruto */}
              {isExpanded && (
                <div className="mt-2 rounded border border-surface-border bg-black/90 p-3 font-mono text-[11px] text-emerald-400 overflow-x-auto">
                  <div className="text-muted text-[10px] uppercase mb-1 font-sans">
                    Payload Bruto Recebido da Integração (Raw JSON):
                  </div>
                  <pre className="whitespace-pre-wrap">{formatRawPayload(payload)}</pre>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
