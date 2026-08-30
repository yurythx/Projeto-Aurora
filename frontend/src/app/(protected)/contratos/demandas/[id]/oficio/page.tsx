"use client";

import { OfficialDocShell } from "../_components/OfficialDocShell";

// Ofício ao Planejamento / OF (§ etapa 1 do fluxo IN SCL 01/2019). Usa a
// mesma casca dos outros documentos oficiais (OfficialDocShell) — antes
// esta página reimplementava cabeçalho, folha A4, bloco de assinatura e
// rodapé por conta própria, e por isso ficou de fora do redesenho do
// selo/marca. Só o corpo é específico daqui.
export default function OficioPDFPage() {
  return (
    <OfficialDocShell
      docTitle="Ordem de Fornecimento & Autorização de Pré-Empenho"
      docTag="OF"
      minEtapa={1}
      signatures={[
        { role: "Fiscal do Contrato", line1: "Portaria de Nomeação nº 491/2025", line2: "Assinatura / Matrícula" },
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
            Pelo presente instrumento, a Secretaria Municipal autoriza a emissão da Nota de Empenho
            referente à medição/demanda mensal do mês de <strong>{demand.ano_mes}</strong>, em
            conformidade com as cláusulas estipuladas no contrato administrativo nº{" "}
            <strong>{demand.contrato_numero || "N/A"}</strong>.
          </p>
          <p>
            O fornecedor contratado fica notificado a manter a regularidade de todas as certidões
            fiscais e trabalhistas exigidas para a efetivação da liquidação financeira e pagamento.
          </p>

          {demand.observacoes && (
            <div className="rounded border border-amber-200 bg-amber-50 p-3 italic text-amber-900">
              <strong>Observações da Demanda:</strong> {demand.observacoes}
            </div>
          )}

          <div>
            <h3 className="mb-2 border-b border-slate-200 pb-1 font-bold uppercase tracking-wider text-slate-900">
              Checklist de documentação exigida (IN SCL 01/2019)
            </h3>
            {demand.documents && demand.documents.length > 0 ? (
              <table className="w-full border-collapse text-left text-[11px]">
                <thead>
                  <tr className="border-b border-slate-300 bg-slate-100 text-slate-700">
                    <th className="p-2 font-bold">Tipo de Documento</th>
                    <th className="p-2 font-bold">Arquivo</th>
                    <th className="p-2 font-bold">Data de Envio</th>
                    <th className="p-2 text-center font-bold">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {demand.documents.map((doc) => (
                    <tr key={doc.id}>
                      <td className="p-2 font-semibold text-slate-900">{doc.doc_type}</td>
                      <td className="p-2 font-mono text-slate-600">{doc.file_name}</td>
                      <td className="p-2 font-mono text-slate-600">
                        {new Date(doc.uploaded_at).toLocaleDateString("pt-BR")}
                      </td>
                      <td className="p-2 text-center font-bold text-emerald-700">✓ Homologado</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p className="italic text-slate-500">
                Pendente de anexação de certidões e comprovantes na plataforma.
              </p>
            )}
          </div>
        </div>
      )}
    </OfficialDocShell>
  );
}
