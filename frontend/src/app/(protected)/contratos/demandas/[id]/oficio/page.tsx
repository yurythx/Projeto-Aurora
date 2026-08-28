"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Printer, ArrowLeft, Building2, CheckCircle2 } from "lucide-react";
import { apiClient } from "@/lib/api/client";
import type { DemandResponse } from "@/types/api";
import { Button } from "@/components/ui/Button";

export default function OficioPDFPage() {
  const params = useParams();
  const demandId = params?.id as string;

  const [demand, setDemand] = useState<DemandResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadDemand() {
      if (!demandId) return;
      try {
        const { data } = await apiClient.get<DemandResponse>(`/api/v1/demands/${demandId}`);
        setDemand(data);
      } catch (err: any) {
        setError(err.message || "Erro ao carregar os dados da demanda.");
      } finally {
        setLoading(false);
      }
    }
    loadDemand();
  }, [demandId]);

  if (loading) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center gap-3 p-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        <p className="text-sm text-muted">Gerando documento oficial para impressão...</p>
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

  const currentDate = new Date().toLocaleDateString("pt-BR", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });

  return (
    <div className="min-h-screen bg-slate-100 p-4 md:p-8 font-sans text-slate-900 print:bg-white print:p-0">
      {/* Top Action Bar (Escondida ao imprimir) */}
      <div className="mx-auto mb-6 flex max-w-4xl items-center justify-between rounded-xl bg-white p-4 shadow-sm border border-slate-200 print:hidden">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => window.history.back()}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Voltar
          </Button>
          <span className="text-xs text-slate-500">Documento Oficial de Ordem de Fornecimento / Pré-Empenho</span>
        </div>

        <Button onClick={() => window.print()} variant="primary" size="sm">
          <Printer className="mr-2 h-4 w-4" />
          Imprimir / Salvar em PDF
        </Button>
      </div>

      {/* Folha do Documento A4 */}
      <div className="mx-auto max-w-4xl bg-white p-10 shadow-lg border border-slate-200 print:shadow-none print:border-none print:max-w-none print:p-0">
        {/* Cabeçalho Oficial */}
        <div className="flex items-center justify-between border-b-2 border-slate-900 pb-6">
          <div className="flex items-center gap-4">
            <div className="flex h-14 w-14 items-center justify-center rounded-lg bg-slate-900 text-white font-bold text-xl">
              PMR
            </div>
            <div>
              <h1 className="text-base font-bold uppercase tracking-wide text-slate-900">
                Prefeitura Municipal de Rondonópolis
              </h1>
              <p className="text-xs text-slate-600 font-medium">Secretaria Municipal de Administração e Gestão de Contratos</p>
              <p className="text-[11px] text-slate-500">Instrução Normativa SCL nº 01/2019 • Sistema de Controle Interno</p>
            </div>
          </div>

          <div className="text-right font-mono text-xs">
            <div className="font-bold text-slate-900 text-sm">OF Nº {demand.id.substring(0, 8).toUpperCase()}/2026</div>
            <div className="text-slate-500">Competência: {demand.ano_mes}</div>
            <div className="text-slate-500">Data: {currentDate}</div>
          </div>
        </div>

        {/* Título do Documento */}
        <div className="my-8 text-center">
          <h2 className="text-lg font-bold uppercase tracking-wider text-slate-900 border-b border-slate-300 pb-2 inline-block">
            Ordem de Fornecimento & Autorização de Pré-Empenho
          </h2>
        </div>

        {/* Quadro de Informações do Contrato */}
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

          {demand.contrato_valor && (
            <div>
              <span className="font-bold text-slate-700 block uppercase">Valor Estimado / Teto Mensal:</span>
              <span className="text-slate-900 font-mono font-bold text-sm">
                R$ {demand.contrato_valor.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
              </span>
            </div>
          )}
        </div>

        {/* Texto de Autorização */}
        <div className="mb-8 text-xs leading-relaxed text-slate-800 space-y-3 text-justify">
          <p>
            Pelo presente instrumento, a Secretaria Municipal autoriza a emissão da Nota de Empenho referente à medição/demanda mensal do mês de <strong>{demand.ano_mes}</strong>, em conformidade com as cláusulas estipuladas no contrato administrativo nº <strong>{demand.contrato_numero}</strong>.
          </p>
          <p>
            O fornecedor contratado fica notificado a manter a regularidade de todas as certidões fiscais e trabalhistas exigidas para a efetivação da liquidação financeira e pagamento.
          </p>
          {demand.observacoes && (
            <div className="rounded border border-amber-200 bg-amber-50 p-3 italic text-amber-900">
              <strong>Observações da Demanda:</strong> {demand.observacoes}
            </div>
          )}
        </div>

        {/* Histórico de Documentos Anexados / Checklist */}
        <div className="mb-10 text-xs">
          <h3 className="font-bold uppercase tracking-wider text-slate-900 mb-3 border-b border-slate-200 pb-1">
            Checklist de Documentação Exigida (IN SCL 01/2019)
          </h3>
          {demand.documents && demand.documents.length > 0 ? (
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-slate-300 bg-slate-100 text-slate-700">
                  <th className="p-2 font-bold">Tipo de Documento</th>
                  <th className="p-2 font-bold">Arquivo</th>
                  <th className="p-2 font-bold">Data de Envio</th>
                  <th className="p-2 font-bold text-center">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200">
                {demand.documents.map((doc) => (
                  <tr key={doc.id}>
                    <td className="p-2 font-semibold text-slate-900">{doc.doc_type}</td>
                    <td className="p-2 text-slate-600 font-mono">{doc.file_name}</td>
                    <td className="p-2 text-slate-600 font-mono">
                      {new Date(doc.uploaded_at).toLocaleDateString("pt-BR")}
                    </td>
                    <td className="p-2 text-center text-emerald-700 font-bold">✓ Homologado</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="italic text-slate-500">Pendente de anexação de certidões e comprovantes na plataforma.</p>
          )}
        </div>

        {/* Campo para Assinatura Formal dos Responsáveis */}
        <div className="mt-16 grid grid-cols-2 gap-12 pt-8 border-t border-slate-300 text-center text-xs">
          <div>
            <div className="border-b border-slate-900 mx-auto w-4/5 mb-2" />
            <p className="font-bold text-slate-900 uppercase">Fiscal do Contrato</p>
            <p className="text-slate-600">Portaria de Nomeação nº 491/2025</p>
            <p className="text-[10px] text-slate-400 mt-0.5">Assinatura Digital / Matrícula</p>
          </div>

          <div>
            <div className="border-b border-slate-900 mx-auto w-4/5 mb-2" />
            <p className="font-bold text-slate-900 uppercase">Ordenador de Despesa / Secretário</p>
            <p className="text-slate-600">Prefeitura Municipal de Rondonópolis</p>
            <p className="text-[10px] text-slate-400 mt-0.5">Assinatura Digital / Carimbo</p>
          </div>
        </div>

        {/* Rodapé do Documento */}
        <div className="mt-12 border-t border-slate-200 pt-4 text-center text-[10px] text-slate-400 font-mono">
          Documento gerado automaticamente pelo Projeto Nova • Código de Autenticidade: {demand.id}
        </div>
      </div>
    </div>
  );
}
