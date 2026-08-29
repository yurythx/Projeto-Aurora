"use client";

import { useEffect, useState } from "react";
import { AlertTriangle, Check, FileText, RefreshCw, Trash2, ArrowUpCircle } from "lucide-react";
import { apiClient } from "@/lib/api/client";
import { pdfUrlWithPage } from "@/lib/typesense-client";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Card, CardContent } from "@/components/ui/Card";
import { useToast } from "@/components/notifications/ToastProvider";

interface ReviewFinding {
  id: string;
  act_type: string;
  servidor_nome?: string;
  empresa_nome?: string;
  cpf?: string;
  matricula?: string;
  cnpj?: string;
  valor?: number;
  secretaria?: string;
  job_role?: string;
  das_level?: string;
  portaria_number?: string;
  raw_content: string;
  pdf_page_number: number;
  edition_number?: string;
  pdf_url?: string;
  confidence: string;
}

const ACT_TYPES = [
  "NOMEACAO_EFETIVO",
  "NOMEACAO_COMISSIONADO",
  "CONTRATACAO_TEMPORARIA",
  "EXONERACAO",
  "RESCISAO",
  "RELOTACAO",
  "DESIGNACAO_FUNCAO",
  "CONTRATO",
  "OUTROS",
];

type Draft = {
  act_type: string;
  confidence: string;
  servidor_nome: string;
  cpf: string;
  matricula: string;
  empresa_nome: string;
  cnpj: string;
  valor: string;
  secretaria: string;
  job_role: string;
  das_level: string;
  portaria_number: string;
  review_note: string;
};

function toDraft(f: ReviewFinding): Draft {
  return {
    act_type: ACT_TYPES.includes(f.act_type) ? f.act_type : "OUTROS",
    confidence: "medium",
    servidor_nome: f.servidor_nome ?? "",
    cpf: f.cpf ?? "",
    matricula: f.matricula ?? "",
    empresa_nome: f.empresa_nome ?? "",
    cnpj: f.cnpj ?? "",
    valor: f.valor != null ? String(f.valor) : "",
    secretaria: f.secretaria ?? "",
    job_role: f.job_role ?? "",
    das_level: f.das_level ?? "",
    portaria_number: f.portaria_number ?? "",
    review_note: "",
  };
}

const orNull = (s: string) => {
  const t = s.trim();
  return t === "" ? null : t;
};

