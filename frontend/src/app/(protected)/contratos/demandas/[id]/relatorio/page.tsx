"use client";

import { OfficialDocShell } from "../_components/OfficialDocShell";

// As 6 certidões obrigatórias da IN SCL 01/2019 + os anexos de instrução do
// relatório. A ordem aqui é a ordem de exibição no checklist.
const REQUIRED_DOCS: { type: string; label: string }[] = [
  { type: "EXTRATO_EMPENHO", label: "Extrato de Empenho (AGILE)" },
  { type: "RELATORIO_PAGAMENTO", label: "Relatório de Pagamento gerado" },
  { type: "CERTIDAO_FEDERAL", label: "Certidão Negativa Federal" },
  { type: "CERTIDAO_ESTADUAL", label: "Certidão Negativa Estadual" },
  { type: "CERTIDAO_MUNICIPAL", label: "Certidão Negativa Municipal" },
  { type: "CERTIDAO_FGTS", label: "Certificado de Regularidade do FGTS" },
  { type: "CERTIDAO_CNDT", label: "Certidão Negativa de Débitos Trabalhistas" },
  { type: "CERTIDAO_SIMPLES", label: "Certidão Simplificada / CND" },
];

export default function RelatorioFiscalizacaoPage() {
  return (
    <OfficialDocShell
      docTitle="Relatório Mensal de Fiscalização (Anexo I — IN SCL 01/2019)"
      docTag="ANEXO I"
      minEtapa={5}
      signatures={[
        {
          role: "Fiscal do Contrato",
          line1: "Nome / Matrícula / Portaria de Designação",
          line2: "Atesto que o objeto foi executado conforme o contrato",
        },
        {
          role: "Gestor do Contrato",
          line1: "Secretaria Municipal",
          line2: "Assinatura / Carimbo",
        },
      ]}
    >
      {(demand) => {
        const byType = new Map((demand.documents ?? []).map((d) => [d.doc_type, d]));
        return (
          <div className="space-y-5 text-justify">
            <p>
              Em cumprimento ao art. 117 da Lei nº 14.133/2021 e à IN SCL nº 01/2019, o fiscal do
              contrato nº <strong>{demand.contrato_numero}</strong> <strong>ATESTA</strong> que a
              empresa <strong>{demand.contratado}</strong> executou o objeto contratual referente à
              competência <strong>{demand.ano_mes}</strong>, e que a documentação de habilitação e
              regularidade fiscal/trabalhista foi conferida.
            </p>

            <div>
              <h3 className="font-bold uppercase tracking-wider text-slate-900 mb-2 border-b border-slate-200 pb-1">
                Apuração de valores e retenções
              </h3>
              <table className="w-full text-left border-collapse text-[11px]">
                <tbody className="divide-y divide-slate-200">
                  <tr>
                    <td className="p-2 font-semibold w-2/3">Valor bruto da medição / nota fiscal (R$)</td>
                    <td className="p-2 text-right font-mono">
                      {demand.contrato_valor != null
                        ? demand.contrato_valor.toLocaleString("pt-BR", { minimumFractionDigits: 2 })
                        : "____________"}
                    </td>
                  </tr>
                  {["ISSQN", "INSS", "IRRF", "Outras retenções"].map((r) => (
                    <tr key={r}>
                      <td className="p-2">(–) {r}</td>
                      <td className="p-2 text-right font-mono text-slate-400">____________</td>
                    </tr>
                  ))}
                  <tr className="border-t border-slate-400 font-bold">
                    <td className="p-2">VALOR LÍQUIDO A PAGAR (R$)</td>
                    <td className="p-2 text-right font-mono">____________</td>
                  </tr>
                </tbody>
              </table>
              <p className="text-[10px] text-slate-400 mt-1">
                Preencher conforme a planilha de medição e a guia DAM (ISSQN) para contratos de serviço.
              </p>
            </div>

            <div>
              <h3 className="font-bold uppercase tracking-wider text-slate-900 mb-2 border-b border-slate-200 pb-1">
                Checklist de conformidade (Anexo I)
              </h3>
              <table className="w-full text-left border-collapse text-[11px]">
                <thead>
                  <tr className="border-b border-slate-300 bg-slate-100 text-slate-700">
                    <th className="p-2 font-bold">Documento</th>
                    <th className="p-2 font-bold">Arquivo</th>
                    <th className="p-2 font-bold text-center">Validade</th>
                    <th className="p-2 font-bold text-center">Situação</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {REQUIRED_DOCS.map(({ type, label }) => {
                    const doc = byType.get(type);
                    const expired =
                      doc?.validade_ate != null && new Date(doc.validade_ate) < new Date();
                    return (
                      <tr key={type}>
                        <td className="p-2 font-semibold text-slate-900">{label}</td>
                        <td className="p-2 font-mono text-slate-600">{doc?.file_name ?? "—"}</td>
                        <td className="p-2 text-center font-mono text-slate-600">
                          {doc?.validade_ate
                            ? new Date(doc.validade_ate).toLocaleDateString("pt-BR")
                            : "—"}
                        </td>
                        <td
                          className={`p-2 text-center font-bold ${
                            !doc ? "text-red-600" : expired ? "text-amber-600" : "text-emerald-700"
                          }`}
                        >
                          {!doc ? "✗ Ausente" : expired ? "! Vencida" : "✓ OK"}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {demand.observacoes && (
              <div className="rounded border border-amber-200 bg-amber-50 p-3 italic text-amber-900">
                <strong>Observações da Demanda:</strong> {demand.observacoes}
              </div>
            )}

            <div className="mt-6 rounded border border-slate-300 p-3 text-[11px] bg-slate-50">
              <p className="font-bold uppercase text-slate-700">Carimbo eletrônico de fiscalização</p>
              <p>Contrato: {demand.contrato_numero} &nbsp;|&nbsp; Competência: {demand.ano_mes}</p>
              <p>Fiscal: ____________________________ &nbsp; Matrícula: ______________</p>
              <p>Portaria de designação nº ____________ &nbsp; Data: ____/____/________</p>
            </div>
          </div>
        );
      }}
    </OfficialDocShell>
  );
}
