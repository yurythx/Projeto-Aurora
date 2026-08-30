"use client";

import { useMemo, useState } from "react";
import { Search, FileText, Building2, User, RefreshCw, Calendar, FileCheck, Eye, Copy, ExternalLink, X, ShieldCheck, Download } from "lucide-react";

import { useToast } from "@/components/notifications/ToastProvider";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { ModalShell } from "@/components/ui/ModalShell";
import { useApiQuery } from "@/lib/api/swr";
import type { PublicContract } from "@/types/api";
import { RONDONOPOLIS_EDITIONS } from "./RondonopolisHREventsFeed";

const ALL = "ALL";

/** Normaliza texto para busca insensível a acentos e maiúsculas */
function normalizeText(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase();
}

export const DEFAULT_PUBLIC_CONTRACTS: PublicContract[] = [
  {
    id: "contract-demo-1",
    contract_number: "Contrato nº 140/2026",
    contract_type: "Tecnologia da Informação",
    status: "NOVO",
    value: "R$ 950.000,00",
    contractor: "TechGov Soluções em Tecnologia LTDA",
    contractor_cnpj: "03.492.110/0001-99",
    fiscal_nome: "YURI SILVA SANTOS",
    fiscal_cpf: "021.946.881-88",
    fiscal_matricula: "MAT-2025-081",
    suplente_nome: "CARLOS EDUARDO OLIVEIRA",
    suplente_cpf: "419.012.331-00",
    suplente_matricula: "MAT-2024-112",
    portaria_number: "PORT-491/2026",
    edition_number: "6263",
    publication_date: new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString(),
    nomeacao_date: new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString(),
    object: "Contratação de empresa especializada para prestação de serviços continuados de modernização tecnológica, suporte a sistemas de gestão e governança digital municipal.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
  {
    id: "contract-demo-2",
    contract_number: "Contrato nº 440/2026",
    contract_type: "Consultoria & Gestão Fiscal",
    status: "ATIVO",
    value: "R$ 840.000,00",
    contractor: "Soluções em Engenharia & Gestão Fiscal LTDA",
    contractor_cnpj: "18.204.991/0001-10",
    fiscal_nome: "ANA PAULA MARTINS",
    fiscal_cpf: "551.492.001-99",
    fiscal_matricula: "MAT-2026-015",
    suplente_nome: "ROBERTO GOMES DA SILVA",
    suplente_cpf: "302.881.009-55",
    suplente_matricula: "MAT-2023-490",
    portaria_number: "PORT-455/2026",
    edition_number: "6262",
    publication_date: new Date(Date.now() - 3 * 24 * 3600 * 1000).toISOString(),
    nomeacao_date: new Date(Date.now() - 3 * 24 * 3600 * 1000).toISOString(),
    object: "Serviços de acompanhamento técnico, fiscalização financeira e auditoria de receitas municipais para a Secretaria Municipal de Fazenda.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
  {
    id: "contract-demo-3",
    contract_number: "Contrato nº 155/2026",
    contract_type: "Serviços Urbanos & Conservação",
    status: "ATIVO",
    value: "R$ 2.450.000,00",
    contractor: "EcoLimpeza Urbana e Serviços Eireli",
    contractor_cnpj: "29.118.402/0001-88",
    fiscal_nome: "ROBERTO GOMES DA SILVA",
    fiscal_cpf: "302.881.009-55",
    fiscal_matricula: "MAT-2023-490",
    suplente_nome: "MARIA FERNANDA ALVES",
    suplente_cpf: "771.309.112-44",
    suplente_matricula: "MAT-2026-004",
    portaria_number: "PORT-462/2026",
    edition_number: "6261",
    publication_date: new Date(Date.now() - 4 * 24 * 3600 * 1000).toISOString(),
    nomeacao_date: new Date(Date.now() - 4 * 24 * 3600 * 1000).toISOString(),
    object: "Serviços continuados de conservação, roçada, limpeza de vias públicas e manutenção de áreas verdes do município de Rondonópolis.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
];

export function RondonopolisContractsFeed() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedType, setSelectedType] = useState<string>(ALL);
  const [selectedStatus, setSelectedStatus] = useState<string>(ALL);
  const [selectedYear, setSelectedYear] = useState<string>(ALL);
  const [selectedEdition, setSelectedEdition] = useState<string>(ALL);
  const [selectedContract, setSelectedContract] = useState<PublicContract | null>(null);

  // Consulta API com filtro textual passado ao backend
  const { data: contracts, error, isLoading, mutate } = useApiQuery<PublicContract[]>(
    `v1/diario-oficial/rondonopolis/contracts${searchTerm.trim() ? `?search=${encodeURIComponent(searchTerm.trim())}` : ""}`
  );
  const { showToast } = useToast();

  const displayContracts = useMemo(() => {
    return Array.isArray(contracts) ? contracts : [];
  }, [contracts]);

  // Filtragem local otimizada por Nome (Fiscal/Suplente), CPF, Matrícula, CNPJ, Empresa, Portaria e Edição
  const filteredContracts = useMemo(() => {
    const activeList = Array.isArray(displayContracts) ? displayContracts : [];

    const query = normalizeText(searchTerm.trim());
    const queryDigits = query.replace(/\D/g, "");
    const isSearching = query.length > 0;

    return activeList.filter((c) => {
      // Filtro por Ano (se selecionado)
      if (selectedYear !== ALL) {
        const dateStr = c.nomeacao_date || c.publication_date;
        const cYear = dateStr ? new Date(dateStr).getFullYear().toString() : "2026";
        if (cYear !== selectedYear) return false;
      }

      // Filtro por Edição específica (se selecionada)
      if (selectedEdition !== ALL) {
        if (c.edition_number && c.edition_number !== selectedEdition) {
          return false;
        }
      }

      // Filtro por tipo de contrato
      if (selectedType !== ALL && c.contract_type !== selectedType) {
        return false;
      }

      // Filtro por status
      if (selectedStatus !== ALL && c.status !== selectedStatus) {
        return false;
      }

      // Busca por texto (Fiscal Nome/CPF/Matrícula, Suplente Nome/CPF/Matrícula, Empresa/CNPJ, Contrato, Portaria, Edição)
      if (isSearching) {
        const fiscalCpfDigits = (c.fiscal_cpf || "").replace(/\D/g, "");
        const fiscalMatDigits = (c.fiscal_matricula || "").replace(/\D/g, "");
        const matchesFiscal =
          normalizeText(c.fiscal_nome || "").includes(query) ||
          normalizeText(c.fiscal_cpf || "").includes(query) ||
          (queryDigits.length >= 3 && fiscalCpfDigits.includes(queryDigits)) ||
          normalizeText(c.fiscal_matricula || "").includes(query) ||
          (queryDigits.length >= 3 && fiscalMatDigits.includes(queryDigits));

        const suplenteCpfDigits = (c.suplente_cpf || "").replace(/\D/g, "");
        const suplenteMatDigits = (c.suplente_matricula || "").replace(/\D/g, "");
        const matchesSuplente =
          normalizeText(c.suplente_nome || "").includes(query) ||
          normalizeText(c.suplente_cpf || "").includes(query) ||
          (queryDigits.length >= 3 && suplenteCpfDigits.includes(queryDigits)) ||
          normalizeText(c.suplente_matricula || "").includes(query) ||
          (queryDigits.length >= 3 && suplenteMatDigits.includes(queryDigits));

        const contractorCnpjDigits = (c.contractor_cnpj || "").replace(/\D/g, "");
        const matchesContractor =
          normalizeText(c.contractor || "").includes(query) ||
          normalizeText(c.contractor_cnpj || "").includes(query) ||
          (queryDigits.length >= 3 && contractorCnpjDigits.includes(queryDigits));

        const matchesDetails =
          normalizeText(c.contract_number || "").includes(query) ||
          normalizeText(c.portaria_number || "").includes(query) ||
          normalizeText(c.contract_type || "").includes(query) ||
          normalizeText(c.edition_number || "").includes(query) ||
          normalizeText(c.object || "").includes(query);

        return matchesFiscal || matchesSuplente || matchesContractor || matchesDetails;
      }

      return true;
    });
  }, [contracts, searchTerm, selectedType, selectedStatus, selectedYear, selectedEdition]);

  // Contratos relacionados ao mesmo CPF ou CNPJ selecionado no modal
  const selectedRelatedContracts = useMemo(() => {
    if (!selectedContract) return [];

    const targetFiscalCpf = (selectedContract.fiscal_cpf || "").replace(/\D/g, "");
    const targetSuplenteCpf = (selectedContract.suplente_cpf || "").replace(/\D/g, "");
    const targetCnpj = (selectedContract.contractor_cnpj || "").replace(/\D/g, "");

    return displayContracts.filter((c) => {
      const fCpf = (c.fiscal_cpf || "").replace(/\D/g, "");
      const sCpf = (c.suplente_cpf || "").replace(/\D/g, "");
      const cnpj = (c.contractor_cnpj || "").replace(/\D/g, "");

      return (
        (targetFiscalCpf && (fCpf === targetFiscalCpf || sCpf === targetFiscalCpf)) ||
        (targetSuplenteCpf && (fCpf === targetSuplenteCpf || sCpf === targetSuplenteCpf)) ||
        (targetCnpj && cnpj === targetCnpj)
      );
    });
  }, [selectedContract, displayContracts]);

  const contractTypes = useMemo(() => {
    return Array.from(new Set(displayContracts.map((c) => c.contract_type).filter(Boolean)));
  }, [displayContracts]);

  const hasActiveFilters = searchTerm.trim() !== "" || selectedType !== ALL || selectedStatus !== ALL || selectedYear !== ALL || selectedEdition !== ALL;

  function handleResetFilters() {
    setSearchTerm("");
    setSelectedType(ALL);
    setSelectedStatus(ALL);
    setSelectedYear(ALL);
    setSelectedEdition(ALL);
  }

  function handleCopyContractCertidao(c: PublicContract) {
    const textToCopy = `[EXTRATO DE AUDITORIA DE CONTRATO PÚBLICO - RONDONÓPOLIS-MT]
Contrato: ${c.contract_number} (${c.contract_type})
Status: ${c.status} | Valor: ${c.value}
Portaria de Nomeação dos Fiscais: ${c.portaria_number || "N/A"}
Contratada: ${c.contractor} (CNPJ: ${c.contractor_cnpj})
Fiscal Titular: ${c.fiscal_nome} (CPF: ${c.fiscal_cpf || "N/A"}, Matrícula: ${c.fiscal_matricula || "N/A"})
Fiscal Suplente: ${c.suplente_nome || "N/A"} (CPF: ${c.suplente_cpf || "N/A"}, Matrícula: ${c.suplente_matricula || "N/A"})
Edição do Diário: Nº ${c.edition_number || "N/A"}
Objeto: ${c.object}
Link do PDF: ${c.doc_url || "N/A"}`;

    navigator.clipboard.writeText(textToCopy);
    showToast({ title: "Copiado para a área de transferência", description: "Extrato contratual copiado com sucesso." });
  }

  const activeEditionObj = RONDONOPOLIS_EDITIONS.find((e) => e.number === selectedEdition);

  return (
    <div className="flex flex-col gap-6">
      {/* Barra de Seleção de Ano e Edição do Diário Oficial para Contratos */}
      <div className="flex flex-col gap-3 rounded-lg border border-surface-border bg-surface p-4">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border pb-3">
          <div className="flex items-center gap-2">
            <Calendar size={16} className="text-primary" />
            <span className="text-sm font-semibold text-foreground">Filtro de Histórico e Edições de Contratos:</span>
            <span className="text-xs font-medium px-2 py-0.5 rounded bg-primary/10 text-primary">
              {selectedYear !== ALL ? `Ano ${selectedYear}` : "Todos os Anos"} ·{" "}
              {selectedEdition === ALL
                ? "Todas as Edições"
                : activeEditionObj?.label || `Edição Nº ${selectedEdition}`}
            </span>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => {
                if (!filteredContracts || filteredContracts.length === 0) {
                  showToast({ title: "Sem dados", description: "Nenhum contrato para exportar.", tone: "info" });
                  return;
                }
                const headers = ["Numero_Contrato", "Tipo", "Status", "Valor", "Empresa", "CNPJ", "Fiscal_Titular", "Fiscal_CPF", "Fiscal_Matricula", "Suplente", "Suplente_CPF", "Portaria", "Edicao", "Objeto"];
                const rows = filteredContracts.map((c) => [
                  `"${c.contract_number}"`,
                  `"${c.contract_type}"`,
                  `"${c.status}"`,
                  `"${c.value}"`,
                  `"${c.contractor}"`,
                  `"${c.contractor_cnpj}"`,
                  `"${c.fiscal_nome}"`,
                  `"${c.fiscal_cpf}"`,
                  `"${c.fiscal_matricula}"`,
                  `"${c.suplente_nome || ""}"`,
                  `"${c.suplente_cpf || ""}"`,
                  `"${c.portaria_number || ""}"`,
                  `"${c.edition_number || ""}"`,
                  `"${(c.object || "").replace(/"/g, '""')}"`,
                ]);
                const csvContent = [headers.join(","), ...rows.map((r) => r.join(","))].join("\n");
                const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
                const url = URL.createObjectURL(blob);
                const link = document.createElement("a");
                link.setAttribute("href", url);
                link.setAttribute("download", `relatorio_auditoria_contratos_${new Date().toISOString().slice(0, 10)}.csv`);
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);
                showToast({ title: "Relatório CSV Exportado", description: `${filteredContracts.length} contrato(s) baixado(s) com sucesso.` });
              }}
              className="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline"
            >
              <Download size={14} />
              Exportar Relatório CSV ({filteredContracts.length})
            </button>
            {hasActiveFilters && (
              <button
                onClick={handleResetFilters}
                className="text-xs text-muted hover:text-foreground font-medium"
              >
                Limpar Filtros
              </button>
            )}
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted">Ano do Diário:</span>
            <select
              value={selectedYear}
              onChange={(e) => setSelectedYear(e.target.value)}
              className="rounded-md border border-surface-border bg-surface px-2.5 py-1 text-xs font-medium text-foreground"
            >
              <option value={ALL}>Todos os Anos (Histórico Completo)</option>
              <option value="2026">2026</option>
              <option value="2025">2025</option>
              <option value="2024">2024</option>
            </select>
          </div>

          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-xs font-medium text-muted mr-1">Edição Específica:</span>
            <button
              onClick={() => setSelectedEdition(ALL)}
              className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
                selectedEdition === ALL
                  ? "bg-primary text-white shadow-sm"
                  : "bg-surface-border/50 text-foreground hover:bg-surface-border"
              }`}
            >
              Todas as Edições
            </button>
            {RONDONOPOLIS_EDITIONS.map((ed) => (
              <button
                key={ed.number}
                onClick={() => setSelectedEdition(ed.number)}
                className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
                  selectedEdition === ed.number
                    ? "bg-primary text-white shadow-sm"
                    : "bg-surface-border/50 text-foreground hover:bg-surface-border"
                }`}
              >
                {ed.isLatest ? `⭐ Ed. ${ed.number} (Última)` : `Ed. ${ed.number}`}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Painel de Filtros e Pesquisa Completa de Contratos */}
      <div className="flex flex-col gap-4 rounded-lg border border-surface-border bg-surface p-4">
        <div className="flex flex-col sm:flex-row gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted" />
            <Input
              placeholder="Buscar por Nome do Fiscal/Suplente, CPF, Matrícula, CNPJ ou Empresa..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-9 text-sm"
            />
          </div>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => {
              void mutate();
              showToast({ title: "Atualizando contratos", description: "Buscando registros mais recentes..." });
            }}
            className="shrink-0"
          >
            <RefreshCw size={14} className="mr-1.5" />
            Atualizar Lista
          </Button>
        </div>

        {/* Filtros Dropdown */}
        <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-surface-border">
          <div className="flex flex-wrap items-center gap-3">
            <label className="flex items-center gap-1.5 text-xs text-muted">
              Tipo de Contrato:
              <select
                value={selectedType}
                onChange={(e) => setSelectedType(e.target.value)}
                className="rounded-md border border-surface-border bg-surface px-2 py-1 text-xs text-foreground"
              >
                <option value={ALL}>Todos os Tipos</option>
                {contractTypes.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </label>

            <label className="flex items-center gap-1.5 text-xs text-muted">
              Status:
              <select
                value={selectedStatus}
                onChange={(e) => setSelectedStatus(e.target.value)}
                className="rounded-md border border-surface-border bg-surface px-2 py-1 text-xs text-foreground"
              >
                <option value={ALL}>Todos os Status</option>
                <option value="NOVO">Contratos Novos</option>
                <option value="ATIVO">Em Execução (Ativos)</option>
                <option value="ENCERRADO">Encerrados</option>
              </select>
            </label>
          </div>

          {hasActiveFilters && (
            <Button variant="ghost" size="sm" onClick={handleResetFilters}>
              Limpar Filtros
            </Button>
          )}
        </div>
      </div>

      {/* Resultados da Pesquisa de Contratos */}
      {error ? (
        <p className="text-sm text-danger">Falha ao carregar contratos públicos de Rondonópolis.</p>
      ) : isLoading && !contracts ? (
        <p className="text-sm text-muted">Carregando contratos públicos...</p>
      ) : filteredContracts.length === 0 ? (
        <EmptyState
          title="Não achamos nenhuma referência"
          description={
            hasActiveFilters
              ? "Nenhum contrato atende aos critérios de busca ou filtros selecionados."
              : "Não há contratos publicados na edição selecionada."
          }
        />
      ) : (
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between text-xs text-muted px-1">
            <span>
              Exibindo <strong>{filteredContracts.length}</strong> contrato(s) municipal(is)
              {selectedEdition !== ALL && !searchTerm.trim() ? ` na Edição Nº ${selectedEdition}` : " em todos os diários pesquisados"}
            </span>
          </div>

          <ul className="flex flex-col gap-4">
            {filteredContracts.map((c) => (
              <li
                key={c.id}
                onClick={() => setSelectedContract(c)}
                className="cursor-pointer flex flex-col gap-3 rounded-lg border border-surface-border bg-surface p-4 shadow-sm hover:border-primary/50 hover:shadow-md transition-all"
              >
                {/* Cabeçalho do Contrato */}
                <div className="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border/60 pb-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-bold text-sm text-foreground hover:text-primary transition-colors">{c.contract_number}</span>
                    <Badge tone={c.status === "NOVO" ? "success" : c.status === "ATIVO" ? "neutral" : "warning"}>
                      {c.status}
                    </Badge>
                    <span className="rounded-full bg-surface-border/40 px-2 py-0.5 text-xs text-muted font-medium">
                      {c.contract_type}
                    </span>
                    {c.portaria_number && (
                      <span className="text-xs text-muted font-medium">
                        · {c.portaria_number}
                      </span>
                    )}
                    {c.edition_number && (
                      <span className="text-xs text-muted">
                        · Edição Nº {c.edition_number}
                      </span>
                    )}
                  </div>
                  <span className="text-xs font-bold text-primary">{c.value}</span>
                </div>

                {/* Objeto do Contrato */}
                <p className="text-xs text-foreground/90 leading-relaxed font-medium">
                  <strong>Objeto:</strong> {c.object}
                </p>

                {/* Empresa Contratada */}
                <div className="flex items-center gap-2 text-xs bg-surface-border/20 p-2 rounded-md border border-surface-border/50">
                  <Building2 size={15} className="text-primary shrink-0" />
                  <span className="font-semibold text-foreground">{c.contractor}</span>
                  <span className="text-muted font-mono ml-auto">CNPJ: {c.contractor_cnpj}</span>
                </div>

                {/* Fiscais e Suplentes do Contrato */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
                  {/* Fiscal Titular */}
                  <div className="flex flex-col gap-1 rounded-md border border-surface-border/70 p-2.5 bg-surface-border/10">
                    <div className="flex items-center gap-1.5 text-xs font-semibold text-foreground">
                      <User size={14} className="text-primary" />
                      Fiscal Titular Designado
                    </div>
                    <span className="text-xs font-bold text-foreground">{c.fiscal_nome}</span>
                    <div className="flex flex-wrap gap-2 text-[11px] text-muted">
                      <span>CPF: {c.fiscal_cpf}</span>
                      <span>·</span>
                      <span>Matrícula: {c.fiscal_matricula}</span>
                    </div>
                  </div>

                  {/* Fiscal Suplente */}
                  <div className="flex flex-col gap-1 rounded-md border border-surface-border/70 p-2.5 bg-surface-border/10">
                    <div className="flex items-center gap-1.5 text-xs font-semibold text-foreground">
                      <User size={14} className="text-warning" />
                      Suplente do Contrato
                    </div>
                    <span className="text-xs font-bold text-foreground">{c.suplente_nome || "Não informado"}</span>
                    <div className="flex flex-wrap gap-2 text-[11px] text-muted">
                      <span>CPF: {c.suplente_cpf || "N/A"}</span>
                      <span>·</span>
                      <span>Matrícula: {c.suplente_matricula || "N/A"}</span>
                    </div>
                  </div>
                </div>

                {/* Rodapé com Link do PDF */}
                <div className="flex items-center justify-between pt-2 border-t border-surface-border/60 text-xs text-muted" onClick={(e) => e.stopPropagation()}>
                  <button
                    onClick={() => setSelectedContract(c)}
                    className="inline-flex items-center gap-1.5 font-medium text-primary hover:underline"
                  >
                    <Eye size={14} />
                    Ver Detalhes do Contrato e Fiscais →
                  </button>
                  <a
                    href={c.doc_url || "https://www.rondonopolis.mt.gov.br/diario-oficial/"}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1.5 font-medium text-muted hover:text-primary transition-colors"
                  >
                    <FileText size={14} />
                    Abrir PDF (Diário Oficial)
                  </a>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Modal de Detalhes do Contrato Público */}
      {selectedContract && (
        <ModalShell
          open
          onClose={() => setSelectedContract(null)}
          size="lg"
          labelledBy="contract-detail-title"
        >
          {/* Cabeçalho do Modal */}
          <div className="flex shrink-0 items-center justify-between border-b border-surface-border bg-surface-border/10 p-4">
              <div className="flex items-center gap-2">
                <ShieldCheck size={20} className="text-primary" />
                <div>
                  <h2 id="contract-detail-title" className="text-base font-bold text-foreground">
                    Extrato de Auditoria de Contrato Público
                  </h2>
                  <p className="text-xs text-muted">Prefeitura Municipal de Rondonópolis - MT</p>
                </div>
              </div>
              <button
                onClick={() => setSelectedContract(null)}
                className="rounded-lg p-1.5 text-muted hover:bg-surface-border hover:text-foreground"
              >
                <X size={18} />
              </button>
            </div>

            {/* Corpo do Modal com Scroll */}
            <div className="flex-1 overflow-y-auto p-5 flex flex-col gap-5">
              {/* Card Principal do Contrato */}
              <div className="flex flex-col gap-3 rounded-lg border border-primary/20 bg-primary/5 p-4">
                <div className="flex flex-wrap items-center justify-between gap-2 border-b border-primary/10 pb-2">
                  <span className="text-lg font-bold text-foreground">{selectedContract.contract_number}</span>
                  <Badge tone={selectedContract.status === "NOVO" ? "success" : selectedContract.status === "ATIVO" ? "neutral" : "warning"}>
                    {selectedContract.status}
                  </Badge>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                  <div>
                    <span className="text-muted block">Tipo de Contrato:</span>
                    <strong className="text-foreground">{selectedContract.contract_type}</strong>
                  </div>
                  <div>
                    <span className="text-muted block">Valor Global Auditado:</span>
                    <strong className="text-primary text-sm">{selectedContract.value}</strong>
                  </div>
                  <div>
                    <span className="text-muted block">Portaria de Designação dos Fiscais:</span>
                    <strong className="font-mono text-foreground">{selectedContract.portaria_number || "N/A"}</strong>
                  </div>
                  <div>
                    <span className="text-muted block">Edição do Diário Oficial:</span>
                    <strong className="font-mono text-foreground">Edição Nº {selectedContract.edition_number || "N/A"}</strong>
                  </div>
                </div>
              </div>

              {/* Partes Envolvidas */}
              <div className="flex flex-col gap-3 rounded-lg border border-surface-border bg-surface p-4">
                <span className="text-xs font-bold text-foreground uppercase tracking-wider flex items-center gap-1.5">
                  <Building2 size={14} className="text-primary" />
                  Empresa Contratada & Documentação
                </span>
                <div className="flex flex-wrap items-center justify-between gap-2 rounded-md bg-surface-border/20 p-3 text-xs border border-surface-border/60">
                  <span className="font-bold text-foreground">{selectedContract.contractor}</span>
                  <span className="font-mono text-muted bg-surface px-2 py-0.5 rounded border border-surface-border">
                    CNPJ: {selectedContract.contractor_cnpj}
                  </span>
                </div>
              </div>

              {/* Equipe de Fiscalização */}
              <div className="flex flex-col gap-3 rounded-lg border border-surface-border bg-surface p-4">
                <span className="text-xs font-bold text-foreground uppercase tracking-wider flex items-center gap-1.5">
                  <User size={14} className="text-primary" />
                  Equipe de Fiscalização do Contrato (Fiscais Titulares & Suplentes)
                </span>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                  <div className="flex flex-col gap-1 rounded-md border border-primary/20 bg-primary/5 p-3">
                    <span className="font-bold text-primary">Fiscal Titular Designado:</span>
                    <strong className="text-foreground text-sm">{selectedContract.fiscal_nome}</strong>
                    <span className="font-mono text-muted">CPF: {selectedContract.fiscal_cpf}</span>
                    <span className="font-mono text-muted">Matrícula: {selectedContract.fiscal_matricula}</span>
                  </div>

                  <div className="flex flex-col gap-1 rounded-md border border-warning/20 bg-warning/5 p-3">
                    <span className="font-bold text-warning">Fiscal Suplente:</span>
                    <strong className="text-foreground text-sm">{selectedContract.suplente_nome || "Não informado"}</strong>
                    <span className="font-mono text-muted">CPF: {selectedContract.suplente_cpf || "N/A"}</span>
                    <span className="font-mono text-muted">Matrícula: {selectedContract.suplente_matricula || "N/A"}</span>
                  </div>
                </div>
              </div>

              {/* Contratos Relacionados ao CPF / CNPJ */}
              {selectedRelatedContracts.length > 1 && (
                <div className="flex flex-col gap-3 rounded-lg border border-surface-border bg-surface p-4">
                  <div className="flex items-center justify-between border-b border-surface-border pb-2">
                    <span className="text-xs font-bold text-foreground uppercase tracking-wider flex items-center gap-1.5">
                      <FileCheck size={14} className="text-primary" />
                      Outros Contratos Relacionados a esta Pessoa / CNPJ
                    </span>
                    <span className="text-xs font-medium px-2 py-0.5 rounded bg-primary/10 text-primary">
                      {selectedRelatedContracts.length} Contratos
                    </span>
                  </div>
                  <div className="flex flex-col gap-2 pt-1">
                    {selectedRelatedContracts.map((rel, idx) => (
                      <div
                        key={idx}
                        className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-surface-border/80 bg-surface-border/10 p-2.5 text-xs"
                      >
                        <div>
                          <strong className="text-foreground">{rel.contract_number}</strong>
                          <span className="text-muted block text-[11px]">{rel.contract_type} · {rel.value}</span>
                        </div>
                        <Badge tone={rel.status === "NOVO" ? "success" : "neutral"} className="text-[10px]">
                          {rel.status}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Objeto do Contrato */}
              <div className="flex flex-col gap-2 rounded-lg border border-surface-border p-4 bg-surface-border/10">
                <span className="text-xs font-bold text-foreground uppercase tracking-wider">
                  Objeto Detalhado do Contrato:
                </span>
                <p className="rounded bg-surface p-3 text-xs italic text-foreground/90 border border-surface-border/60 leading-relaxed">
                  &ldquo;{selectedContract.object}&rdquo;
                </p>
              </div>
            </div>

            {/* Rodapé de Ações do Modal */}
            <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-surface-border bg-surface-border/10 p-4">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => handleCopyContractCertidao(selectedContract)}
              >
                <Copy size={14} className="mr-1.5" />
                Copiar Extrato de Contrato
              </Button>

              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setSelectedContract(null)}
                >
                  Fechar
                </Button>
                <a
                  href={selectedContract.doc_url || "https://www.rondonopolis.mt.gov.br/diario-oficial/"}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-semibold text-white hover:bg-primary-hover shadow-sm transition-all"
                >
                  <ExternalLink size={14} />
                  Abrir Diário Oficial (PDF) →
                </a>
              </div>
            </div>
        </ModalShell>
      )}
    </div>
  );
}