export default function RevisaoPage() {
  const { showToast } = useToast();
  const [items, setItems] = useState<ReviewFinding[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  const [drafts, setDrafts] = useState<Record<string, Draft>>({});
  const [busy, setBusy] = useState<string | null>(null);

  const load = async () => {
    setLoading(true);
    setErr(null);
    try {
      const res = await apiClient.get<ReviewFinding[]>(
        "v1/diario-oficial/rondonopolis/review-queue?limit=200",
      );
      const list = Array.isArray(res.data) ? res.data : [];
      setItems(list);
      setDrafts(Object.fromEntries(list.map((f) => [f.id, toDraft(f)])));
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Falha ao carregar a fila de revisão.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const patchDraft = (id: string, part: Partial<Draft>) =>
    setDrafts((cur) => {
      const base = cur[id];
      if (!base) return cur;
      return { ...cur, [id]: { ...base, ...part } };
    });

  const drop = (id: string) => setItems((xs) => xs.filter((x) => x.id !== id));

  const promote = async (f: ReviewFinding) => {
    const d = drafts[f.id];
    if (!d) return;
    setBusy(f.id);
    try {
      await apiClient.patch(`v1/diario-oficial/rondonopolis/review-queue/${f.id}`, {
        act_type: d.act_type,
        confidence: d.confidence,
        servidor_nome: orNull(d.servidor_nome),
        cpf: orNull(d.cpf),
        matricula: orNull(d.matricula),
        empresa_nome: orNull(d.empresa_nome),
        cnpj: orNull(d.cnpj),
        valor: d.valor.trim() === "" ? null : Number(d.valor.replace(",", ".")),
        secretaria: orNull(d.secretaria),
        job_role: orNull(d.job_role),
        das_level: orNull(d.das_level),
        portaria_number: orNull(d.portaria_number),
        review_note: d.review_note,
      });
      drop(f.id);
      showToast({ title: "Finding promovido", description: "Já entra na busca do usuário.", tone: "success" });
    } catch (e) {
      showToast({
        title: "Não foi possível promover",
        description: e instanceof Error ? e.message : undefined,
        tone: "danger",
      });
    } finally {
      setBusy(null);
    }
  };

  const acknowledge = async (f: ReviewFinding) => {
    setBusy(f.id);
    try {
      await apiClient.post(`v1/diario-oficial/rondonopolis/review-queue/${f.id}/ack`, {
        review_note: drafts[f.id]?.review_note ?? "",
      });
      drop(f.id);
      showToast({ title: "Marcado como revisado", description: "Sai da fila, permanece só no PostgreSQL.", tone: "info" });
    } catch (e) {
      showToast({ title: "Falha ao marcar", description: e instanceof Error ? e.message : undefined, tone: "danger" });
    } finally {
      setBusy(null);
    }
  };

  const discard = async (f: ReviewFinding) => {
    if (!confirm("Descartar este finding em definitivo? (volta se a edição for reprocessada)")) return;
    setBusy(f.id);
    try {
      await apiClient.delete(`v1/diario-oficial/rondonopolis/review-queue/${f.id}`);
      drop(f.id);
      showToast({ title: "Finding descartado", tone: "info" });
    } catch (e) {
      showToast({ title: "Falha ao descartar", description: e instanceof Error ? e.message : undefined, tone: "danger" });
    } finally {
      setBusy(null);
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 border-b border-surface-border pb-4">
        <div>
          <h1 className="text-xl font-bold text-foreground flex items-center gap-2">
            <AlertTriangle className="h-6 w-6 text-amber-500" />
            Fila de Revisão — Extrações de Baixa Confiança
          </h1>
          <p className="text-xs text-muted mt-1">
            Findings <strong>confidence=low</strong>, fora da busca do usuário. Corrija e{" "}
            <strong>promova</strong> para indexar, <strong>marque como revisado</strong> se for
            legítimo mas fraco, ou <strong>descarte</strong> se for ruído do parser.
          </p>
        </div>
        <Button size="sm" variant="ghost" onClick={load} disabled={loading} className="inline-flex items-center gap-2">
          <RefreshCw size={14} className={loading ? "animate-spin" : ""} />
          Atualizar
        </Button>
      </div>

      {err && <div className="rounded-lg border border-danger/40 bg-danger/10 p-4 text-sm text-danger">{err}</div>}

      {!err && !loading && items.length === 0 && (
        <div className="rounded-xl border border-dashed border-surface-border bg-surface p-12 text-center text-muted">
          <AlertTriangle size={32} className="mx-auto text-muted/40 mb-3" />
          <p className="font-semibold text-foreground">Nenhum finding pendente de revisão.</p>
          <p className="text-xs mt-1">O pipeline descarta ruído a montante — fila vazia é o esperado.</p>
        </div>
      )}

      <div className="flex flex-col gap-3">
        {items.length > 0 && <span className="text-xs text-muted">{items.length} registro(s)</span>}
        {items.map((f) => {
          const d = drafts[f.id];
          if (!d) return null;
          const isBusy = busy === f.id;
          return (
            <Card key={f.id}>
              <CardContent className="p-4 flex flex-col gap-3">
                <div className="flex flex-wrap items-center gap-2 text-xs">
                  <Badge tone="warning">{f.confidence}</Badge>
                  {f.edition_number && <span className="font-mono text-muted">Edição #{f.edition_number}</span>}
                  {f.pdf_url && (
                    <a
                      href={pdfUrlWithPage(f.pdf_url, f.pdf_page_number)}
                      target="_blank"
                      rel="noreferrer"
                      className="text-primary hover:underline inline-flex items-center gap-1"
                    >
                      <FileText size={12} />
                      {f.pdf_page_number > 1 ? `PDF (pág. ${f.pdf_page_number})` : "PDF oficial"} →
                    </a>
                  )}
                </div>

                <p className="text-xs text-muted line-clamp-3 border-l-2 border-amber-400 pl-3 bg-surface-hover/40 py-2">
                  {f.raw_content}
                </p>

                <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                  <label className="flex flex-col gap-1 text-xs text-muted">
                    Tipo de ato
                    <select
                      value={d.act_type}
                      onChange={(e) => patchDraft(f.id, { act_type: e.target.value })}
                      className="rounded border border-surface-border bg-surface px-2 py-1 text-sm text-foreground"
                    >
                      {ACT_TYPES.map((t) => (
                        <option key={t} value={t}>
                          {t.replace(/_/g, " ")}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="flex flex-col gap-1 text-xs text-muted">
                    Confiança
                    <select
                      value={d.confidence}
                      onChange={(e) => patchDraft(f.id, { confidence: e.target.value })}
                      className="rounded border border-surface-border bg-surface px-2 py-1 text-sm text-foreground"
                    >
                      <option value="medium">medium</option>
                      <option value="high">high</option>
                    </select>
                  </label>
                  {(
                    [
                      ["servidor_nome", "Servidor"],
                      ["cpf", "CPF"],
                      ["matricula", "Matrícula"],
                      ["empresa_nome", "Empresa"],
                      ["cnpj", "CNPJ"],
                      ["valor", "Valor"],
                      ["secretaria", "Secretaria"],
                      ["job_role", "Cargo/Função"],
                      ["das_level", "DAS"],
                      ["portaria_number", "Portaria/Ref."],
                    ] as [keyof Draft, string][]
                  ).map(([k, label]) => (
                    <label key={k} className="flex flex-col gap-1 text-xs text-muted">
                      {label}
                      <input
                        value={d[k]}
                        onChange={(e) => patchDraft(f.id, { [k]: e.target.value } as Partial<Draft>)}
                        className="rounded border border-surface-border bg-surface px-2 py-1 text-sm text-foreground"
                      />
                    </label>
                  ))}
                </div>

                <input
                  value={d.review_note}
                  onChange={(e) => patchDraft(f.id, { review_note: e.target.value })}
                  placeholder="Nota de revisão (opcional)"
                  className="rounded border border-surface-border bg-surface px-2 py-1 text-sm text-foreground"
                />

                <div className="flex flex-wrap gap-2 pt-1">
                  <Button size="sm" onClick={() => promote(f)} disabled={isBusy} className="inline-flex items-center gap-1.5">
                    <ArrowUpCircle size={14} /> Promover
                  </Button>
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() => acknowledge(f)}
                    disabled={isBusy}
                    className="inline-flex items-center gap-1.5"
                  >
                    <Check size={14} /> Marcar como revisado
                  </Button>
                  <Button
                    size="sm"
                    variant="danger"
                    onClick={() => discard(f)}
                    disabled={isBusy}
                    className="inline-flex items-center gap-1.5"
                  >
                    <Trash2 size={14} /> Descartar
                  </Button>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
