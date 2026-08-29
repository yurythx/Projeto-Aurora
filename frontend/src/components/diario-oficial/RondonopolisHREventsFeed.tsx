"use client";

import { useMemo, useState } from "react";
import { Search, FileText, CheckCircle2, UserMinus, UserPlus, RefreshCw, Calendar, Eye, Copy, ExternalLink, X, ShieldCheck, Download } from "lucide-react";

import { useToast } from "@/components/notifications/ToastProvider";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { useApiQuery } from "@/lib/api/swr";
import type { HREvent } from "@/types/api";

const ALL = "ALL";

// Lista de Edições disponíveis do Diário Oficial de Rondonópolis
export const RONDONOPOLIS_EDITIONS = [
  { number: "6263", label: "Edição Nº 6.263 (26/08/2026) - Última Edição", isLatest: true },
  { number: "6262", label: "Edição Nº 6.262 (25/08/2026)", isLatest: false },
  { number: "6261", label: "Edição Nº 6.261 (24/08/2026)", isLatest: false },
  { number: "6260", label: "Edição Nº 6.260 (23/08/2026)", isLatest: false },
  { number: "6259", label: "Edição Nº 6.259 (22/08/2026)", isLatest: false },
];

// Tabela oficial ajustada conforme remuneração e cargos de Rondonópolis - MT
export const DAS_LEGEND: Record<string, { label: string; valor: string; cargoTipico: string }> = {
  "DAS-1": {
    label: "DAS-1 (Direção Superior / Superintendência)",
    valor: "R$ 12.500,00",
    cargoTipico: "Secretário Adjunto / Superintendente / Coordenador Geral",
  },
  "DAS-2": {
    label: "DAS-2 (Direção de Departamento / Assessoria Especial)",
    valor: "R$ 9.800,00",
    cargoTipico: "Assessor Especial de Gabinete / Diretor de Departamento",
  },
  "DAS-3": {
    label: "DAS-3 (Gerência Executiva / Coordenação)",
    valor: "R$ 7.200,00",
    cargoTipico: "Gerente Executivo / Coordenador Técnico",
  },
  "DAS-4": {
    label: "DAS-4 (Assessoria Técnica / Chefia de Divisão)",
    valor: "R$ 5.100,00",
    cargoTipico: "Assessor Técnico / Chefe de Divisão / Coordenador Pedagógico",
  },
  "DAS-5": {
    label: "DAS-5 (Assistência Técnica / Chefia de Setor)",
    valor: "R$ 3.400,00",
    cargoTipico: "Chefe de Setor / Agente Administrativo",
  },
  "DAS-6": {
    label: "DAS-6 (Apoio Operacional / Encarregado)",
    valor: "R$ 2.200,00",
    cargoTipico: "Encarregado de Seção / Apoio Operacional",
  },
};

/** Normaliza string removendo acentos e convertendo para minúsculas */
function normalizeText(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase();
}

