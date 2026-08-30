"use client";

import { useEffect, useState } from "react";
import { Search, Filter, UserCheck, Calendar, Download, Link2 } from "lucide-react";
import { searchPersonnelActs, pdfUrlWithPage, PersonnelAct, TypesenseSearchResponse } from "@/lib/typesense-client";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Card, CardContent } from "@/components/ui/Card";
import { useToast } from "@/components/notifications/ToastProvider";
import {
  readSearchState,
  writeSearchState,
  dateInputToUnix,
  toCSV,
  downloadTextFile,
} from "@/lib/search-tools";

const ACT_TYPES = [
  { value: "", label: "Todos os Atos" },
  { value: "NOMEACAO_EFETIVO", label: "Nomeação (Efetivo)" },
  { value: "NOMEACAO_COMISSIONADO", label: "Nomeação (Comissionado)" },
  { value: "CONTRATACAO_TEMPORARIA", label: "Contratação Temporária" },
  { value: "EXONERACAO", label: "Exoneração" },
  { value: "RESCISAO", label: "Rescisão" },
  { value: "RELOTACAO", label: "Relotação / Transferência" },
  { value: "DESIGNACAO_FUNCAO", label: "Designação de Função" },
];

export default function PessoalSearchPage() {
  const initial = typeof window !== "undefined" ? readSearchState() : {};
  const { showToast } = useToast();
  const [query, setQuery] = useState(initial.q ?? "");
  const [actType, setActType] = useState(initial.act_type ?? "");
  const [secretaria, setSecretaria] = useState(initial.secretaria ?? "");
  const [dasLevel, setDasLevel] = useState(initial.das ?? "");
  const [dateFrom, setDateFrom] = useState(initial.from ?? "");
  const [dateTo, setDateTo] = useState(initial.to ?? "");
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [results, setResults] = useState<TypesenseSearchResponse<PersonnelAct>>({
    found: 0,
    page: 1,
    hits: [],
    facet_counts: [],
  });

  const searchArgs = () => ({
    query,
    actType: actType || undefined,
    secretaria: secretaria || undefined,
    dasLevel: dasLevel || undefined,
    dateFrom: dateInputToUnix(dateFrom),
    dateTo: dateInputToUnix(dateTo, true),
  });

  const handleSearch = async () => {
    setLoading(true);
    writeSearchState({ q: query, act_type: actType, secretaria, das: dasLevel, from: dateFrom, to: dateTo });
    try {
      const res = await searchPersonnelActs({ ...searchArgs(), page: 1, perPage: 25 });
      setResults(res);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleExportCSV = async () => {
    setExporting(true);
    try {
      const res = await searchPersonnelActs({ ...searchArgs(), page: 1, perPage: 250 });
      const rows = res.hits.map((h) => {
        const d = h.document;
        return {
          data: d.publication_date ? new Date(d.publication_date * 1000).toLocaleDateString("pt-BR") : "",
          edicao: d.edition_number,
          tipo_ato: d.act_type,
          servidor: d.person_name,
          cpf: d.person_cpf ?? "",
          matricula: d.person_matricula ?? "",
          cargo: d.job_role ?? "",
          secretaria: d.secretaria ?? "",
          das: d.das_level ?? "",
          salario: d.salary_value ?? "",
          portaria: d.portaria_number ?? "",
          confianca: d.confidence ?? "",
          pdf: pdfUrlWithPage(d.pdf_storage_url, d.pdf_page_number),
        };
      });
      const csv = toCSV(rows, [
        { key: "data", label: "Data" },
        { key: "edicao", label: "Edição" },
        { key: "tipo_ato", label: "Tipo de Ato" },
        { key: "servidor", label: "Servidor(a)" },
        { key: "cpf", label: "CPF" },
        { key: "matricula", label: "Matrícula" },
        { key: "cargo", label: "Cargo" },
        { key: "secretaria", label: "Secretaria" },
        { key: "das", label: "DAS" },
        { key: "salario", label: "Salário" },
        { key: "portaria", label: "Portaria" },
        { key: "confianca", label: "Confiança" },
        { key: "pdf", label: "PDF Oficial" },
      ]);
      downloadTextFile(`atos-pessoal-${new Date().toISOString().slice(0, 10)}.csv`, csv);
      showToast({ title: "CSV exportado", description: `${rows.length} registro(s).`, tone: "success" });
    } catch {
      showToast({ title: "Falha ao exportar CSV", tone: "danger" });
    } finally {
      setExporting(false);
    }
  };

  const copyPermalink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      showToast({ title: "Link copiado", description: "A busca atual está no seu clipboard.", tone: "info" });
    } catch {
      showToast({ title: "Não foi possível copiar o link", tone: "danger" });
    }
  };

  // Busca ao montar (usando o estado vindo da URL) e sempre que um filtro muda.
  useEffect(() => {
    handleSearch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [actType, secretaria, dasLevel, dateFrom, dateTo]);

  // Extrai facetas de secretarias e DAS do Typesense
  const secretariaFacets = results.facet_counts?.find((f) => f.field_name === "secretaria")?.counts || [];
  const dasFacets = results.facet_counts?.find((f) => f.field_name === "das_level")?.counts || [];

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-surface-border pb-4">
        <div>
          <p className="dateline">DIORONDON-E · Atos de pessoal</p>
          <h1 className="mt-2 text-2xl font-semibold text-foreground">
            Inteligência de atos de pessoal
          </h1>
          <p className="mt-1 text-sm text-muted">
            Busca instantânea de nomeações, exonerações, remunerações/DAS e movimentações
            publicadas no Diário Oficial de Rondonópolis.
          </p>
        </div>
      </div>

      {/* Barra de Pesquisa Principal */}
      <div className="flex flex-col sm:flex-row gap-3 rounded-xl border border-surface-border bg-surface p-4 shadow-sm">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-3 h-4 w-4 text-muted pointer-events-none" />
          <Input
            placeholder="Buscar por Nome, CPF, Matrícula, Portaria ou Cargo..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="pl-10 text-sm"
          />
        </div>

        <select
          value={actType}
          onChange={(e) => setActType(e.target.value)}
          className="rounded-md border border-surface-border bg-surface px-3 py-2 text-xs font-medium text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
        >
          {ACT_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>

        <Button onClick={handleSearch} disabled={loading} size="sm">
          {loading ? "Buscando..." : "Pesquisar"}
        </Button>
      </div>

      {/* Grid Principal: Filtros Facetados (Lateral) + Resultados */}
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Painel Lateral de Filtros Facetados */}
        <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-4 shadow-sm h-fit">
          <div className="flex items-center gap-2 border-b border-surface-border pb-3 text-xs font-bold text-foreground">
            <Filter size={14} className="text-primary" />
            <span>Filtros Facetados</span>
          </div>

          {/* Filtro por Secretaria */}
          <div>
            <label className="block text-[11px] font-semibold text-muted uppercase tracking-wider mb-2">
              Secretaria / Órgão
            </label>
            <select
              value={secretaria}
              onChange={(e) => setSecretaria(e.target.value)}
              className="w-full rounded-md border border-surface-border bg-surface p-2 text-xs text-foreground"
            >
              <option value="">Todas as Secretarias</option>
              {secretariaFacets.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.value} ({s.count})
                </option>
              ))}
            </select>
          </div>

          {/* Filtro por Nível DAS */}
          <div>
            <label className="block text-[11px] font-semibold text-muted uppercase tracking-wider mb-2">
              Nível Cargo / DAS
            </label>
            <select
              value={dasLevel}
              onChange={(e) => setDasLevel(e.target.value)}
              className="w-full rounded-md border border-surface-border bg-surface p-2 text-xs text-foreground"
            >
              <option value="">Todos os Níveis</option>
              {dasFacets.map((d) => (
                <option key={d.value} value={d.value}>
                  {d.value} ({d.count})
                </option>
              ))}
            </select>
          </div>

          {/* Filtro por período de publicação */}
          <div>
            <label className="block text-[11px] font-semibold text-muted uppercase tracking-wider mb-2">
              Período de publicação
            </label>
            <div className="flex flex-col gap-2">
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="w-full rounded-md border border-surface-border bg-surface p-2 text-xs text-foreground"
                aria-label="Data inicial"
              />
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="w-full rounded-md border border-surface-border bg-surface p-2 text-xs text-foreground"
                aria-label="Data final"
              />
            </div>
          </div>

          {(actType || secretaria || dasLevel || dateFrom || dateTo) && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setActType("");
                setSecretaria("");
                setDasLevel("");
                setDateFrom("");
                setDateTo("");
              }}
              className="mt-2 text-xs text-danger"
            >
              Limpar Filtros
            </Button>
          )}
        </div>

        {/* Lista de Resultados de Atos */}
        <div className="lg:col-span-3 flex flex-col gap-4">
          <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
            <span>
              Exibindo <strong>{results.hits.length}</strong> de <strong>{results.found}</strong> registros encontrados
            </span>
            <div className="flex gap-2">
              <Button variant="ghost" size="sm" onClick={copyPermalink} className="text-xs inline-flex items-center gap-1.5">
                <Link2 size={13} />
                Copiar link
              </Button>
              <Button
                variant="secondary"
                size="sm"
                onClick={handleExportCSV}
                disabled={exporting || results.found === 0}
                className="text-xs inline-flex items-center gap-1.5"
              >
                <Download size={13} />
                {exporting ? "Exportando…" : "Exportar CSV"}
              </Button>
            </div>
          </div>

          {results.hits.length === 0 ? (
            <div className="rounded-xl border border-dashed border-surface-border bg-surface p-12 text-center text-muted">
              <UserCheck size={32} className="mx-auto text-muted/40 mb-3" />
              <p className="font-semibold text-foreground">Nenhum ato de pessoal encontrado com os termos pesquisados.</p>
              <p className="text-xs mt-1">Tente ajustar as palavras-chave ou remover os filtros de secretaria/tipo de ato.</p>
            </div>
          ) : (
            results.hits.map((hit, idx) => {
              const doc = hit.document;
              const isExoneracao = doc.act_type === "EXONERACAO" || doc.act_type === "RESCISAO";

              return (
                <Card key={doc.id || idx} className="hover:border-primary/50 transition-colors">
                  <CardContent className="p-5 flex flex-col gap-3">
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-surface-border/60 pb-3">
                      <div className="flex items-center gap-2">
                        <Badge tone={isExoneracao ? "danger" : "success"}>
                          {doc.act_type.replace("_", " ")}
                        </Badge>
                        {doc.das_level && <Badge tone="info">{doc.das_level}</Badge>}
                        {doc.confidence === "medium" && (
                          <Badge tone="warning" title="Extração automática sem CPF/matrícula — conferir na fonte">
                            conferir
                          </Badge>
                        )}
                        {doc.portaria_number && (
                          <span className="text-xs font-mono text-muted">Portaria: {doc.portaria_number}</span>
                        )}
                      </div>

                      <div className="flex items-center gap-2 text-xs text-muted font-mono">
                        <Calendar size={12} />
                        <span>Edição #{doc.edition_number}</span>
                      </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                      <div>
                        <span className="text-muted block text-[10px] uppercase font-semibold">Servidor(a)</span>
                        <strong className="text-sm font-bold text-foreground">{doc.person_name}</strong>
                        {doc.person_matricula && (
                          <span className="block text-muted font-mono mt-0.5">Matrícula: {doc.person_matricula}</span>
                        )}
                        {doc.person_cpf && (
                          <span className="block text-muted font-mono">CPF: {doc.person_cpf}</span>
                        )}
                      </div>

                      <div>
                        <span className="text-muted block text-[10px] uppercase font-semibold">Cargo / Secretaria</span>
                        <strong className="text-foreground">{doc.job_role || "Não informado"}</strong>
                        <span className="block text-muted mt-0.5">{doc.secretaria || "Prefeitura Municipal"}</span>
                        {doc.salary_value ? (
                          <span className="block text-emerald-600 dark:text-emerald-400 font-semibold font-mono mt-1">
                            R$ {doc.salary_value.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
                          </span>
                        ) : null}
                      </div>
                    </div>

                    {/* Trecho destacado (Snippet) */}
                    <div className="rounded-md bg-surface-hover/50 p-3 text-xs text-muted font-sans border-l-2 border-primary">
                      <p className="line-clamp-3">&ldquo;{doc.full_act_text}&rdquo;</p>
                    </div>

                    {doc.pdf_storage_url && (
                      <div className="flex justify-end pt-1">
                        <a
                          href={pdfUrlWithPage(doc.pdf_storage_url, doc.pdf_page_number)}
                          target="_blank"
                          rel="noreferrer"
                          className="text-xs text-primary hover:underline font-medium inline-flex items-center gap-1"
                        >
                          {doc.pdf_page_number > 1
                            ? `Ver no PDF oficial (pág. ${doc.pdf_page_number}) →`
                            : "Ver PDF da Edição Oficial →"}
                        </a>
                      </div>
                    )}
                  </CardContent>
                </Card>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}
