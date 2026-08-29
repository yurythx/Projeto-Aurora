"use client";

import { OfficialDocShell } from "../_components/OfficialDocShell";

const CERT_LABELS: Record<string, string> = {
  EMPENHO_ASSINADO: "Nota de Empenho Assinada",
  COMPROVANTE_ENVIO: "Comprovante de Envio ao Fornecedor",
};

export default function OrdemServicoPage() {
  return (
    <OfficialDocShell
      docTitle="Ordem de Serviço / Autorização de Execução"
      docTag="OS"
      minEtapa={3}
      signatures={[
        { role: "Fiscal do Contrato", line1: "Portaria de Designação", line2: "Assinatura / Matrícula" },
        {
          role: "Ordenador de Despesa / Secretário",
          line1: "Prefeitura Municipal de Rondonópolis",
          line2: "Assinatura / Carimbo",
        },
      ]}
    >
      {(demand) => (
        <div className="space-y-4 text-justify">
          <p>
            A Secretaria Municipal, por meio do fiscal do contrato nº{" "}
            <strong>{demand.contrato_numero}</strong>, <strong>AUTORIZA</strong> a empresa{" "}
            <strong>{demand.contratado}</strong> a executar os serviços / fornecer os bens
            referentes à competência <strong>{demand.ano_mes}</strong>, nos termos e quantitativos
            pactuados no instrumento contratual e na respectiva Nota de Empenho.
          </p>

          <table className="w-full text-left border-collapse text-[11px]">
            <thead>
              <tr className="border-b border-slate-300 bg-slate-100 text-slate-700">
                <th className="p-2 font-bold">Item</th>
                <th className="p-2 font-bold">Descrição / Objeto</th>
                <th className="p-2 font-bold text-center">Marca</th>
                <th className="p-2 font-bold text-center">Qtd.</th>
                <th className="p-2 font-bold text-right">Valor Unit.</th>
                <th className="p-2 font-bold text-right">Valor Total</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200">
              <tr>
                <td className="p-2 font-mono">01</td>
                <td className="p-2">{demand.contrato_objeto || "Conforme contrato"}</td>
                <td className="p-2 text-center">—</td>
                <td className="p-2 text-center">1</td>
                <td className="p-2 text-right font-mono">
                  {demand.contrato_valor != null
                    ? demand.contrato_valor.toLocaleString("pt-BR", { minimumFractionDigits: 2 })
                    : "—"}
                </td>
                <td className="p-2 text-right font-mono">
                  {demand.contrato_valor != null
                    ? demand.contrato_valor.toLocaleString("pt-BR", { minimumFractionDigits: 2 })
                    : "—"}
                </td>
              </tr>
            </tbody>
            <tfoot>
              <tr className="border-t border-slate-400 font-bold">
                <td className="p-2" colSpan={5}>
                  TOTAL DA ORDEM DE SERVIÇO (R$)
                </td>
                <td className="p-2 text-right font-mono">
                  {demand.contrato_valor != null
                    ? demand.contrato_valor.toLocaleString("pt-BR", { minimumFractionDigits: 2 })
                    : "—"}
                </td>
              </tr>
            </tfoot>
          </table>

          <div>
            <h3 className="font-bold uppercase tracking-wider text-slate-900 mb-2 border-b border-slate-200 pb-1">
              Documentos de instrução da OS
            </h3>
            {demand.documents && demand.documents.length > 0 ? (
              <ul className="list-disc pl-5 space-y-0.5">
                {demand.documents
                  .filter((d) => d.doc_type in CERT_LABELS)
                  .map((d) => (
                    <li key={d.id}>
                      {CERT_LABELS[d.doc_type]} — <span className="font-mono text-slate-600">{d.file_name}</span> (
                      {new Date(d.uploaded_at).toLocaleDateString("pt-BR")})
                    </li>
                  ))}
                {demand.documents.filter((d) => d.doc_type in CERT_LABELS).length === 0 && (
                  <li className="italic text-slate-500 list-none">
                    Empenho assinado / comprovante de envio ainda não anexados.
                  </li>
                )}
              </ul>
            ) : (
              <p className="italic text-slate-500">Nenhum documento anexado à demanda.</p>
            )}
          </div>

          <p>
            A empresa deverá observar rigorosamente os prazos de entrega/execução e emitir a Nota
            Fiscal correspondente somente após a conclusão, acompanhada do Termo de Recepção do
            sistema legado (AGILE).
          </p>

          {demand.observacoes && (
            <div className="rounded border border-amber-200 bg-amber-50 p-3 italic text-amber-900">
              <strong>Observações da Demanda:</strong> {demand.observacoes}
            </div>
          )}
        </div>
      )}
    </OfficialDocShell>
  );
}
