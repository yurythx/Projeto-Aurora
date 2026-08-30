"use client";

import { useEffect, useState, type ReactNode } from "react";
import { useParams } from "next/navigation";
import { Printer, ArrowLeft } from "lucide-react";
import { apiClient } from "@/lib/api/client";
import type { DemandResponse } from "@/types/api";
import { Button } from "@/components/ui/Button";
import { Seal } from "@/components/ui/Seal";

export interface SignatureSlot {
  role: string;
  line1?: string;
  line2?: string;
}

interface Props {
  /** Ex.: "Ordem de Serviço", "Relatório Mensal de Fiscalização (Anexo I)". */
  docTitle: string;
  /** Rótulo curto na barra de ação e no cabeçalho (ex.: "OS", "ANEXO I"). */
  docTag: string;
  /** Etapa mínima da demanda para o documento fazer sentido; abaixo disso
   *  mostra um aviso em vez do corpo. */
  minEtapa?: number;
  signatures: SignatureSlot[];
  children: (demand: DemandResponse) => ReactNode;
}

/** Casca comum dos documentos oficiais imprimíveis das demandas (mesma
 * identidade visual do Ofício ao Planejamento já existente): carrega a
 * demanda por id, monta o cabeçalho da PMR, a folha A4, a barra de
 * impressão (some no print), o bloco de assinaturas e o rodapé com código
 * de autenticidade. O conteúdo específico de cada documento entra em
 * children. */