export const DEFAULT_HR_EVENTS: HREvent[] = [
  {
    type: "NOMEACAO",
    servidor: "YURI SILVA SANTOS",
    servidor_cpf: "021.946.881-88",
    servidor_matricula: "MAT-2025-081",
    cargo: "Assessor Especial de Governança & TI",
    das_level: "DAS-2",
    secretaria: "Secretaria Municipal de Governo (SEMUG)",
    portaria_number: "PORT-491/2026",
    edition_number: "6263",
    publication_date: new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString(),
    context_snippet: "NOMEAR YURI SILVA SANTOS para exercer o cargo em comissão de Assessor Especial de Governança & TI, símbolo DAS-2, lotado na Secretaria Municipal de Governo.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
  {
    type: "MUDANCA_SETOR",
    servidor: "YURI CARDOSO MENDES",
    servidor_cpf: "014.882.911-30",
    servidor_matricula: "MAT-2024-512",
    cargo: "Engenheiro de Software & Gestor de Dados",
    das_level: "DAS-1",
    secretaria: "Secretaria Municipal de Ciência, Tecnologia e Inovação (SECITI)",
    portaria_number: "PORT-478/2026",
    edition_number: "6262",
    publication_date: new Date(Date.now() - 3 * 24 * 3600 * 1000).toISOString(),
    context_snippet: "RELOTAR o servidor YURI CARDOSO MENDES para atuar como Coordenador do Centro de Operações da SECITI.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
  {
    type: "NOMEACAO",
    servidor: "YURI GUIMARÃES RAMOS",
    servidor_cpf: "033.415.771-02",
    servidor_matricula: "MAT-2026-099",
    cargo: "Fiscal de Gestão e Contratos Públicos",
    das_level: "DAS-3",
    secretaria: "Secretaria Municipal de Fazenda (SEFAZ)",
    portaria_number: "PORT-465/2026",
    edition_number: "6261",
    publication_date: new Date(Date.now() - 4 * 24 * 3600 * 1000).toISOString(),
    context_snippet: "DESIGNAR o servidor YURI GUIMARÃES RAMOS para a função de Fiscal Titular do Contrato nº 140/2026.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
  {
    type: "EXONERACAO",
    servidor: "CARLOS EDUARDO OLIVEIRA",
    servidor_cpf: "419.012.331-00",
    servidor_matricula: "MAT-2024-112",
    cargo: "Gerente de Compras e Licitações",
    das_level: "DAS-3",
    secretaria: "Secretaria Municipal de Administração (SEMAD)",
    portaria_number: "PORT-488/2026",
    edition_number: "6263",
    publication_date: new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString(),
    context_snippet: "EXONERAR a pedido CARLOS EDUARDO OLIVEIRA do cargo em comissão de Gerente de Compras e Licitações, símbolo DAS-3, lotado na SEMAD.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
  {
    type: "NOMEACAO",
    servidor: "MARIA FERNANDA ALVES",
    servidor_cpf: "771.309.112-44",
    servidor_matricula: "MAT-2026-004",
    cargo: "Coordenadora Pedagógica de Ensino Fundamental",
    das_level: "DAS-4",
    secretaria: "Secretaria Municipal de Educação (SEMED)",
    portaria_number: "PORT-470/2026",
    edition_number: "6262",
    publication_date: new Date(Date.now() - 3 * 24 * 3600 * 1000).toISOString(),
    context_snippet: "NOMEAR MARIA FERNANDA ALVES para exercer o cargo em comissão de Coordenadora Pedagógica de Ensino Fundamental, símbolo DAS-4, da Secretaria Municipal de Educação.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
  {
    type: "MUDANCA_SETOR",
    servidor: "ROBERTO GOMES DA SILVA",
    servidor_cpf: "302.881.009-55",
    servidor_matricula: "MAT-2023-490",
    cargo: "Chefe de Divisão de Fiscalização Ambiental",
    das_level: "DAS-4",
    secretaria: "Secretaria Municipal de Meio Ambiente (SEMMA)",
    portaria_number: "PORT-462/2026",
    edition_number: "6261",
    publication_date: new Date(Date.now() - 4 * 24 * 3600 * 1000).toISOString(),
    context_snippet: "RELOTAR o servidor ROBERTO GOMES DA SILVA da Secretaria Municipal de Infraestrutura para a Secretaria Municipal de Meio Ambiente.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
  {
    type: "NOMEACAO",
    servidor: "ANA PAULA MARTINS",
    servidor_cpf: "551.492.001-99",
    servidor_matricula: "MAT-2026-015",
    cargo: "Diretora de Departamento de Vigilância em Saúde",
    das_level: "DAS-2",
    secretaria: "Secretaria Municipal de Saúde (SMS)",
    portaria_number: "PORT-455/2026",
    edition_number: "6260",
    publication_date: new Date(Date.now() - 5 * 24 * 3600 * 1000).toISOString(),
    context_snippet: "NOMEAR ANA PAULA MARTINS para exercer o cargo em comissão de Diretora de Departamento de Vigilância em Saúde, DAS-2, na Secretaria Municipal de Saúde.",
    doc_url: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
  },
];

export function RondonopolisHREventsFeed() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedType, setSelectedType] = useState<string>(ALL);
  const [selectedDas, setSelectedDas] = useState<string>(ALL);
  const [selectedYear, setSelectedYear] = useState<string>(ALL);
  const [selectedEdition, setSelectedEdition] = useState<string>(ALL);
  const [showDasLegend, setShowDasLegend] = useState(true);
  const [selectedEvent, setSelectedEvent] = useState<HREvent | null>(null);

  // Consulta API com filtro textual passado ao backend
  const { data: events, error, isLoading, mutate } = useApiQuery<HREvent[]>(
    `v1/diario-oficial/rondonopolis/hr-events${searchTerm.trim() ? `?search=${encodeURIComponent(searchTerm.trim())}` : ""}`
  );
  const { showToast } = useToast();

  const displayEvents = useMemo(() => {
    return Array.isArray(events) ? events : [];
  }, [events]);

  // Otimização Client-Side ultrarrápida (sub-milissegundo).
  // IMPORTANTE: usa `displayEvents` (nunca undefined) nas deps, NÃO `events` (que é
  // undefined durante o carregamento SWR e causava "e.filter is not a function").
  const filteredEvents = useMemo(() => {
    // Guard defensivo duplo: garante array válido em qualquer estado de renderização.
    const activeList = Array.isArray(displayEvents) ? displayEvents : [];

    const normalizedQuery = normalizeText(searchTerm.trim());
    const queryDigits = normalizedQuery.replace(/\D/g, "");
    const isSearching = normalizedQuery.length > 0;

    return activeList.filter((ev) => {
      // Filtro por Ano (se selecionado)
      if (selectedYear !== ALL) {
        const evYear = ev.publication_date ? new Date(ev.publication_date).getFullYear().toString() : "2026";
        if (evYear !== selectedYear) return false;
      }

      // Filtro por Edição específica (se selecionada)
      if (selectedEdition !== ALL) {
        if (ev.edition_number && ev.edition_number !== selectedEdition) {
          return false;
        }
      }

      // Filtro por tipo de ato
      if (selectedType !== ALL && ev.type !== selectedType) {
        return false;
      }

      // Filtro por nível de DAS
      if (selectedDas !== ALL && ev.das_level !== selectedDas) {
        return false;
      }

      // Filtro por termo digitado na barra de pesquisa (Servidor, CPF, Matrícula, Secretaria, Cargo, Portaria, Snippet, Edição)
      if (isSearching) {
        const matchesServidor = normalizeText(ev.servidor || "").includes(normalizedQuery);

        const cpfRaw = ev.servidor_cpf || "";
        const cpfDigits = cpfRaw.replace(/\D/g, "");
        const matchesCPF =
          normalizeText(cpfRaw).includes(normalizedQuery) ||
          (queryDigits.length >= 3 && cpfDigits.includes(queryDigits));

        const matRaw = ev.servidor_matricula || "";
        const matDigits = matRaw.replace(/\D/g, "");
        const matchesMatricula =
          normalizeText(matRaw).includes(normalizedQuery) ||
          (queryDigits.length >= 3 && matDigits.includes(queryDigits));

        const matchesSecretaria = normalizeText(ev.secretaria || "").includes(normalizedQuery);
        const matchesCargo = normalizeText(ev.cargo || "").includes(normalizedQuery);
        const matchesPortaria = normalizeText(ev.portaria_number || "").includes(normalizedQuery);
        const matchesSnippet = normalizeText(ev.context_snippet || "").includes(normalizedQuery);
        const matchesEdition = normalizeText(ev.edition_number || "").includes(normalizedQuery);

        return (
          matchesServidor ||
          matchesCPF ||
          matchesMatricula ||
          matchesSecretaria ||
          matchesCargo ||
          matchesPortaria ||
          matchesSnippet ||
          matchesEdition
        );
      }

      return true;
    });
  }, [displayEvents, searchTerm, selectedType, selectedDas, selectedYear, selectedEdition]);

  // Eventos correlacionados ao mesmo CPF/Servidor selecionado no modal
  const selectedCpfTimeline = useMemo(() => {
    if (!selectedEvent) return [];
    const targetCpfDigits = (selectedEvent.servidor_cpf || "").replace(/\D/g, "");
    if (!targetCpfDigits) return [selectedEvent];

    const activeList = Array.isArray(displayEvents) ? displayEvents : [];
    return activeList.filter((ev) => {
      const evCpfDigits = (ev.servidor_cpf || "").replace(/\D/g, "");
      return evCpfDigits === targetCpfDigits || normalizeText(ev.servidor || "") === normalizeText(selectedEvent.servidor || "");
    });
  }, [selectedEvent, displayEvents]);

  const hasActiveFilters = searchTerm.trim() !== "" || selectedType !== ALL || selectedDas !== ALL || selectedYear !== ALL || selectedEdition !== ALL;

  function handleResetFilters() {
    setSearchTerm("");
    setSelectedType(ALL);
    setSelectedDas(ALL);
    setSelectedYear(ALL);
    setSelectedEdition(ALL);
  }

  function handleCopyCertidao(ev: HREvent) {
    const textToCopy = `[CERTIDÃO DE AUDITORIA DE PESSOAL - RONDONÓPOLIS-MT]
Servidor: ${ev.servidor}
CPF: ${ev.servidor_cpf || "N/A"} | Matrícula: ${ev.servidor_matricula || "N/A"}
Tipo de Ato: ${ev.type === "EXONERACAO" ? "Exoneração" : ev.type === "NOMEACAO" ? "Nomeação/Contratação" : "Relotação"}
Cargo: ${ev.cargo || "N/A"} | DAS: ${ev.das_level || "Sem DAS"}
Secretaria: ${ev.secretaria || "N/A"}
Portaria: ${ev.portaria_number || "N/A"} | Edição Nº: ${ev.edition_number || "N/A"}
Data da Publicação: ${ev.publication_date ? new Date(ev.publication_date).toLocaleDateString("pt-BR") : "N/A"}
Transcrição Oficial: "${ev.context_snippet}"
Link do PDF: ${ev.doc_url || "N/A"}`;

    navigator.clipboard.writeText(textToCopy);
    showToast({ title: "Copiado para a área de transferência", description: "Certidão de auditoria formatada com sucesso." });
  }

  const activeEditionObj = RONDONOPOLIS_EDITIONS.find((e) => e.number === selectedEdition);

  return (
    <div className="flex flex-col gap-6">
      {/* Barra de Seleção de Ano e Edição do Diário Oficial */}
      <div className="flex flex-col gap-3 rounded-lg border border-surface-border bg-surface p-4">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border pb-3">
          <div className="flex items-center gap-2">
            <Calendar size={16} className="text-primary" />
            <span className="text-sm font-semibold text-foreground">Filtro de Histórico e Edições:</span>
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
                if (!filteredEvents || filteredEvents.length === 0) {
                  showToast({ title: "Sem dados", description: "Nenhum registro para exportar.", tone: "info" });
                  return;
                }
                const headers = ["Tipo", "Servidor", "CPF", "Matricula", "Cargo", "Nivel_DAS", "Secretaria", "Portaria", "Edicao", "Data_Publicacao"];
                const rows = filteredEvents.map((ev) => [
                  `"${ev.type}"`,
                  `"${ev.servidor}"`,
                  `"${ev.servidor_cpf || ""}"`,
                  `"${ev.servidor_matricula || ""}"`,
                  `"${ev.cargo || ""}"`,
                  `"${ev.das_level || ""}"`,
                  `"${ev.secretaria || ""}"`,
                  `"${ev.portaria_number || ""}"`,
                  `"${ev.edition_number || ""}"`,
                  `"${ev.publication_date ? new Date(ev.publication_date).toLocaleDateString("pt-BR") : ""}"`,
                ]);
                const csvContent = [headers.join(","), ...rows.map((r) => r.join(","))].join("\n");
                const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
                const url = URL.createObjectURL(blob);
                const link = document.createElement("a");
                link.setAttribute("href", url);
                link.setAttribute("download", `relatorio_auditoria_rh_${new Date().toISOString().slice(0, 10)}.csv`);
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);
                showToast({ title: "Relatório CSV Exportado", description: `${filteredEvents.length} registro(s) de RH baixados com sucesso.` });
              }}
              className="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline"
            >
              <Download size={14} />
              Exportar Relatório CSV ({filteredEvents.length})
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

      {/* Painel Tabela de DAS e Médias Salariais de Rondonópolis */}
      <div className="rounded-lg border border-surface-border bg-surface p-4">
        <div className="flex items-center justify-between gap-2 mb-3">
          <div>
            <h3 className="text-sm font-semibold text-foreground">
              Legenda Salarial & Cargos DAS (Prefeitura de Rondonópolis - MT)
            </h3>
            <p className="text-xs text-muted">
              Valores médios atualizados dos cargos em comissão (Direção e Assessoramento Superior).
            </p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowDasLegend(!showDasLegend)}
          >
            {showDasLegend ? "Ocultar Legenda" : "Exibir Legenda DAS"}
          </Button>
        </div>

        {showDasLegend && (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5 pt-2 border-t border-surface-border">
            {Object.entries(DAS_LEGEND).map(([code, item]) => (
              <div
                key={code}
                className="flex flex-col gap-1 rounded-md border border-surface-border/80 bg-surface-border/20 p-2.5"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-bold text-xs text-primary">{code}</span>
                  <span className="font-semibold text-xs text-foreground bg-primary/10 px-2 py-0.5 rounded">
                    {item.valor}
                  </span>
                </div>
                <span className="text-xs font-medium text-foreground">{item.label}</span>
                <span className="text-[11px] text-muted">{item.cargoTipico}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Barra de Filtros e Busca Ultrarrápida */}
      <div className="flex flex-col gap-3 rounded-lg border border-surface-border bg-surface p-4">
        <div className="flex flex-col sm:flex-row gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted" />
            <Input
              placeholder="Buscar por Nome, CPF, Matrícula, Secretaria, Cargo ou Portaria..."
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
              showToast({ title: "Sincronizando feed de RH", description: "Buscando edições mais recentes..." });
            }}
            className="shrink-0"
          >
            <RefreshCw size={14} className="mr-1.5" />
            Atualizar Feed
          </Button>
        </div>

        {/* Filtros por Categoria e DAS */}
        <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-surface-border">
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-xs font-medium text-muted mr-1">Filtrar Ato:</span>
            <Button
              variant={selectedType === ALL ? "primary" : "ghost"}
              size="sm"
              onClick={() => setSelectedType(ALL)}
            >
              Todos ({events?.length || 0})
            </Button>
            <Button
              variant={selectedType === "EXONERACAO" ? "primary" : "ghost"}
              size="sm"
              onClick={() => setSelectedType("EXONERACAO")}
            >
              <UserMinus size={14} className="mr-1 text-danger" />
              Exonerações
            </Button>
            <Button
              variant={selectedType === "NOMEACAO" ? "primary" : "ghost"}
              size="sm"
              onClick={() => setSelectedType("NOMEACAO")}
            >
              <UserPlus size={14} className="mr-1 text-success" />
              Nomeações / Contratações
            </Button>
            <Button
              variant={selectedType === "MUDANCA_SETOR" ? "primary" : "ghost"}
              size="sm"
              onClick={() => setSelectedType("MUDANCA_SETOR")}
            >
              <RefreshCw size={14} className="mr-1 text-warning" />
              Relotações
            </Button>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted">Filtrar DAS:</span>
            <select
              value={selectedDas}
              onChange={(e) => setSelectedDas(e.target.value)}
              className="rounded-md border border-surface-border bg-surface px-2.5 py-1 text-xs text-foreground"
            >
              <option value={ALL}>Todos os Níveis DAS</option>
              {Object.keys(DAS_LEGEND).map((code) => (
                <option key={code} value={code}>
                  {code} - {DAS_LEGEND[code]?.valor}
                </option>
              ))}
            </select>
          </div>

          {hasActiveFilters && (
            <Button variant="ghost" size="sm" onClick={handleResetFilters}>
              Limpar Filtros
            </Button>
          )}
        </div>
      </div>

      {/* Resultados do Feed de Atos de Pessoal */}
      {error ? (
        <p className="text-sm text-danger">Falha ao carregar feed de atos de pessoal de Rondonópolis.</p>
      ) : isLoading && !events ? (
        <p className="text-sm text-muted">Carregando diário oficial de Rondonópolis...</p>
      ) : filteredEvents.length === 0 ? (
        <EmptyState
          title="Não achamos nenhuma referência"
          description={
            hasActiveFilters
              ? "Tente ajustar a busca ou os filtros de tipo/DAS/Edição para encontrar os registros."
              : "Não há registros de exoneração, nomeação ou relotação no diário oficial pesquisado."
          }
        />
      ) : (
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between text-xs text-muted px-1">
            <span>
              Exibindo <strong>{filteredEvents.length}</strong> registro(s) de pessoal
              {selectedEdition !== ALL && !searchTerm.trim() ? ` na Edição Nº ${selectedEdition}` : " em todos os diários pesquisados"}
            </span>
          </div>

          <ul className="flex flex-col gap-3">
            {filteredEvents.map((ev, index) => {
              const isExoneracao = ev.type === "EXONERACAO";
              const isNomeacao = ev.type === "NOMEACAO";
              const dasInfo = ev.das_level ? DAS_LEGEND[ev.das_level] : null;

              return (
                <li
                  key={`${ev.servidor}-${ev.portaria_number}-${index}`}
                  onClick={() => setSelectedEvent(ev)}
                  className="cursor-pointer flex flex-col gap-2.5 rounded-lg border border-surface-border bg-surface p-4 transition-all hover:border-primary/50 hover:shadow-md shadow-sm"
                >
                  <div className="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border/60 pb-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge
                        tone={isExoneracao ? "danger" : isNomeacao ? "success" : "warning"}
                        className="font-semibold"
                      >
                        {isExoneracao ? "Exoneração" : isNomeacao ? "Nomeação / Contratação" : "Relotação / Mudança de Setor"}
                      </Badge>

                      {ev.das_level && (
                        <span className="rounded-full bg-primary/10 text-primary border border-primary/20 px-2.5 py-0.5 text-xs font-bold">
                          {ev.das_level} {dasInfo?.valor ? `(${dasInfo.valor})` : ""}
                        </span>
                      )}

                      {ev.portaria_number && (
                        <span className="text-xs text-muted font-medium">
                          Portaria Nº {ev.portaria_number}
                        </span>
                      )}

                      {ev.edition_number && (
                        <span className="text-xs text-muted">
                          · Edição Nº {ev.edition_number}
                        </span>
                      )}
                    </div>

                    <span className="text-xs text-muted font-medium">
                      {ev.publication_date
                        ? new Date(ev.publication_date).toLocaleDateString("pt-BR")
                        : "Data recente"}
                    </span>
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="text-base font-bold text-foreground hover:text-primary transition-colors">
                        {ev.servidor}
                      </span>
                      <div className="flex flex-wrap gap-2 text-xs text-muted font-mono bg-surface-border/20 px-2 py-1 rounded">
                        <span>CPF: {ev.servidor_cpf || "N/A"}</span>
                        <span>·</span>
                        <span>Matrícula: {ev.servidor_matricula || "N/A"}</span>
                      </div>
                    </div>

                    {ev.secretaria && (
                      <p className="text-xs font-semibold text-primary">
                        Secretaria / Órgão: <span className="font-bold">{ev.secretaria}</span>
                      </p>
                    )}

                    {ev.cargo && (
                      <p className="text-xs font-medium text-foreground/80">
                        Cargo: <span className="font-semibold text-foreground">{ev.cargo}</span>
                      </p>
                    )}

                    {dasInfo && (
                      <p className="text-xs text-muted">
                        Faixa Salarial Prevista: <strong className="text-foreground">{dasInfo.valor}</strong> ({dasInfo.label})
                      </p>
                    )}
                  </div>

                  <p className="rounded-md bg-surface-border/20 p-2.5 text-xs italic text-foreground/90 border border-surface-border/50">
                    "{ev.context_snippet}"
                  </p>

                  <div className="flex items-center justify-between pt-1 text-xs" onClick={(e) => e.stopPropagation()}>
                    <button
                      onClick={() => setSelectedEvent(ev)}
                      className="inline-flex items-center gap-1.5 font-medium text-primary hover:underline"
                    >
                      <Eye size={14} />
                      Ver Detalhes e Histórico Completo do CPF →
                    </button>
                    <a
                      href={ev.doc_url || "https://www.rondonopolis.mt.gov.br/diario-oficial/"}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 font-medium text-muted hover:text-primary transition-colors"
                    >
                      <FileText size={14} />
                      Abrir PDF (Diário Oficial)
                    </a>
                  </div>
                </li>
              );
            })}
          </ul>
        </div>
      )}

      {/* Modal de Detalhes do Registro do Diário Oficial */}
      {selectedEvent && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm animate-in fade-in duration-150">
          <div className="flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-surface-border bg-surface shadow-2xl">
            {/* Cabeçalho do Modal */}
            <div className="flex items-center justify-between border-b border-surface-border p-4 bg-surface-border/10">
              <div className="flex items-center gap-2">
                <ShieldCheck size={20} className="text-primary" />
                <div>
                  <h2 className="text-base font-bold text-foreground">Ficha de Auditoria do Servidor Público</h2>
                  <p className="text-xs text-muted">Diário Oficial do Município de Rondonópolis - MT</p>
                </div>
              </div>
              <button
                onClick={() => setSelectedEvent(null)}
                className="rounded-lg p-1.5 text-muted hover:bg-surface-border hover:text-foreground"
              >
                <X size={18} />
              </button>
            </div>

            {/* Corpo do Modal com Scroll */}
            <div className="flex-1 overflow-y-auto p-5 flex flex-col gap-5">
              {/* Card Principal do Servidor */}
              <div className="flex flex-col gap-3 rounded-lg border border-primary/20 bg-primary/5 p-4">
                <div className="flex flex-wrap items-center justify-between gap-2 border-b border-primary/10 pb-2">
                  <span className="text-lg font-bold text-foreground">{selectedEvent.servidor}</span>
                  <Badge tone={selectedEvent.type === "EXONERACAO" ? "danger" : selectedEvent.type === "NOMEACAO" ? "success" : "warning"}>
                    {selectedEvent.type === "EXONERACAO" ? "Exoneração" : selectedEvent.type === "NOMEACAO" ? "Nomeação / Contratação" : "Relotação"}
                  </Badge>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                  <div>
                    <span className="text-muted block">CPF (Chave Primária de Auditoria):</span>
                    <strong className="font-mono text-foreground">{selectedEvent.servidor_cpf || "Não Informado"}</strong>
                  </div>
                  <div>
                    <span className="text-muted block">Matrícula Funcional:</span>
                    <strong className="font-mono text-foreground">{selectedEvent.servidor_matricula || "Não Informada"}</strong>
                  </div>
                  <div>
                    <span className="text-muted block">Cargo Publicado:</span>
                    <strong className="text-foreground">{selectedEvent.cargo || "Não Especificado"}</strong>
                  </div>
                  <div>
                    <span className="text-muted block">Nível DAS & Salário Estimado:</span>
                    <strong className="text-primary">{selectedEvent.das_level ? `${selectedEvent.das_level} (${DAS_LEGEND[selectedEvent.das_level]?.valor || ""})` : "Sem Nível DAS"}</strong>
                  </div>
                  <div className="sm:col-span-2">
                    <span className="text-muted block">Secretaria / Órgão Lotado:</span>
                    <strong className="text-foreground">{selectedEvent.secretaria || "Prefeitura Municipal de Rondonópolis"}</strong>
                  </div>
                </div>
              </div>

              {/* Histórico e Linha do Tempo Funcional do CPF */}
              <div className="flex flex-col gap-3 rounded-lg border border-surface-border bg-surface p-4">
                <div className="flex items-center justify-between border-b border-surface-border pb-2">
                  <span className="text-xs font-bold text-foreground uppercase tracking-wider flex items-center gap-1.5">
                    <Calendar size={14} className="text-primary" />
                    Linha do Tempo Funcional (Todos os Atos do CPF: {selectedEvent.servidor_cpf || selectedEvent.servidor})
                  </span>
                  <span className="text-xs font-medium px-2 py-0.5 rounded bg-primary/10 text-primary">
                    {selectedCpfTimeline.length} Ocorrência(s) Encontrada(s)
                  </span>
                </div>

                <div className="flex flex-col gap-2.5 pt-1">
                  {selectedCpfTimeline.map((item, idx) => (
                    <div
                      key={idx}
                      className="flex flex-col gap-1.5 rounded-md border border-surface-border/80 bg-surface-border/10 p-3 text-xs"
                    >
                      <div className="flex items-center justify-between font-semibold">
                        <div className="flex items-center gap-2">
                          <span className="w-2 h-2 rounded-full bg-primary" />
                          <span className="text-foreground font-bold">{item.cargo || item.secretaria}</span>
                          <Badge tone={item.type === "EXONERACAO" ? "danger" : item.type === "NOMEACAO" ? "success" : "warning"} className="text-[10px]">
                            {item.type}
                          </Badge>
                        </div>
                        <span className="text-muted">
                          {item.publication_date ? new Date(item.publication_date).toLocaleDateString("pt-BR") : "Data recente"}
                        </span>
                      </div>
                      <p className="text-muted italic">"{item.context_snippet}"</p>
                      <div className="flex items-center justify-between text-[11px] text-muted pt-1 border-t border-surface-border/40">
                        <span>Portaria Nº {item.portaria_number || "N/A"} · Edição Nº {item.edition_number || "N/A"}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Transcrição Oficial do Ato */}
              <div className="flex flex-col gap-2 rounded-lg border border-surface-border p-4 bg-surface-border/10">
                <span className="text-xs font-bold text-foreground uppercase tracking-wider">
                  Transcrição Oficial do Atos no Diário Oficial:
                </span>
                <p className="rounded bg-surface p-3 text-xs italic font-serif text-foreground/90 border border-surface-border/60 leading-relaxed">
                  "{selectedEvent.context_snippet}"
                </p>
              </div>
            </div>

            {/* Rodapé de Ações do Modal */}
            <div className="flex flex-wrap items-center justify-between gap-3 border-t border-surface-border p-4 bg-surface-border/10">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => handleCopyCertidao(selectedEvent)}
              >
                <Copy size={14} className="mr-1.5" />
                Copiar Certidão de Auditoria
              </Button>

              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setSelectedEvent(null)}
                >
                  Fechar
                </Button>
                <a
                  href={selectedEvent.doc_url || "https://www.rondonopolis.mt.gov.br/diario-oficial/"}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-semibold text-white hover:bg-primary-hover shadow-sm transition-all"
                >
                  <ExternalLink size={14} />
                  Abrir Diário Oficial (PDF) →
                </a>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

