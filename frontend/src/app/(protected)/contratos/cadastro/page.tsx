"use client";

import { useCallback, useEffect, useState } from "react";
import {
  FileSignature,
  Search,
  ChevronDown,
  ChevronRight,
  Newspaper,
  FileText,
  RefreshCw,
} from "lucide-react";
import { fetchContratos, getContratoDetail } from "@/lib/api/contratos";
import { safeHttpUrl } from "@/lib/search-tools";
import type { Contrato, DiarioRef, PaginationMeta } from "@/types/api";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Card, CardContent } from "@/components/ui/Card";

const STATUS_OPTIONS = [
  { value: "", label: "Todos os status" },
  { value: "rascunho", label: "Rascunho" },
  { value: "em_analise", label: "Em Análise" },
  { value: "aprovado", label: "Aprovado" },
  { value: "vigente", label: "Vigente" },
  { value: "encerrado", label: "Encerrado" },
  { value: "cancelado", label: "Cancelado" },
];

function tipoEventoTone(tipo?: string): "info" | "warning" | "danger" | "success" | "neutral" {
  switch (tipo) {
    case "RESCISAO":
      return "danger";
    case "RELOTACAO":
      return "warning";
    case "FISCALIZACAO":
      return "info";
    case "CONTRATO":
    case "NOMEACAO":
      return "success";
    default:
      return "neutral";
  }
}

function fmtDate(s?: string): string {
  if (!s) return "—";
  const d = new Date(s);
  return Number.isNaN(d.getTime()) ? s : d.toLocaleDateString("pt-BR");
}

function DiarioRefsPanel({ refs }: { refs: DiarioRef[] }) {
  if (refs.length === 0) {
    return (
      <p className="text-xs text-muted italic">
        Nenhuma publicação do Diário Oficial vinculada. O casador automático roda a cada 6h; verifique
        se o número/CNPJ do contrato aparece no DIORONDON.
      </p>
    );
  }
  return (
    <ul className="flex flex-col gap-2">
      {refs.map((ref, i) => (
        <li key={`${ref.edition_number}-${i}`} className="rounded-md border border-border/40 p-3">
          <div className="flex flex-wrap items-center gap-2 text-xs">
            <Badge tone={tipoEventoTone(ref.tipo_evento)}>{ref.tipo_evento || "MENÇÃO"}</Badge>
            <span className="font-mono text-muted">Edição #{ref.edition_number}</span>
            <span className="text-muted">{fmtDate(ref.publicado_em)}</span>
          </div>
          {ref.contexto && (
            <p className="mt-1.5 line-clamp-3 border-l-2 border-primary/50 pl-2 text-xs text-muted">
              {ref.contexto}
            </p>
          )}
          {safeHttpUrl(ref.doc_url) && (
            <a
              href={safeHttpUrl(ref.doc_url)}
              target="_blank"
              rel="noreferrer"
              className="mt-1 inline-flex items-center gap-1 text-xs text-primary hover:underline"
            >
              <FileText size={12} />
              Ver no PDF oficial →
            </a>
          )}
        </li>
      ))}
    </ul>
  );
}