export function OfficialDocShell({ docTitle, docTag, minEtapa = 1, signatures, children }: Props) {
  const params = useParams();
  const demandId = params?.id as string;

  const [demand, setDemand] = useState<DemandResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!demandId) return;
    apiClient
      .get<DemandResponse>(`/api/v1/demands/${demandId}`)
      .then(({ data }) => setDemand(data))
      .catch((err) => setError(err instanceof Error ? err.message : "Erro ao carregar a demanda."))
      .finally(() => setLoading(false));
  }, [demandId]);

  if (loading) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center gap-3 p-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        <p className="text-sm text-muted">Gerando {docTitle} para impressão...</p>
      </div>
    );
  }

  if (error || !demand) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center gap-4 p-8 text-center">
        <p className="text-danger font-semibold">{error || "Demanda não encontrada."}</p>
        <Button onClick={() => window.close()} variant="secondary">
          Fechar Janela
        </Button>
      </div>
    );
  }

  const currentDate = new Date().toLocaleDateString("pt-BR", { day: "numeric", month: "long", year: "numeric" });
  const belowStage = demand.etapa < minEtapa;

  return (
    <div className="min-h-screen bg-slate-100 p-4 md:p-8 font-sans text-slate-900 print:bg-white print:p-0">
      <div className="mx-auto mb-6 flex max-w-4xl items-center justify-between rounded-xl bg-white p-4 shadow-sm border border-slate-200 print:hidden">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => window.history.back()}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Voltar
          </Button>
          <span className="text-xs text-slate-500">{docTitle}</span>
        </div>
        <Button onClick={() => window.print()} variant="primary" size="sm" disabled={belowStage}>
          <Printer className="mr-2 h-4 w-4" />
          Imprimir / Salvar em PDF
        </Button>
      </div>

      <div className="relative mx-auto max-w-4xl overflow-hidden bg-white p-10 shadow-lg border border-slate-200 print:shadow-none print:border-none print:max-w-none print:p-0">
        <div className="flex items-center justify-between border-b-2 border-slate-900 pb-6">
          <div className="flex items-center gap-4">
            {/* Monograma da Prefeitura no papel timbrado — o brasão do
                município (não a marca do software, que fica só no rodapé
                e no selo de autenticidade). */}
            <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-lg border border-slate-900 font-mono text-lg font-bold tracking-tight text-slate-900">
              PMR
            </div>
            <div>
              <h1 className="text-base font-bold uppercase tracking-wide text-slate-900">
                Prefeitura Municipal de Rondonópolis
              </h1>
              <p className="text-xs text-slate-600 font-medium">
                Secretaria Municipal de Administração e Gestão de Contratos
              </p>
              <p className="text-[11px] text-slate-500">
                Instrução Normativa SCL nº 01/2019 • Sistema de Controle Interno
              </p>
            </div>
          </div>
          <div className="text-right font-mono text-xs">
            <div className="font-bold text-slate-900 text-sm">
              {docTag} Nº {demand.id.substring(0, 8).toUpperCase()}/{demand.ano_mes.slice(0, 4)}
            </div>
            <div className="text-slate-500">Competência: {demand.ano_mes}</div>
            <div className="text-slate-500">Data: {currentDate}</div>
          </div>
        </div>

        <div className="my-8 text-center">
          <h2 className="text-lg font-bold uppercase tracking-wider text-slate-900 border-b border-slate-300 pb-2 inline-block">
            {docTitle}
          </h2>
        </div>

        {belowStage ? (
          <div className="rounded-lg border border-amber-300 bg-amber-50 p-6 text-sm text-amber-900">
            Esta demanda está na <strong>Etapa {demand.etapa}</strong>. O documento{" "}
            <strong>{docTitle}</strong> só é emitido a partir da <strong>Etapa {minEtapa}</strong> do fluxo
            (IN SCL 01/2019). Avance o card no Kanban para liberar a geração.
          </div>
        ) : (
          <>
            <div className="mb-8 rounded-lg border border-slate-300 p-5 space-y-3 bg-slate-50 text-xs">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <span className="font-bold text-slate-700 block uppercase">Nº do Contrato:</span>
                  <span className="text-slate-900 font-mono font-semibold">{demand.contrato_numero || "N/A"}</span>
                </div>
                <div>
                  <span className="font-bold text-slate-700 block uppercase">Empresa Contratada / Fornecedor:</span>
                  <span className="text-slate-900 font-semibold">{demand.contratado || "N/A"}</span>
                </div>
              </div>
              <div>
                <span className="font-bold text-slate-700 block uppercase">Objeto Contratual:</span>
                <p className="text-slate-800 leading-relaxed mt-0.5">{demand.contrato_objeto || "Sem descrição"}</p>
              </div>
              {demand.contrato_valor != null && (
                <div>
                  <span className="font-bold text-slate-700 block uppercase">Valor Estimado / Teto Mensal:</span>
                  <span className="text-slate-900 font-mono font-bold text-sm">
                    R$ {demand.contrato_valor.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
                  </span>
                </div>
              )}
            </div>

            <div className="text-xs leading-relaxed text-slate-800">{children(demand)}</div>

            <div
              className="mt-16 grid gap-12 pt-8 border-t border-slate-300 text-center text-xs"
              style={{ gridTemplateColumns: `repeat(${Math.min(signatures.length, 2)}, minmax(0, 1fr))` }}
            >
              {signatures.map((s, i) => (
                <div key={i}>
                  <div className="border-b border-slate-900 mx-auto w-4/5 mb-2" />
                  <p className="font-bold text-slate-900 uppercase">{s.role}</p>
                  {s.line1 && <p className="text-slate-600">{s.line1}</p>}
                  {s.line2 && <p className="text-[10px] text-slate-400 mt-0.5">{s.line2}</p>}
                </div>
              ))}
            </div>
          </>
        )}

        <div className="mt-12 border-t border-slate-200 pt-4 text-center text-[10px] text-slate-400 font-mono">
          Documento gerado automaticamente pelo Projeto Nova • Código de Autenticidade: {demand.id}
        </div>

        {!belowStage && (
          <Seal
            size={150}
            decorative
            className="pointer-events-none absolute bottom-16 right-8 -rotate-6 text-slate-900/[0.07]"
          />
        )}
      </div>
    </div>
  );
}