function ContratoRow({ contrato }: { contrato: Contrato }) {
  const [open, setOpen] = useState(false);
  const [detail, setDetail] = useState<Contrato | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);

  const toggle = async () => {
    const next = !open;
    setOpen(next);
    if (next && !detail) {
      setLoadingDetail(true);
      try {
        setDetail(await getContratoDetail(contrato.id));
      } catch {
        setDetail(contrato);
      } finally {
        setLoadingDetail(false);
      }
    }
  };

  const d = detail ?? contrato;

  return (
    <Card>
      <button
        onClick={toggle}
        className="flex w-full items-center gap-3 p-4 text-left hover:bg-surface-hover/40"
      >
        {open ? <ChevronDown size={16} className="shrink-0 text-muted" /> : <ChevronRight size={16} className="shrink-0 text-muted" />}
        <span className="font-mono text-sm font-semibold text-primary">{contrato.numero}</span>
        <span className="flex-1 truncate text-sm">{contrato.objeto}</span>
        <span className="hidden sm:block truncate text-xs text-muted max-w-[180px]">{contrato.contratado}</span>
        <Badge tone={contrato.status === "vigente" ? "success" : contrato.status === "cancelado" ? "danger" : "neutral"}>
          {contrato.status_label}
        </Badge>
      </button>

      {open && (
        <CardContent className="border-t border-border/40 p-4 flex flex-col gap-4">
          {loadingDetail ? (
            <p className="text-xs text-muted">Carregando detalhes…</p>
          ) : (
            <>
              <div className="grid grid-cols-2 gap-3 text-xs md:grid-cols-4">
                <div>
                  <span className="block text-[10px] uppercase text-muted">Contratante</span>
                  {d.contratante || "—"}
                </div>
                <div>
                  <span className="block text-[10px] uppercase text-muted">Contratado</span>
                  {d.contratado || "—"}
                </div>
                <div>
                  <span className="block text-[10px] uppercase text-muted">CNPJ</span>
                  <span className="font-mono">{d.cnpj || "—"}</span>
                </div>
                <div>
                  <span className="block text-[10px] uppercase text-muted">Valor</span>
                  {d.valor != null
                    ? `R$ ${d.valor.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}`
                    : "—"}
                </div>
                <div>
                  <span className="block text-[10px] uppercase text-muted">Vigência</span>
                  {fmtDate(d.data_vigencia_inicio)} – {fmtDate(d.data_vigencia_fim)}
                </div>
                <div>
                  <span className="block text-[10px] uppercase text-muted">Assinatura</span>
                  {fmtDate(d.data_assinatura)}
                </div>
              </div>

              <div>
                <h4 className="mb-2 flex items-center gap-1.5 text-xs font-semibold text-foreground">
                  <Newspaper size={13} className="text-primary" />
                  Publicações do Diário Oficial ({d.diario_refs?.length ?? 0})
                </h4>
                <DiarioRefsPanel refs={d.diario_refs ?? []} />
              </div>

              {(d.aditivos?.length ?? 0) > 0 && (
                <div>
                  <h4 className="mb-2 text-xs font-semibold text-foreground">Aditivos ({d.aditivos!.length})</h4>
                  <ul className="flex flex-col gap-1 text-xs text-muted">
                    {d.aditivos!.map((a) => (
                      <li key={a.id}>
                        {a.numero || "Aditivo"} · {a.objeto || "—"}
                        {a.valor_adicional != null &&
                          ` · +R$ ${a.valor_adicional.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}`}
                        {a.data_assinatura && ` · ${fmtDate(a.data_assinatura)}`}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </>
          )}
        </CardContent>
      )}
    </Card>
  );
}

export default function ContratosCadastroPage() {
  const [items, setItems] = useState<Contrato[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const [busca, setBusca] = useState("");
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setErr(null);
    try {
      const res = await fetchContratos(page, 20, status || undefined, busca.trim() || undefined);
      setItems(res.data ?? []);
      setMeta(res.meta);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Falha ao carregar contratos.");
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [page, status, busca]);

  // Debounce: página/status recarregam na hora; a busca textual espera o
  // usuário parar de digitar (~350ms) para não disparar um request por tecla.
  useEffect(() => {
    const t = setTimeout(load, 350);
    return () => clearTimeout(t);
  }, [load]);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 border-b border-surface-border pb-4">
        <div>
          <h1 className="text-xl font-bold text-foreground flex items-center gap-2">
            <FileSignature className="h-6 w-6 text-primary" />
            Contratos Municipais — Cadastro
          </h1>
          <p className="text-xs text-muted mt-1">
            Cada contrato mostra as publicações do Diário Oficial vinculadas automaticamente (número / CNPJ).
          </p>
        </div>
        <Button size="sm" variant="ghost" onClick={load} disabled={loading} className="inline-flex items-center gap-2">
          <RefreshCw size={14} className={loading ? "animate-spin" : ""} />
          Atualizar
        </Button>
      </div>

      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted pointer-events-none" />
          <input
            placeholder="Buscar por objeto ou contratado…"
            value={busca}
            onChange={(e) => {
              setPage(1);
              setBusca(e.target.value);
            }}
            className="w-full rounded-md border border-surface-border bg-surface pl-10 pr-3 py-2 text-sm text-foreground"
          />
        </div>
        <select
          value={status}
          onChange={(e) => {
            setPage(1);
            setStatus(e.target.value);
          }}
          className="rounded-md border border-surface-border bg-surface px-3 py-2 text-xs font-medium text-foreground"
        >
          {STATUS_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>

      {err && <div className="rounded-lg border border-danger/40 bg-danger/10 p-4 text-sm text-danger">{err}</div>}

      {!err && !loading && items.length === 0 && (
        <div className="rounded-xl border border-dashed border-surface-border bg-surface p-12 text-center text-muted">
          <FileSignature size={32} className="mx-auto text-muted/40 mb-3" />
          <p className="font-semibold text-foreground">Nenhum contrato encontrado.</p>
        </div>
      )}

      <div className="flex flex-col gap-2">
        {items.map((c) => (
          <ContratoRow key={c.id} contrato={c} />
        ))}
      </div>

      {meta && meta.total_pages > 1 && (
        <div className="flex items-center justify-center gap-3 text-xs">
          <Button size="sm" variant="secondary" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            Anterior
          </Button>
          <span className="text-muted">
            Página {meta.page} de {meta.total_pages} · {meta.total_items} contrato(s)
          </span>
          <Button
            size="sm"
            variant="secondary"
            disabled={page >= meta.total_pages}
            onClick={() => setPage((p) => p + 1)}
          >
            Próxima
          </Button>
        </div>
      )}
    </div>
  );
}
